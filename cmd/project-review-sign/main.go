// project-review-sign signs one JSON review attestation with an offline key.
package main

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"academy/internal/projectreview"
)

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("project-review-sign", flag.ContinueOnError)
	flags.SetOutput(stderr)
	keyPath := flags.String("key", "", "path to raw 64-byte Ed25519 private key (must have mode 0600)")
	inPath := flags.String("in", "-", "unsigned attestation JSON path, or - for stdin")
	outPath := flags.String("out", "-", "signed attestation JSON path, or - for stdout")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *keyPath == "" {
		return errors.New("usage: project-review-sign --key private.key [--in unsigned.json] [--out signed.json]")
	}
	info, err := os.Lstat(*keyPath)
	if err != nil {
		return fmt.Errorf("inspect key file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("key path must be a regular file, not a symlink or device")
	}
	if info.Mode().Perm() != 0600 {
		return errors.New("private key file must have mode 0600")
	}
	key, err := os.ReadFile(*keyPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	if len(key) != ed25519.PrivateKeySize {
		clear(key)
		return errors.New("private key file must contain exactly 64 raw Ed25519 private-key bytes")
	}
	defer clear(key)
	var in io.Reader = stdin
	if *inPath != "-" {
		f, err := os.Open(*inPath)
		if err != nil {
			return fmt.Errorf("open attestation: %w", err)
		}
		defer f.Close()
		in = f
	}
	dec := json.NewDecoder(in)
	dec.DisallowUnknownFields()
	var att projectreview.Attestation
	if err := dec.Decode(&att); err != nil {
		return fmt.Errorf("decode attestation: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("attestation input must contain exactly one JSON value")
	}
	signed, err := projectreview.Sign(att, ed25519.PrivateKey(key))
	if err != nil {
		return fmt.Errorf("sign attestation: %w", err)
	}
	data, err := json.MarshalIndent(signed, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if *outPath == "-" {
		_, err = stdout.Write(data)
		return err
	}
	// Exclusive create prevents accidental replacement of a previously signed record.
	f, err := os.OpenFile(*outPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create signed attestation: %w", err)
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		os.Remove(*outPath)
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		os.Remove(*outPath)
		return err
	}
	if err = f.Close(); err != nil {
		os.Remove(*outPath)
		return err
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
