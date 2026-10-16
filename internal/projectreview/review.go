// Package projectreview verifies offline reviewer attestations. A reviewer ID
// authenticates possession of an operator-pinned key; it does not establish a
// real-world person's identity or prove independence from a learner.
package projectreview

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
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const Version = 1
const domain = "open-go-academy/project-review/v1\x00"

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Attestation struct {
	Version        int    `json:"version"`
	ReviewerID     string `json:"reviewer_id"`
	SubmissionID   string `json:"submission_id"`
	EvaluationID   string `json:"evaluation_id"`
	ProjectID      string `json:"project_id"`
	ProjectVersion string `json:"project_version"`
	ArtifactHash   string `json:"artifact_hash"`
	RubricVersion  string `json:"rubric_version"`
	RubricHash     string `json:"rubric_hash"`
	Decision       string `json:"decision"`
	Comments       string `json:"comments"`
	IssuedAt       string `json:"issued_at"`
	Signature      string `json:"signature"`
}

// Registry contains only operator-configured reviewer public keys.
type Registry struct{ keys map[string]ed25519.PublicKey }

type registryFile struct {
	Reviewers []registryEntry `json:"reviewers"`
}
type registryEntry struct {
	ReviewerID string `json:"reviewer_id"`
	PublicKey  string `json:"public_key"`
}

// LoadRegistry decodes an operator-owned JSON allowlist. Each public key is
// standard-base64 encoded Ed25519 public key bytes.
func LoadRegistry(r io.Reader) (*Registry, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var file registryFile
	if err := dec.Decode(&file); err != nil {
		return nil, fmt.Errorf("decode reviewer registry: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, errors.New("reviewer registry must contain exactly one JSON value")
	}
	if len(file.Reviewers) == 0 || len(file.Reviewers) > 10000 {
		return nil, errors.New("reviewer registry must contain 1..10000 reviewers")
	}
	reg := &Registry{keys: make(map[string]ed25519.PublicKey, len(file.Reviewers))}
	for _, entry := range file.Reviewers {
		if !validID(entry.ReviewerID) {
			return nil, fmt.Errorf("invalid reviewer_id %q", entry.ReviewerID)
		}
		if _, exists := reg.keys[entry.ReviewerID]; exists {
			return nil, fmt.Errorf("duplicate reviewer_id %q", entry.ReviewerID)
		}
		key, err := base64.StdEncoding.DecodeString(entry.PublicKey)
		if err != nil || len(key) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("reviewer %q has invalid Ed25519 public_key", entry.ReviewerID)
		}
		reg.keys[entry.ReviewerID] = append(ed25519.PublicKey(nil), key...)
	}
	return reg, nil
}

func validID(s string) bool   { return idPattern.MatchString(s) }
func validHash(s string) bool { return hashPattern.MatchString(s) }

func validate(a Attestation, requireSignature bool) error {
	if a.Version != Version {
		return fmt.Errorf("unsupported attestation version %d", a.Version)
	}
	for label, value := range map[string]string{"reviewer_id": a.ReviewerID, "submission_id": a.SubmissionID, "evaluation_id": a.EvaluationID, "project_id": a.ProjectID, "project_version": a.ProjectVersion, "rubric_version": a.RubricVersion} {
		if !validID(value) {
			return fmt.Errorf("invalid %s", label)
		}
	}
	if !validHash(a.ArtifactHash) || !validHash(a.RubricHash) {
		return errors.New("artifact_hash and rubric_hash must be lowercase SHA-256 hex")
	}
	switch a.Decision {
	case "approve", "changes_requested", "reject":
	default:
		return errors.New("invalid decision")
	}
	if !utf8.ValidString(a.Comments) || len(a.Comments) > 4000 || strings.IndexFunc(a.Comments, func(r rune) bool { return r < 0x20 && r != '\n' && r != '\r' && r != '\t' || r == 0x7f }) >= 0 {
		return errors.New("comments must be valid UTF-8, at most 4000 bytes, without control characters")
	}
	t, err := time.Parse(time.RFC3339, a.IssuedAt)
	if err != nil || t.Format(time.RFC3339) != a.IssuedAt {
		return errors.New("issued_at must be canonical RFC3339")
	}
	if requireSignature {
		sig, err := base64.StdEncoding.DecodeString(a.Signature)
		if err != nil || len(sig) != ed25519.SignatureSize || base64.StdEncoding.EncodeToString(sig) != a.Signature {
			return errors.New("signature must be base64 Ed25519 signature")
		}
	} else if a.Signature != "" {
		return errors.New("signature must be empty before signing")
	}
	return nil
}

type canonicalPayload struct {
	Version        int    `json:"version"`
	ReviewerID     string `json:"reviewer_id"`
	SubmissionID   string `json:"submission_id"`
	EvaluationID   string `json:"evaluation_id"`
	ProjectID      string `json:"project_id"`
	ProjectVersion string `json:"project_version"`
	ArtifactHash   string `json:"artifact_hash"`
	RubricVersion  string `json:"rubric_version"`
	RubricHash     string `json:"rubric_hash"`
	Decision       string `json:"decision"`
	Comments       string `json:"comments"`
	IssuedAt       string `json:"issued_at"`
}

func canonical(a Attestation) ([]byte, error) {
	if err := validate(a, false); err != nil {
		return nil, err
	}
	body, err := json.Marshal(canonicalPayload{a.Version, a.ReviewerID, a.SubmissionID, a.EvaluationID, a.ProjectID, a.ProjectVersion, a.ArtifactHash, a.RubricVersion, a.RubricHash, a.Decision, a.Comments, a.IssuedAt})
	if err != nil {
		return nil, err
	}
	return append([]byte(domain), body...), nil
}

// Sign signs an unsigned attestation using an Ed25519 private key. It returns
// a copy with a base64 signature; the caller owns and protects the key bytes.
func Sign(a Attestation, privateKey ed25519.PrivateKey) (Attestation, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return Attestation{}, errors.New("invalid Ed25519 private key size")
	}
	if a.Signature != "" {
		return Attestation{}, errors.New("attestation is already signed")
	}
	message, err := canonical(a)
	if err != nil {
		return Attestation{}, err
	}
	a.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, message))
	return a, nil
}

// Verify checks schema, allowlist membership and signature over all bound fields.
func Verify(registry *Registry, a Attestation) error {
	if registry == nil {
		return errors.New("reviewer registry is required")
	}
	if err := validate(a, true); err != nil {
		return err
	}
	key, ok := registry.keys[a.ReviewerID]
	if !ok {
		return fmt.Errorf("unknown reviewer %q", a.ReviewerID)
	}
	message, err := canonical(Attestation{Version: a.Version, ReviewerID: a.ReviewerID, SubmissionID: a.SubmissionID, EvaluationID: a.EvaluationID, ProjectID: a.ProjectID, ProjectVersion: a.ProjectVersion, ArtifactHash: a.ArtifactHash, RubricVersion: a.RubricVersion, RubricHash: a.RubricHash, Decision: a.Decision, Comments: a.Comments, IssuedAt: a.IssuedAt})
	if err != nil {
		return err
	}
	sig, _ := base64.StdEncoding.DecodeString(a.Signature)
	if !ed25519.Verify(key, message, sig) {
		return errors.New("invalid reviewer signature")
	}
	return nil
}

// Hash returns the lowercase SHA-256 digest of bytes, useful for artifact and rubric binding.
func Hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// GenerateKeyPair is provided for offline operator tooling and tests.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}
