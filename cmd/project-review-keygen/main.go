// project-review-keygen creates an offline Ed25519 reviewer key and registry entry.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var reviewerIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type registryDocument struct {
	Reviewers []registryEntry `json:"reviewers"`
}

type registryEntry struct {
	ReviewerID string `json:"reviewer_id"`
	PublicKey  string `json:"public_key"`
}

type createdFile struct {
	path string
	info os.FileInfo
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("project-review-keygen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	reviewerID := flags.String("reviewer-id", "", "stable operator-assigned reviewer key ID")
	privateOut := flags.String("private-out", "", "new path for the raw 64-byte Ed25519 private key")
	registryOut := flags.String("registry-out", "", "new path for the reviewer registry JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *reviewerID == "" || *privateOut == "" || *registryOut == "" {
		return errors.New("usage: project-review-keygen --reviewer-id ID --private-out PATH --registry-out PATH")
	}
	if !reviewerIDPattern.MatchString(*reviewerID) {
		return errors.New("reviewer-id must be 1..128 ASCII letters, digits, dot, underscore, colon, or hyphen; the first character must be alphanumeric")
	}
	if filepath.Clean(*privateOut) == filepath.Clean(*registryOut) {
		return errors.New("private-out and registry-out must be different paths")
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate Ed25519 key pair: %w", err)
	}
	defer clear(privateKey)
	registry, err := json.MarshalIndent(registryDocument{Reviewers: []registryEntry{{ReviewerID: *reviewerID, PublicKey: base64.StdEncoding.EncodeToString(publicKey)}}}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode public registry: %w", err)
	}
	registry = append(registry, '\n')

	created := make([]createdFile, 0, 2)
	cleanup := func() {
		for i := len(created) - 1; i >= 0; i-- {
			removeIfSameFile(created[i])
		}
	}
	privateInfo, err := writeExclusive(*privateOut, privateKey)
	if err != nil {
		return fmt.Errorf("create private key file: %w", err)
	}
	created = append(created, createdFile{path: *privateOut, info: privateInfo})
	registryInfo, err := writeExclusive(*registryOut, registry)
	if err != nil {
		cleanup()
		return fmt.Errorf("create reviewer registry file: %w", err)
	}
	created = append(created, createdFile{path: *registryOut, info: registryInfo})
	if err := syncDirectory(filepath.Dir(*privateOut)); err != nil {
		cleanup()
		return fmt.Errorf("sync private key directory: %w", err)
	}
	if filepath.Clean(filepath.Dir(*registryOut)) != filepath.Clean(filepath.Dir(*privateOut)) {
		if err := syncDirectory(filepath.Dir(*registryOut)); err != nil {
			cleanup()
			return fmt.Errorf("sync registry directory: %w", err)
		}
	}
	// Key material is written only to the private output. Keep stdout and stderr
	// free of key bytes so this command is safe to invoke from automation.
	_ = stdout
	return nil
}

func writeExclusive(path string, contents []byte) (os.FileInfo, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	info, statErr := f.Stat()
	if statErr != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, statErr
	}
	if !info.Mode().IsRegular() {
		_ = f.Close()
		removeIfSameFile(createdFile{path: path, info: info})
		return nil, errors.New("output must be a regular file")
	}
	if info.Mode().Perm() != 0600 {
		_ = f.Close()
		removeIfSameFile(createdFile{path: path, info: info})
		return nil, errors.New("output file mode is not 0600")
	}
	if _, err = f.Write(contents); err != nil {
		_ = f.Close()
		removeIfSameFile(createdFile{path: path, info: info})
		return nil, err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		removeIfSameFile(createdFile{path: path, info: info})
		return nil, err
	}
	if err = f.Close(); err != nil {
		removeIfSameFile(createdFile{path: path, info: info})
		return nil, err
	}
	return info, nil
}

func removeIfSameFile(file createdFile) {
	if file.info == nil {
		return
	}
	info, err := os.Lstat(file.path)
	if err == nil && os.SameFile(file.info, info) {
		_ = os.Remove(file.path)
	}
}

func syncDirectory(path string) error {
	if path == "" {
		path = "."
	}
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
