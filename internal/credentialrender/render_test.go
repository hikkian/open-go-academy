package credentialrender

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"academy/internal/credentials"
)

func testEnvelope(t *testing.T) credentials.Envelope {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("a", 32)
	criteriaHash := strings.Repeat("c", 64)
	envelope, err := credentials.Sign(key, credentials.Payload{
		Format: credentials.Format, ID: id, IssuerName: "Локальная академия", SubjectName: "Алия Ермекова",
		Rank: "Zero", ExamID: "zero-core", CriteriaVersion: "1.0.0", CriteriaHash: criteriaHash,
		AttemptID: strings.Repeat("b", 32), IssuedAt: time.Now().UTC().Format(time.RFC3339Nano), VerifyURL: "http://127.0.0.1:8380/verify/" + id,
		Competencies: []string{"Понимает назначение исходного кода", "Объявляет и использует переменные", "Пишет простые функции на Go", "Запускает проверку и исправляет ошибки"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}

func TestPDFPNGAndQRRender(t *testing.T) {
	envelope := testEnvelope(t)
	pdfBytes, err := PDF(envelope, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) || len(pdfBytes) < 1500 {
		t.Fatalf("invalid PDF output: %d bytes, prefix %q", len(pdfBytes), pdfBytes[:min(len(pdfBytes), 12)])
	}
	if !bytes.Contains(pdfBytes, []byte("/FontFile2")) {
		t.Fatal("PDF does not embed the trusted Unicode font")
	}
	if !bytes.Contains(pdfBytes, []byte("/ToUnicode")) {
		t.Fatal("PDF has no Unicode character map for selectable Cyrillic text")
	}

	fullPNG, err := PNG(envelope, false)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(fullPNG))
	if err != nil {
		t.Fatalf("decode certificate PNG: %v", err)
	}
	if img.Bounds().Dx() != canvasWidth || img.Bounds().Dy() != canvasHeight {
		t.Fatalf("certificate dimensions = %v, want %dx%d", img.Bounds(), canvasWidth, canvasHeight)
	}
	if !hasMultipleColors(img, img.Bounds()) {
		t.Fatal("certificate PNG appears blank")
	}

	qrPNG, err := QRPNG(envelope)
	if err != nil {
		t.Fatal(err)
	}
	qr, err := png.Decode(bytes.NewReader(qrPNG))
	if err != nil {
		t.Fatalf("decode QR PNG: %v", err)
	}
	if qr.Bounds().Dx() < 50 || qr.Bounds().Dy() < 50 || qr.Bounds().Dx() != qr.Bounds().Dy() {
		t.Fatalf("unexpected QR dimensions: %v", qr.Bounds())
	}
	if !hasMultipleColors(qr, qr.Bounds()) {
		t.Fatal("QR PNG appears blank")
	}
}

func TestLongTextWrapsAndRevocationIsVisible(t *testing.T) {
	envelope := testEnvelope(t)
	envelope.Payload.SubjectName = strings.Repeat("Очень длинное имя владельца сертификата ", 2)
	envelope.Payload.Competencies = []string{
		strings.Repeat("Go source code is checked by compiler before execution; ", 2),
		"Понимает основы переменных, типов и функций в Go.",
		"Использует условия и циклы для команд программы.",
		"Читает ошибки компилятора и исправляет учебную программу на Go.",
	}
	if !utf8.ValidString(envelope.Payload.SubjectName) {
		t.Fatal("test subject name must be UTF-8")
	}
	if _, err := PDF(envelope, true); err != nil {
		t.Fatalf("render long-text revoked PDF: %v", err)
	}
	encoded, err := PNG(envelope, true)
	if err != nil {
		t.Fatalf("render long-text revoked PNG: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if !containsColor(img, img.Bounds(), revokedRed) {
		t.Fatal("revoked certificate has no visible revocation mark")
	}
}

func TestRejectsInvalidEnvelopeAndURL(t *testing.T) {
	envelope := testEnvelope(t)
	bad := envelope
	bad.Payload.VerifyURL = "javascript:alert(1)"
	if _, err := PDF(bad, false); err == nil {
		t.Fatal("PDF accepted non-HTTP verification URL")
	}
	if _, err := PNG(bad, false); err == nil {
		t.Fatal("PNG accepted non-HTTP verification URL")
	}
	if _, err := QRPNG(bad); err == nil {
		t.Fatal("QR accepted non-HTTP verification URL")
	}
	bad = envelope
	bad.KeyID = hex.EncodeToString([]byte("short"))
	if _, err := QRPNG(bad); err == nil {
		t.Fatal("renderer accepted malformed issuer fingerprint")
	}
}

func hasMultipleColors(img interface {
	Bounds() image.Rectangle
	At(int, int) color.Color
}, bounds image.Rectangle) bool {
	first := img.At(bounds.Min.X, bounds.Min.Y)
	for y := bounds.Min.Y; y < bounds.Max.Y; y += max(1, bounds.Dy()/80) {
		for x := bounds.Min.X; x < bounds.Max.X; x += max(1, bounds.Dx()/80) {
			if img.At(x, y) != first {
				return true
			}
		}
	}
	return false
}

func containsColor(img interface {
	Bounds() image.Rectangle
	At(int, int) color.Color
}, bounds image.Rectangle, target color.RGBA) bool {
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if r, g, b, _ := img.At(x, y).RGBA(); uint8(r>>8) == target.R && uint8(g>>8) == target.G && uint8(b>>8) == target.B {
				return true
			}
		}
	}
	return false
}
