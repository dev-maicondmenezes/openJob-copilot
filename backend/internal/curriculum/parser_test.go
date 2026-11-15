package curriculum

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestParseMarkdown(t *testing.T) {
	md := `# João Silva
Engenheiro de Software

Experiência Profissional
• Desenvolveu APIs REST e microsserviços em Go
• Implementou CI/CD com GitHub Actions

Habilidades
* Go, Docker, SQLite, Kubernetes
`
	out, err := Parse("curriculo.md", strings.NewReader(md))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "## Experiência") {
		t.Errorf("expected section '## Experiência', got:\n%s", out)
	}
	if !strings.Contains(out, "- Desenvolveu APIs REST e microsserviços em Go") {
		t.Errorf("expected normalized bullet point, got:\n%s", out)
	}
	if !strings.Contains(out, "## Habilidades") {
		t.Errorf("expected section '## Habilidades', got:\n%s", out)
	}
}

func TestParsePlainText(t *testing.T) {
	txt := `Maria Souza
Resumo
Desenvolvedora frontend com foco em Svelte e TypeScript.

Educação
Bacharelado em Ciência da Computação - USP
`
	out, err := Parse("resume.txt", strings.NewReader(txt))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "## Resumo") {
		t.Errorf("expected '## Resumo', got:\n%s", out)
	}
	if !strings.Contains(out, "## Educação") {
		t.Errorf("expected '## Educação', got:\n%s", out)
	}
}

func TestParseValidation(t *testing.T) {
	if _, err := Parse("doc.docx", strings.NewReader("content")); err != ErrUnsupportedFormat {
		t.Errorf("expected ErrUnsupportedFormat, got %v", err)
	}

	if _, err := Parse("empty.md", strings.NewReader("   \n\t  ")); err != ErrEmptyFile {
		t.Errorf("expected ErrEmptyFile, got %v", err)
	}
}

func buildTestPDF(text string) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, 6)

	offsets[1] = buf.Len()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	offsets[2] = buf.Len()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	offsets[3] = buf.Len()
	if text == "" {
		buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\nendobj\n")
		xrefOffset := buf.Len()
		buf.WriteString("xref\n0 4\n0000000000 65535 f \n")
		for i := 1; i <= 3; i++ {
			buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
		}
		buf.WriteString(fmt.Sprintf("trailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset))
		return buf.Bytes()
	}

	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")

	offsets[4] = buf.Len()
	stream := fmt.Sprintf("BT /F1 24 Tf 100 700 Td (%s) Tj ET", text)
	buf.WriteString(fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(stream), stream))

	offsets[5] = buf.Len()
	buf.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	xrefOffset := buf.Len()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset))
	return buf.Bytes()
}

func TestParsePDF(t *testing.T) {
	pdfBytes := buildTestPDF("Hello World Resume")
	out, err := Parse("cv.pdf", bytes.NewReader(pdfBytes))
	if err != nil {
		t.Fatalf("unexpected error parsing PDF: %v", err)
	}
	if !strings.Contains(out, "Hello World Resume") {
		t.Errorf("expected extracted text to contain 'Hello World Resume', got:\n%s", out)
	}
}

func TestParseEmptyPDF(t *testing.T) {
	emptyPDF := buildTestPDF("")
	_, err := Parse("empty.pdf", bytes.NewReader(emptyPDF))
	if err == nil {
		t.Fatal("expected error on empty PDF text, got nil")
	}
	if err != ErrEmptyPDFText && !strings.Contains(err.Error(), "PDF não contém texto selecionável") {
		t.Fatalf("expected ErrEmptyPDFText, got %v", err)
	}
}
