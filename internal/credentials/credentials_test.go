package credentials

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testPayload(key ed25519.PrivateKey) Payload {
	return Payload{
		ID: strings.Repeat("a", 32), IssuerName: "Open Go Academy — локальный выпускатель",
		SubjectName: "Айгерим Өмірбек", Rank: "Zero", ExamID: "zero-core",
		CriteriaVersion: "1.0.0", CriteriaHash: strings.Repeat("c", 64), AttemptID: strings.Repeat("b", 32),
		IssuedAt: time.Now().UTC().Format(time.RFC3339Nano), VerifyURL: "http://127.0.0.1:8380/verify/" + strings.Repeat("a", 32),
		Competencies: []string{"Основы Go", "Чтение ошибок компилятора"},
	}
}

func TestSignVerifyAndTamperEveryPayloadField(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payload := testPayload(private)
	envelope, err := Sign(private, payload)
	if err != nil {
		t.Fatal(err)
	}
	if envelope.Payload.Format != Format || envelope.Payload.IssuerID != "local:"+KeyID(public) || envelope.KeyID != KeyID(public) {
		t.Fatalf("unexpected signed envelope: %+v", envelope)
	}
	if err := Verify(public, envelope); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*Payload){
		"format":           func(p *Payload) { p.Format = "oga-other" },
		"id":               func(p *Payload) { p.ID = strings.Repeat("d", 32) },
		"issuer id":        func(p *Payload) { p.IssuerID = "local:" + strings.Repeat("f", 64) },
		"issuer name":      func(p *Payload) { p.IssuerName = "Изменённое имя" },
		"subject":          func(p *Payload) { p.SubjectName = "Другое имя" },
		"rank":             func(p *Payload) { p.Rank = "Principal" },
		"exam":             func(p *Payload) { p.ExamID = "other-exam" },
		"criteria version": func(p *Payload) { p.CriteriaVersion = "2.0.0" },
		"criteria hash":    func(p *Payload) { p.CriteriaHash = strings.Repeat("d", 64) },
		"attempt":          func(p *Payload) { p.AttemptID = strings.Repeat("e", 32) },
		"date":             func(p *Payload) { p.IssuedAt = "2020-01-01T00:00:00Z" },
		"URL":              func(p *Payload) { p.VerifyURL = "http://127.0.0.1:8380/verify/" + strings.Repeat("d", 32) },
		"competencies":     func(p *Payload) { p.Competencies[0] = "Другая компетенция" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			altered := envelope
			altered.Payload.Competencies = append([]string(nil), envelope.Payload.Competencies...)
			mutate(&altered.Payload)
			if err := Verify(public, altered); err == nil {
				t.Fatal("tampered payload verified")
			}
		})
	}
	altered := envelope
	altered.Signature = strings.Repeat("A", len(envelope.Signature))
	if Verify(public, altered) == nil {
		t.Fatal("tampered signature verified")
	}
	altered = envelope
	altered.KeyID = strings.Repeat("f", 64)
	if Verify(public, altered) == nil {
		t.Fatal("tampered envelope key id verified")
	}
	_, otherKey, _ := ed25519.GenerateKey(rand.Reader)
	if Verify(otherKey.Public().(ed25519.PublicKey), envelope) == nil {
		t.Fatal("wrong pinned public key verified")
	}
}

func TestPayloadValidationAndUnicode(t *testing.T) {
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	p := testPayload(private)
	if _, err := Sign(private, p); err != nil {
		t.Fatalf("unicode names should be accepted: %v", err)
	}
	cases := []struct {
		name   string
		change func(*Payload)
	}{
		{"unknown rank", func(p *Payload) { p.Rank = "Guru" }},
		{"bad hash", func(p *Payload) { p.CriteriaHash = "not-a-hash" }},
		{"bad date", func(p *Payload) { p.IssuedAt = "tomorrow" }},
		{"far future", func(p *Payload) { p.IssuedAt = time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339Nano) }},
		{"credential URL host", func(p *Payload) { p.VerifyURL = "http://localhost:8380/verify/" + p.ID }},
		{"credential URL path", func(p *Payload) { p.VerifyURL = "http://127.0.0.1:8380/other/" + p.ID }},
		{"credential URL query", func(p *Payload) { p.VerifyURL = "http://127.0.0.1:8380/verify/" + p.ID + "?x=1" }},
		{"credential URL credentials", func(p *Payload) { p.VerifyURL = "http://u:p@127.0.0.1:8380/verify/" + p.ID }},
		{"duplicate competency", func(p *Payload) { p.Competencies = []string{"A", "A"} }},
		{"invalid UTF-8", func(p *Payload) { p.SubjectName = string([]byte{0xff}) }},
		{"oversized name", func(p *Payload) { p.SubjectName = strings.Repeat("я", 161) }},
		{"missing issuer fingerprint", func(p *Payload) { p.IssuerID = "local:someone-else" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := p
			tc.change(&q)
			if _, err := Sign(private, q); err == nil {
				t.Fatal("invalid payload accepted")
			}
		})
	}
	if len(public) != ed25519.PublicKeySize {
		t.Fatal("bad fixture key")
	}
	sum := sha256.Sum256(public)
	if KeyID(public) != hex.EncodeToString(sum[:]) {
		t.Fatal("key ID is not SHA-256 public-key fingerprint")
	}
}

func TestLoadOrCreateKeySafePermissionsAndIntegrity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "signing.key")
	key, err := LoadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("new key mode %o", info.Mode().Perm())
	}
	loaded, err := LoadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(key, loaded) {
		t.Fatal("key changed across reload")
	}
	if KeyID(key.Public().(ed25519.PublicKey)) == "" {
		t.Fatal("missing key fingerprint")
	}
	bad := filepath.Join(dir, "bad.key")
	if err := os.WriteFile(bad, key, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreateKey(bad); err == nil {
		t.Fatal("insecure existing mode accepted")
	}
	if err := os.Chmod(bad, 0600); err != nil {
		t.Fatal(err)
	}
	data := append([]byte(nil), key...)
	data[len(data)-1] ^= 1
	if err := os.WriteFile(bad, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreateKey(bad); err == nil {
		t.Fatal("mismatched key public half accepted")
	}
	target := filepath.Join(dir, "target.key")
	if err := os.WriteFile(target, key, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.key")
	if err := os.Symlink(target, link); err == nil {
		if _, e := LoadOrCreateKey(link); e == nil {
			t.Fatal("symlink accepted")
		}
	} else {
		t.Skipf("symlink unavailable: %v", err)
	}
	nonregular := filepath.Join(dir, "directory.key")
	if err := os.Mkdir(nonregular, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreateKey(nonregular); err == nil {
		t.Fatal("directory accepted as key")
	}
}

func TestConcurrentKeyInitializationPublishesOneCompleteKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issuer.key")
	const workers = 24
	keys := make(chan ed25519.PrivateKey, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key, err := LoadOrCreateKey(path)
			if err != nil {
				errs <- err
				return
			}
			keys <- key
		}()
	}
	wg.Wait()
	close(keys)
	close(errs)
	for err := range errs {
		t.Errorf("concurrent initialization failed: %v", err)
	}
	var first ed25519.PrivateKey
	for key := range keys {
		if first == nil {
			first = key
			continue
		}
		if !equalBytes(first, key) {
			t.Fatal("concurrent initializers returned different issuer keys")
		}
	}
	if len(first) != ed25519.PrivateKeySize {
		t.Fatal("no complete key was published")
	}
}
