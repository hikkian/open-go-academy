// Package credentialrender produces local, self-issued learning certificate
// artifacts. It does not verify signatures or assert the holder's identity.
package credentialrender

import (
	"bytes"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gpdf-dev/gpdf"
	"github.com/gpdf-dev/gpdf/document"
	"github.com/gpdf-dev/gpdf/pdf"
	"github.com/gpdf-dev/gpdf/qrcode"
	"github.com/gpdf-dev/gpdf/template"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"academy/internal/credentials"
)

//go:embed assets/DejaVuSans.ttf
var assetFS embed.FS

const (
	canvasWidth  = 1400
	canvasHeight = 1000
	fontFamily   = "AcademyDejaVu"
)

var (
	idPattern  = regexp.MustCompile(`^[a-f0-9]{32}$`)
	keyPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
	fontOnce   sync.Once
	fontData   []byte
	fontErr    error
)

var (
	ink        = color.RGBA{R: 27, G: 48, B: 66, A: 255}
	muted      = color.RGBA{R: 87, G: 108, B: 124, A: 255}
	blue       = color.RGBA{R: 25, G: 83, B: 126, A: 255}
	teal       = color.RGBA{R: 26, G: 139, B: 143, A: 255}
	paper      = color.RGBA{R: 250, G: 251, B: 249, A: 255}
	line       = color.RGBA{R: 198, G: 211, B: 215, A: 255}
	revokedRed = color.RGBA{R: 178, G: 49, B: 57, A: 255}
)

// PDF renders one extractable-text A4 landscape certificate page.
func PDF(e credentials.Envelope, revoked bool) ([]byte, error) {
	if err := validate(e); err != nil {
		return nil, err
	}
	fontBytes, err := embeddedFont()
	if err != nil {
		return nil, err
	}
	doc := gpdf.NewDocument(
		gpdf.WithPageSize(document.Size{Width: 841.89, Height: 595.28}),
		gpdf.WithMargins(document.Edges{Top: document.Mm(13), Right: document.Mm(18), Bottom: document.Mm(13), Left: document.Mm(18)}),
		gpdf.WithFont(fontFamily, fontBytes),
		gpdf.WithDefaultFont(fontFamily, 11),
		gpdf.WithMetadata(document.DocumentMetadata{Title: "Go Academy — учебный сертификат", Author: e.Payload.IssuerName, Subject: "Локальный учебный результат", Creator: "Open Go Academy"}),
	)
	page := doc.AddPage()
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(12, func(c *template.ColBuilder) {
			c.Text("OPEN GO ACADEMY", template.FontSize(10), template.Bold(), template.TextColor(pdf.RGBHex(0x197F83)), template.AlignCenter())
			c.Spacer(document.Mm(5))
			c.Text("УЧЕБНЫЙ СЕРТИФИКАТ", template.FontSize(25), template.Bold(), template.TextColor(pdf.RGBHex(0x1B3042)), template.AlignCenter())
			c.Spacer(document.Mm(2))
			c.Text("Go Academy · результат обучения", template.FontSize(11), template.TextColor(pdf.RGBHex(0x576C7C)), template.AlignCenter())
			c.Spacer(document.Mm(7))
		})
	})
	if revoked {
		page.AutoRow(func(r *template.RowBuilder) {
			r.Col(12, func(c *template.ColBuilder) {
				c.Text("ОТОЗВАН", template.FontSize(19), template.Bold(), template.TextColor(pdf.RGBHex(0xB23139)), template.AlignCenter(), template.BgColor(pdf.RGBHex(0xFBE9E9)), template.TextPadding(document.UniformEdges(document.Mm(2))))
				c.Spacer(document.Mm(3))
			})
		})
	}
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(12, func(c *template.ColBuilder) {
			c.Text("Этот документ относится к учебному профилю", template.FontSize(11), template.TextColor(pdf.RGBHex(0x576C7C)), template.AlignCenter())
			c.Spacer(document.Mm(2))
			c.Text(e.Payload.SubjectName, template.FontSize(25), template.Bold(), template.TextColor(pdf.RGBHex(0x1B3042)), template.AlignCenter())
			c.Spacer(document.Mm(3))
			c.Text("подтвердил учебные критерии Go Academy для ранга", template.FontSize(12), template.TextColor(pdf.RGBHex(0x576C7C)), template.AlignCenter())
			c.Spacer(document.Mm(2))
			c.Text(e.Payload.Rank, template.FontSize(21), template.Bold(), template.TextColor(pdf.RGBHex(0x197F83)), template.AlignCenter())
			c.Spacer(document.Mm(7))
		})
	})
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(12, func(c *template.ColBuilder) {
			c.Text("КРИТЕРИИ И КОМПЕТЕНЦИИ", template.FontSize(9), template.Bold(), template.TextColor(pdf.RGBHex(0x197F83)))
			c.Spacer(document.Mm(2))
			for _, competency := range e.Payload.Competencies {
				c.Text("•  "+competency, template.FontSize(10), template.TextColor(pdf.RGBHex(0x1B3042)))
				c.Spacer(document.Mm(1))
			}
			c.Spacer(document.Mm(4))
		})
	})
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(6, func(c *template.ColBuilder) {
			c.Text("Экзамен", template.FontSize(8), template.TextColor(pdf.RGBHex(0x576C7C)))
			c.Text(e.Payload.ExamID, template.FontSize(10), template.Bold())
			c.Spacer(document.Mm(2))
			c.Text("Версия критериев", template.FontSize(8), template.TextColor(pdf.RGBHex(0x576C7C)))
			c.Text(e.Payload.CriteriaVersion, template.FontSize(10), template.Bold())
			c.Spacer(document.Mm(2))
			c.Text("Дата выпуска (UTC)", template.FontSize(8), template.TextColor(pdf.RGBHex(0x576C7C)))
			c.Text(issueDate(e.Payload.IssuedAt), template.FontSize(10), template.Bold())
		})
		r.Col(6, func(c *template.ColBuilder) {
			c.Text("Идентификатор документа", template.FontSize(8), template.TextColor(pdf.RGBHex(0x576C7C)))
			c.Text(e.Payload.ID, template.FontSize(9))
			c.Spacer(document.Mm(2))
			c.Text("Самостоятельно настроенный локальный эмитент", template.FontSize(8), template.TextColor(pdf.RGBHex(0x576C7C)))
			c.Text(e.Payload.IssuerName, template.FontSize(10), template.Bold())
			c.Text("Отпечаток ключа эмитента (SHA-256)", template.FontSize(8), template.TextColor(pdf.RGBHex(0x576C7C)))
			c.Text(e.KeyID, template.FontSize(8))
		})
	})
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(12, func(c *template.ColBuilder) {
			c.Spacer(document.Mm(4))
			c.Text("Проверка подписи и статуса", template.FontSize(9), template.Bold(), template.TextColor(pdf.RGBHex(0x197F83)))
			c.Text(e.Payload.VerifyURL, template.FontSize(8), template.TextColor(pdf.RGBHex(0x1B537E)))
			c.QRCode(e.Payload.VerifyURL, template.QRSize(document.Mm(28)))
			c.Spacer(document.Mm(3))
			c.Text("Самостоятельно выпущенное локальное подтверждение учебных критериев Go Academy. Оно не удостоверяет личность, трудоустройство, годы опыта или аккредитацию.", template.FontSize(8), template.TextColor(pdf.RGBHex(0x576C7C)))
		})
	})
	return doc.Generate()
}

// PNG renders a complete 1400×1000 landscape certificate, including its QR code.
func PNG(e credentials.Envelope, revoked bool) ([]byte, error) {
	if err := validate(e); err != nil {
		return nil, err
	}
	fontBytes, err := embeddedFont()
	if err != nil {
		return nil, err
	}
	parsed, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, errors.New("parse embedded certificate font")
	}
	canvas := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: paper}, image.Point{}, draw.Src)
	drawFrame(canvas)
	drawLabel(canvas, parsed, "OPEN GO ACADEMY", 700, 110, 17, teal, true, 1200)
	drawLabel(canvas, parsed, "УЧЕБНЫЙ СЕРТИФИКАТ", 660, 178, 49, ink, true, 820)
	drawLabel(canvas, parsed, "Go Academy · результат обучения", 700, 215, 19, muted, false, 1150)
	if revoked {
		drawBand(canvas, image.Rect(120, 240, 1280, 292), color.RGBA{R: 251, G: 233, B: 233, A: 255})
		drawLabel(canvas, parsed, "ОТОЗВАН", 700, 277, 28, revokedRed, true, 1100)
	}
	startY := 310
	if revoked {
		startY = 338
	}
	drawLabel(canvas, parsed, "Этот документ относится к учебному профилю", 700, startY, 18, muted, false, 1180)
	nameY := drawWrappedCentered(canvas, parsed, e.Payload.SubjectName, 700, startY+64, 44, ink, true, 1120, 53, 3)
	drawLabel(canvas, parsed, "подтвердил учебные критерии Go Academy для ранга", 700, nameY+36, 18, muted, false, 1180)
	drawLabel(canvas, parsed, e.Payload.Rank, 700, nameY+91, 38, teal, true, 1000)
	contentY := nameY + 148
	drawLeftLabel(canvas, parsed, "КРИТЕРИИ И КОМПЕТЕНЦИИ", 120, contentY, 15, teal, true, 620)
	y := contentY + 36
	for _, competency := range e.Payload.Competencies {
		y = drawWrapped(canvas, parsed, "•  "+competency, 135, y, 14, ink, false, 520, 18, 2) + 8
	}
	metadataY := max(contentY, 600)
	drawLine(canvas, image.Pt(120, metadataY), image.Pt(1280, metadataY), line)
	drawCompactField(canvas, parsed, 700, metadataY+22, "Экзамен", e.Payload.ExamID, 14, 540)
	drawCompactField(canvas, parsed, 700, metadataY+64, "Версия критериев", e.Payload.CriteriaVersion, 14, 540)
	drawCompactField(canvas, parsed, 700, metadataY+106, "Дата выпуска (UTC)", issueDate(e.Payload.IssuedAt), 14, 540)
	drawCompactField(canvas, parsed, 700, metadataY+148, "Идентификатор документа", e.Payload.ID, 13, 540)
	drawCompactField(canvas, parsed, 700, metadataY+190, "Самостоятельно настроенный локальный эмитент", e.Payload.IssuerName, 13, 540)
	drawCompactField(canvas, parsed, 700, metadataY+232, "Отпечаток ключа эмитента (SHA-256)", e.KeyID, 9, 540)
	drawLeftLabel(canvas, parsed, "URL проверки", 700, metadataY+274, 10, muted, false, 540)
	drawWrapped(canvas, parsed, e.Payload.VerifyURL, 700, metadataY+290, 9, blue, false, 540, 12, 2)
	qrBytes, err := QRPNG(e)
	if err != nil {
		return nil, err
	}
	qr, err := png.Decode(bytes.NewReader(qrBytes))
	if err != nil {
		return nil, errors.New("decode generated verification QR")
	}
	qrRect := image.Rect(1100, 54, 1100+qr.Bounds().Dx(), 54+qr.Bounds().Dy())
	draw.Draw(canvas, qrRect, qr, qr.Bounds().Min, draw.Over)
	drawLabel(canvas, parsed, "Проверить документ", qrRect.Min.X+qrRect.Dx()/2, qrRect.Max.Y+22, 12, teal, true, qrRect.Dx())
	drawWrapped(canvas, parsed, "Самостоятельно выпущенное локальное подтверждение критериев Go Academy. Не удостоверяет личность, трудоустройство, годы опыта или аккредитацию.", 120, 952, 10, muted, false, 1140, 13, 1)
	var output bytes.Buffer
	if err = png.Encode(&output, canvas); err != nil {
		return nil, errors.New("encode certificate PNG")
	}
	return output.Bytes(), nil
}

// QRPNG returns a PNG containing only the verification URL QR code.
func QRPNG(e credentials.Envelope) ([]byte, error) {
	if err := validate(e); err != nil {
		return nil, err
	}
	code, err := qrcode.Encode(e.Payload.VerifyURL, qrcode.LevelM)
	if err != nil {
		return nil, fmt.Errorf("encode verification QR: %w", err)
	}
	scale := 220 / (code.Size() + 8)
	if scale < 1 {
		scale = 1
	}
	if scale > 6 {
		scale = 6
	}
	return code.PNG(scale)
}

func validate(e credentials.Envelope) error {
	p := e.Payload
	signature, signatureErr := base64.RawURLEncoding.DecodeString(e.Signature)
	if signatureErr != nil || len(signature) != 64 || p.Format != credentials.Format || !keyPattern.MatchString(e.KeyID) ||
		!idPattern.MatchString(p.ID) || !idPattern.MatchString(p.AttemptID) ||
		!boundedText(p.SubjectName, 160) || !boundedText(p.IssuerName, 160) ||
		!boundedText(p.Rank, 48) || !boundedText(p.ExamID, 64) || !boundedText(p.CriteriaVersion, 40) ||
		!boundedText(p.VerifyURL, 512) || p.IssuerID != "local:"+e.KeyID || !keyPattern.MatchString(p.CriteriaHash) ||
		len(p.Competencies) == 0 || len(p.Competencies) > 8 {
		return errors.New("invalid certificate envelope")
	}
	if _, err := time.Parse(time.RFC3339, p.IssuedAt); err != nil {
		return errors.New("invalid certificate issue date")
	}
	parsed, err := url.ParseRequestURI(p.VerifyURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil {
		return errors.New("invalid certificate verification URL")
	}
	for _, value := range append([]string{p.ID, p.IssuerID, p.CriteriaHash}, p.Competencies...) {
		if !utf8.ValidString(value) || strings.TrimSpace(value) == "" || len(value) > 128 || hasControl(value) {
			return errors.New("invalid certificate text")
		}
	}
	return nil
}

func boundedText(s string, maxBytes int) bool {
	return utf8.ValidString(s) && strings.TrimSpace(s) != "" && len(s) <= maxBytes && !hasControl(s)
}

func hasControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return true
		}
	}
	return false
}

func embeddedFont() ([]byte, error) {
	fontOnce.Do(func() {
		fontData, fontErr = assetFS.ReadFile("assets/DejaVuSans.ttf")
		if fontErr == nil {
			_, fontErr = opentype.Parse(fontData)
		}
		if fontErr != nil {
			fontErr = errors.New("load embedded certificate font")
		}
	})
	return fontData, fontErr
}

func issueDate(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return t.UTC().Format("02.01.2006")
}

func drawFrame(img *image.RGBA) {
	drawRect(img, image.Rect(28, 28, canvasWidth-28, 38), teal)
	drawRect(img, image.Rect(28, canvasHeight-38, canvasWidth-28, canvasHeight-28), teal)
	drawRect(img, image.Rect(28, 38, 38, canvasHeight-38), line)
	drawRect(img, image.Rect(canvasWidth-38, 38, canvasWidth-28, canvasHeight-38), line)
	drawLine(img, image.Pt(100, 242), image.Pt(1300, 242), line)
}

func drawCompactField(img *image.RGBA, parsed *opentype.Font, x, y int, label, value string, size, maxWidth int) {
	drawLeftLabel(img, parsed, label, x, y, 11, muted, false, maxWidth)
	drawWrapped(img, parsed, value, x, y+17, size, ink, true, maxWidth, size+3, 2)
}

func drawLeftLabel(img *image.RGBA, parsed *opentype.Font, text string, x, baseline, size int, color color.RGBA, bold bool, maxWidth int) {
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(size), DPI: 96, Hinting: font.HintingFull})
	if err != nil {
		return
	}
	defer face.Close()
	text = sanitize(text)
	if maxWidth > 0 && font.MeasureString(face, text).Ceil() > maxWidth {
		drawWrapped(img, parsed, text, x, baseline, size, color, bold, maxWidth, size+4, 1)
		return
	}
	d := font.Drawer{Dst: img, Src: image.NewUniform(color), Face: face, Dot: fixed.P(x, baseline)}
	d.DrawString(text)
	if bold {
		d.Dot = fixed.P(x+1, baseline)
		d.DrawString(text)
	}
}

func drawLabel(img *image.RGBA, parsed *opentype.Font, text string, centerX, baseline, size int, color color.RGBA, bold bool, maxWidth int) {
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(size), DPI: 96, Hinting: font.HintingFull})
	if err != nil {
		return
	}
	defer face.Close()
	text = sanitize(text)
	w := font.MeasureString(face, text).Ceil()
	if maxWidth > 0 && w > maxWidth {
		drawWrapped(img, parsed, text, centerX-maxWidth/2, baseline, size, color, bold, maxWidth, size+6, 1)
		return
	}
	x := centerX - w/2
	point := fixed.P(x, baseline)
	drawer := font.Drawer{Dst: img, Src: image.NewUniform(color), Face: face, Dot: point}
	drawer.DrawString(text)
	if bold {
		drawer.Dot = fixed.P(x+1, baseline)
		drawer.DrawString(text)
	}
}

func drawWrapped(img *image.RGBA, parsed *opentype.Font, text string, x, baseline, size int, color color.RGBA, bold bool, maxWidth, lineHeight, maxLines int) int {
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(size), DPI: 96, Hinting: font.HintingFull})
	if err != nil {
		return baseline
	}
	defer face.Close()
	lines := wrap(face, sanitize(text), maxWidth, maxLines)
	y := baseline
	for _, lineText := range lines {
		d := font.Drawer{Dst: img, Src: image.NewUniform(color), Face: face, Dot: fixed.P(x, y)}
		d.DrawString(lineText)
		if bold {
			d.Dot = fixed.P(x+1, y)
			d.DrawString(lineText)
		}
		y += lineHeight
	}
	return y
}

func drawWrappedCentered(img *image.RGBA, parsed *opentype.Font, text string, centerX, baseline, size int, color color.RGBA, bold bool, maxWidth, lineHeight, maxLines int) int {
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(size), DPI: 96, Hinting: font.HintingFull})
	if err != nil {
		return baseline
	}
	defer face.Close()
	lines := wrap(face, sanitize(text), maxWidth, maxLines)
	y := baseline
	for _, lineText := range lines {
		width := font.MeasureString(face, lineText).Ceil()
		drawWrapped(img, parsed, lineText, centerX-width/2, y, size, color, bold, maxWidth, lineHeight, 1)
		y += lineHeight
	}
	return y
}

func wrap(face font.Face, text string, maxWidth, maxLines int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	lineText := ""
	truncated := false
	appendLine := func(value string) {
		if len(lines) < maxLines {
			lines = append(lines, value)
		} else {
			truncated = true
		}
	}
	for _, word := range words {
		candidate := word
		if lineText != "" {
			candidate = lineText + " " + word
		}
		if font.MeasureString(face, candidate).Ceil() <= maxWidth {
			lineText = candidate
			continue
		}
		if lineText != "" {
			appendLine(lineText)
			lineText = ""
		}
		if font.MeasureString(face, word).Ceil() <= maxWidth {
			lineText = word
			continue
		}
		lineText = breakLongWord(face, word, maxWidth, appendLine)
	}
	if lineText != "" {
		appendLine(lineText)
	}
	if truncated && len(lines) > 0 {
		last := lines[len(lines)-1]
		for font.MeasureString(face, last+"…").Ceil() > maxWidth && len([]rune(last)) > 1 {
			last = string([]rune(last)[:len([]rune(last))-1])
		}
		lines[len(lines)-1] = last + "…"
	}
	return lines
}

func breakLongWord(face font.Face, word string, maxWidth int, appendLine func(string)) string {
	runes := []rune(word)
	var current string
	for _, r := range runes {
		candidate := current + string(r)
		if current != "" && font.MeasureString(face, candidate).Ceil() > maxWidth {
			appendLine(current)
			current = string(r)
		} else {
			current = candidate
		}
	}
	return current
}

func sanitize(text string) string {
	var result strings.Builder
	for _, r := range text {
		if unicode.IsControl(r) {
			if r == '\n' || r == '\t' {
				result.WriteRune(' ')
			}
			continue
		}
		result.WriteRune(r)
	}
	return strings.Join(strings.Fields(result.String()), " ")
}

func drawBand(img *image.RGBA, bounds image.Rectangle, fill color.RGBA) {
	drawRect(img, bounds, fill)
}

func drawRect(img *image.RGBA, bounds image.Rectangle, fill color.RGBA) {
	draw.Draw(img, bounds.Intersect(img.Bounds()), &image.Uniform{C: fill}, image.Point{}, draw.Src)
}

func drawLine(img *image.RGBA, start, end image.Point, c color.RGBA) {
	if start.Y == end.Y {
		drawRect(img, image.Rect(start.X, start.Y, end.X, end.Y+1), c)
		return
	}
	if start.X == end.X {
		drawRect(img, image.Rect(start.X, start.Y, end.X+1, end.Y), c)
	}
}
