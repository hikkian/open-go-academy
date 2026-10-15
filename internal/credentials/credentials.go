// Package credentials implements local, self-issued learning credentials.
// These signatures do not establish a person's identity or an external issuer's trust.
package credentials

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const Format = "oga-local-v1"
const signingDomain = "open-go-academy:credential:v1\x00"
const maxPayloadBytes = 16 << 10

var (
	ErrInvalid     = errors.New("invalid local credential")
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?(?:\+[A-Za-z0-9.-]+)?$`)
	examIDPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	idPattern      = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

// Payload is the complete, fixed-shape data covered by the Ed25519 signature.
type Payload struct {
	Format          string   `json:"format"`
	ID              string   `json:"id"`
	IssuerID        string   `json:"issuer_id"`
	IssuerName      string   `json:"issuer_name"`
	SubjectName     string   `json:"subject_name"`
	Rank            string   `json:"rank"`
	ExamID          string   `json:"exam_id"`
	CriteriaVersion string   `json:"criteria_version"`
	CriteriaHash    string   `json:"criteria_hash"`
	AttemptID       string   `json:"attempt_id"`
	IssuedAt        string   `json:"issued_at"`
	VerifyURL       string   `json:"verify_url"`
	Competencies    []string `json:"competencies"`
}

// Envelope deliberately omits an embedded public key. Verification must pin
// the trusted issuer key supplied by the verifier.
type Envelope struct {
	Payload   Payload `json:"payload"`
	KeyID     string  `json:"key_id"`
	Signature string  `json:"signature"`
}

// LoadOrCreateKey loads a private Ed25519 key or creates a new one atomically.
// Existing symlinks, non-regular files, group/world-readable files, and malformed
// keys are rejected. The raw 64-byte ed25519.PrivateKey format is used.
func LoadOrCreateKey(path string) (ed25519.PrivateKey, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("credential key path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("prepare credential key directory: %w", err)
	}
	if k, err := loadKey(path); err == nil {
		return k, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, errors.New("generate credential signing key")
	}
	// Build the complete private file under an exclusive random name, sync it,
	// then publish it with a same-directory hard link. Link creation is atomic
	// and refuses to replace an existing path, so readers never see a partial key.
	f, err := os.CreateTemp(filepath.Dir(path), ".oga-credential-key-*")
	if err != nil {
		return nil, errors.New("create temporary credential key")
	}
	tempPath := f.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(private)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, errors.New("write credential signing key")
	}
	if err = os.Link(tempPath, path); errors.Is(err, os.ErrExist) {
		return loadKey(path)
	} else if err != nil {
		return nil, errors.New("publish credential signing key")
	}
	if dir, e := os.Open(filepath.Dir(path)); e == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return append(ed25519.PrivateKey(nil), private...), nil
}

func loadKey(path string) (ed25519.PrivateKey, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, errors.New("credential key must be a regular, non-symlink file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("open credential key")
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, errors.New("credential key changed while opening")
	}
	if after.Mode().Perm()&0077 != 0 {
		return nil, errors.New("credential key permissions must be 0600 or stricter")
	}
	data, err := io.ReadAll(f)
	if err != nil || len(data) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid credential key file")
	}
	derived := ed25519.NewKeyFromSeed(data[:ed25519.SeedSize])
	if !equalBytes(derived, data) {
		return nil, errors.New("credential key public half does not match its seed")
	}
	return ed25519.PrivateKey(append([]byte(nil), data...)), nil
}

// KeyID is the lowercase hexadecimal SHA-256 fingerprint of the public key.
func KeyID(public ed25519.PublicKey) string {
	if len(public) != ed25519.PublicKeySize {
		return ""
	}
	sum := sha256.Sum256(public)
	return hex.EncodeToString(sum[:])
}

// Sign validates and signs a local educational credential with a domain-separated
// canonical JSON representation of the fixed Payload struct.
func Sign(key ed25519.PrivateKey, payload Payload) (Envelope, error) {
	if len(key) != ed25519.PrivateKeySize || !equalBytes(ed25519.NewKeyFromSeed(key[:ed25519.SeedSize]), key) {
		return Envelope{}, ErrInvalid
	}
	if payload.Format == "" {
		payload.Format = Format
	}
	if payload.IssuerID == "" {
		payload.IssuerID = "local:" + KeyID(key.Public().(ed25519.PublicKey))
	}
	if err := validatePayload(payload, key.Public().(ed25519.PublicKey)); err != nil {
		return Envelope{}, err
	}
	canonical, err := json.Marshal(payload)
	if err != nil || len(canonical) > maxPayloadBytes {
		return Envelope{}, ErrInvalid
	}
	signed := append([]byte(signingDomain), canonical...)
	sig := ed25519.Sign(key, signed)
	return Envelope{Payload: payload, KeyID: KeyID(key.Public().(ed25519.PublicKey)), Signature: base64.RawURLEncoding.EncodeToString(sig)}, nil
}

// Verify validates the fixed payload and signature against a caller-pinned key.
// The key ID included in the envelope is only a fingerprint, never a trust root.
func Verify(pinned ed25519.PublicKey, envelope Envelope) error {
	if len(pinned) != ed25519.PublicKeySize || envelope.KeyID != KeyID(pinned) {
		return ErrInvalid
	}
	if err := validatePayload(envelope.Payload, pinned); err != nil {
		return err
	}
	signature, err := base64.RawURLEncoding.DecodeString(envelope.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize || base64.RawURLEncoding.EncodeToString(signature) != envelope.Signature {
		return ErrInvalid
	}
	canonical, err := json.Marshal(envelope.Payload)
	if err != nil || len(canonical) > maxPayloadBytes {
		return ErrInvalid
	}
	signed := append([]byte(signingDomain), canonical...)
	if !ed25519.Verify(pinned, signed, signature) {
		return ErrInvalid
	}
	return nil
}

func validatePayload(p Payload, public ed25519.PublicKey) error {
	if p.Format != Format || !idPattern.MatchString(p.ID) || !idPattern.MatchString(p.AttemptID) ||
		p.IssuerID != "local:"+KeyID(public) || !validText(p.IssuerName, 160) || !validText(p.SubjectName, 160) ||
		!validRank(p.Rank) || !examIDPattern.MatchString(p.ExamID) || !versionPattern.MatchString(p.CriteriaVersion) ||
		!validHash(p.CriteriaHash) || !validTimestamp(p.IssuedAt) || !validVerifyURL(p.VerifyURL, p.ID) ||
		len(p.Competencies) == 0 || len(p.Competencies) > 32 {
		return ErrInvalid
	}
	for i, c := range p.Competencies {
		if !validText(c, 128) {
			return ErrInvalid
		}
		for _, previous := range p.Competencies[:i] {
			if previous == c {
				return ErrInvalid
			}
		}
	}
	data, err := json.Marshal(p)
	if err != nil || len(data) > maxPayloadBytes {
		return ErrInvalid
	}
	return nil
}

func validText(s string, max int) bool {
	if strings.TrimSpace(s) == "" || len(s) > max || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func validRank(rank string) bool {
	switch rank {
	case "Zero", "Junior", "Junior+", "Middle", "Middle+", "Senior":
		return true
	default:
		return false
	}
}

func validHash(s string) bool {
	if len(s) != sha256.Size*2 {
		return false
	}
	b, err := hex.DecodeString(s)
	return err == nil && hex.EncodeToString(b) == s
}

func validTimestamp(s string) bool {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil || t.Format(time.RFC3339Nano) != s {
		return false
	}
	return !t.After(time.Now().Add(time.Minute))
}

func validVerifyURL(raw, id string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.Path != "/verify/"+id || (u.RawPath != "" && u.EscapedPath() != u.Path) {
		return false
	}
	if u.Hostname() == "" {
		return false
	}
	if port := u.Port(); port != "" {
		n, e := strconv.Atoi(port)
		if e != nil || n < 1 || n > 65535 {
			return false
		}
	}
	host := u.Hostname()
	if net.ParseIP(host) == nil {
		return false
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.Zone() == "" && ip.IsLoopback()
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
