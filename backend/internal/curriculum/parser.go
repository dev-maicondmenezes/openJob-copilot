package curriculum

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

var (
	// ErrUnsupportedFormat is returned when the uploaded file extension is not supported.
	ErrUnsupportedFormat = errors.New("formato de arquivo não suportado (utilize .pdf, .md ou .txt)")
	// ErrEmptyFile is returned when the uploaded file is empty.
	ErrEmptyFile = errors.New("arquivo vazio")
	// ErrEmptyPDFText is returned when a PDF contains no extractable text.
	ErrEmptyPDFText = errors.New("o PDF não contém texto selecionável (PDFs escaneados ou imagens não são suportados; utilize um PDF com texto, .md ou .txt)")
)

// MaxUploadSize is 10 MiB, plenty for any text or PDF resume.
const MaxUploadSize = 10 << 20

// Known section headers (case-insensitive) for structuring plain text resumes.
var knownSections = []struct {
	pattern *regexp.Regexp
	title   string
}{
	{regexp.MustCompile(`(?i)^(resumo(\s+profissional)?|sobre(\s+mim)?|summary|profile|about(\s+me)?):?$`), "Resumo"},
	{regexp.MustCompile(`(?i)^(experi[êe]ncia(\s+profissional)?|work\s+experience|professional\s+experience|hist[óo]rico\s+profissional):?$`), "Experiência"},
	{regexp.MustCompile(`(?i)^(habilidades(\s+t[ée]cnicas)?|skills|compet[êe]ncias|conhecimentos(\s+t[ée]cnicos)?|tecnologias):?$`), "Habilidades"},
	{regexp.MustCompile(`(?i)^(forma[çc][ãa]o(\s+acad[êe]mica)?|educa[çc][ãa]o|education|academic\s+background):?$`), "Educação"},
	{regexp.MustCompile(`(?i)^(projetos(\s+relevantes)?|projects|principais\s+projetos):?$`), "Projetos"},
	{regexp.MustCompile(`(?i)^(certifica[çc][õo]es|certifications|cursos|licen[çc]as):?$`), "Certificações"},
	{regexp.MustCompile(`(?i)^(idiomas|languages):?$`), "Idiomas"},
	{regexp.MustCompile(`(?i)^(contato|contact(\s+info(rmation)?)?):?$`), "Contato"},
}

var bulletRegex = regexp.MustCompile(`^[\s]*[•·*–o]\s*(.*)$`)

// Parse extracts text from the uploaded file and normalizes it to structured Markdown.
func Parse(filename string, r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxUploadSize+1))
	if err != nil {
		return "", fmt.Errorf("falha ao ler arquivo: %w", err)
	}
	if len(data) == 0 {
		return "", ErrEmptyFile
	}
	if len(data) > MaxUploadSize {
		return "", fmt.Errorf("arquivo excede o limite máximo de %d MiB", MaxUploadSize/(1<<20))
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return parsePDF(data)
	case ".md", ".markdown":
		return parseMarkdown(data)
	case ".txt":
		return parsePlainText(data)
	default:
		return "", ErrUnsupportedFormat
	}
}

func parsePDF(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("falha ao ler PDF: %w", err)
	}

	textReader, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("falha ao extrair texto do PDF: %w", err)
	}

	extracted, err := io.ReadAll(textReader)
	if err != nil {
		return "", fmt.Errorf("falha ao processar texto extraído: %w", err)
	}

	rawText := strings.TrimSpace(string(extracted))
	if rawText == "" {
		return "", ErrEmptyPDFText
	}

	return NormalizeToMarkdown(rawText), nil
}

func parseMarkdown(data []byte) (string, error) {
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return "", ErrEmptyFile
	}
	return NormalizeToMarkdown(raw), nil
}

func parsePlainText(data []byte) (string, error) {
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return "", ErrEmptyFile
	}
	return NormalizeToMarkdown(raw), nil
}

// NormalizeToMarkdown cleans and structures raw text or markdown into neat Markdown.
func NormalizeToMarkdown(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	prevEmpty := false

	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r \t")
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			if !prevEmpty && len(result) > 0 {
				result = append(result, "")
				prevEmpty = true
			}
			continue
		}
		prevEmpty = false

		// Check if line is already a markdown heading (#, ##, ###)
		if strings.HasPrefix(trimmed, "#") {
			result = append(result, trimmed)
			continue
		}

		// Check if line matches a known section title
		matchedSection := false
		for _, s := range knownSections {
			if s.pattern.MatchString(trimmed) {
				// Insert extra blank line before header if previous was not empty
				if len(result) > 0 && result[len(result)-1] != "" {
					result = append(result, "")
				}
				result = append(result, "## "+s.title)
				matchedSection = true
				break
			}
		}
		if matchedSection {
			continue
		}

		// Check if line is a bullet item (•, -, *, etc.)
		if m := bulletRegex.FindStringSubmatch(line); len(m) > 1 {
			result = append(result, "- "+strings.TrimSpace(m[1]))
			continue
		}

		// Keep existing markdown list item if starts with - or *
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			result = append(result, "- "+strings.TrimSpace(trimmed[2:]))
			continue
		}

		result = append(result, trimmed)
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}

// ErrorMessage returns a user-friendly message for a known curriculum parser error.
func ErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrUnsupportedFormat):
		return "Formato de arquivo não suportado. Use .pdf, .md ou .txt."
	case errors.Is(err, ErrEmptyFile):
		return "Arquivo vazio. Por favor, envie um arquivo com conteúdo."
	case errors.Is(err, ErrEmptyPDFText):
		return "O PDF não contém texto selecionável. Use um PDF com texto ou envie .md/.txt."
	default:
		return "Erro ao processar o arquivo. Verifique o formato e tente novamente."
	}
}
