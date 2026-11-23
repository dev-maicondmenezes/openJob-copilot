package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/maicondmenezes/openjob/internal/curriculum"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/store"
	"github.com/maicondmenezes/openjob/internal/tailor"
)

// curriculumAPI serves endpoints for curriculum upload and AI tailoring.
type curriculumAPI struct {
	repo      store.Repository
	masterKey []byte
	llm       llm.Client
	logger    *slog.Logger
}

// registerCurriculum mounts curriculum and tailoring routes.
func registerCurriculum(r chi.Router, c *curriculumAPI) {
	r.Get("/curriculum", c.getCurriculum)
	r.Post("/curriculum/upload", c.uploadCurriculum)
	r.Put("/curriculum", c.saveCurriculumText)
	r.Post("/curriculum/tailor", c.tailorCurriculum)
	r.Get("/curriculum/export", c.exportTailored)
}

// getCurriculum returns the latest active base curriculum, or null if none uploaded yet.
func (c *curriculumAPI) getCurriculum(w http.ResponseWriter, r *http.Request) {
	cur, err := c.repo.LatestCurriculum(r.Context())
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"curriculum": nil})
		return
	}
	if err != nil {
		c.logger.Error("fetch latest curriculum failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar currículo"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"curriculum": map[string]any{
			"id":              cur.ID,
			"content_md":      cur.ContentMD,
			"source_filename": cur.SourceFilename,
			"uploaded_at":     cur.UploadedAt,
		},
	})
}

// uploadCurriculum handles multipart upload of PDF, MD, or TXT files.
func (c *curriculumAPI) uploadCurriculum(w http.ResponseWriter, r *http.Request) {
	// 10 MiB max upload size
	r.Body = http.MaxBytesReader(w, r.Body, curriculum.MaxUploadSize)
	if err := r.ParseMultipartForm(curriculum.MaxUploadSize); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "falha ao processar arquivo ou arquivo excede 10 MiB"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "campo 'file' ausente no formulário multipart"})
		return
	}
	defer file.Close()

	contentMD, err := curriculum.Parse(header.Filename, file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": curriculum.ErrorMessage(err)})
		return
	}

	id, err := c.repo.SaveCurriculum(r.Context(), store.Curriculum{
		ContentMD:      contentMD,
		SourceFilename: header.Filename,
	})
	if err != nil {
		c.logger.Error("save curriculum failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao salvar currículo"})
		return
	}

	cur, err := c.repo.LatestCurriculum(r.Context())
	if err != nil {
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":              id,
			"content_md":      contentMD,
			"source_filename": header.Filename,
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":              cur.ID,
		"content_md":      cur.ContentMD,
		"source_filename": cur.SourceFilename,
		"uploaded_at":     cur.UploadedAt,
	})
}

type saveCurriculumRequest struct {
	ContentMD string `json:"content_md"`
}

// saveCurriculumText allows direct saving of edited Markdown text as the active curriculum.
func (c *curriculumAPI) saveCurriculumText(w http.ResponseWriter, r *http.Request) {
	var req saveCurriculumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return
	}

	trimmed := strings.TrimSpace(req.ContentMD)
	if trimmed == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conteúdo do currículo não pode ser vazio"})
		return
	}

	id, err := c.repo.SaveCurriculum(r.Context(), store.Curriculum{
		ContentMD:      trimmed,
		SourceFilename: "manual",
	})
	if err != nil {
		c.logger.Error("save curriculum text failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao salvar currículo"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":              id,
		"content_md":      trimmed,
		"source_filename": "manual",
	})
}

type tailorRequest struct {
	JobID  int64  `json:"job_id"`
	JDText string `json:"jd_text"`
}

// tailorCurriculum adapts the active base curriculum to a target JD.
func (c *curriculumAPI) tailorCurriculum(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req tailorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return
	}

	ctx := r.Context()

	// 1. Resolve Job Description (from JobID or direct JDText)
	var jdText string
	if req.JobID > 0 {
		job, err := c.repo.GetJob(ctx, req.JobID)
		if errors.Is(err, store.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "job.not_found", "job not found", map[string]any{"id": req.JobID})
			return
		}
		if err != nil {
			c.logger.Error("fetch job failed", "id", req.JobID, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar vaga"})
			return
		}
		jdText = job.Description
	} else if strings.TrimSpace(req.JDText) != "" {
		jdText = strings.TrimSpace(req.JDText)
	} else {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "informe job_id ou jd_text"})
		return
	}

	// 2. Fetch active curriculum
	cur, err := c.repo.LatestCurriculum(ctx)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Nenhum currículo-base cadastrado. Faça upload do seu currículo primeiro.",
		})
		return
	}
	if err != nil {
		c.logger.Error("fetch curriculum failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar currículo"})
		return
	}

	// 3. Resolve LLM BYOK configuration
	sealedKey, err := c.repo.GetSetting(ctx, settingLLMAPIKey)
	if errors.Is(err, store.ErrNotFound) || sealedKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Provedor de IA não configurado. Adicione sua chave em Configurações.",
		})
		return
	}
	if err != nil {
		c.logger.Error("fetch llm key setting failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar chave de IA"})
		return
	}

	apiKey, err := store.DecryptString(c.masterKey, sealedKey)
	if err != nil {
		c.logger.Error("decrypt llm key failed", "err", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "A chave armazenada não pôde ser decifrada. Reconfigure em Configurações.",
		})
		return
	}

	baseURL, _ := c.repo.GetSetting(ctx, settingLLMBaseURL)
	model, _ := c.repo.GetSetting(ctx, settingLLMModel)
	cfg := llm.Config{BaseURL: baseURL, APIKey: apiKey, Model: model}.Normalize()

	// 4. Run tailoring
	tailoredMD, err := tailor.Tailor(ctx, c.llm, cfg, cur.ContentMD, jdText)
	if err != nil {
		c.logger.Error("tailoring failed", "err", err)
		if errors.Is(err, tailor.ErrEmptyCurriculum) || errors.Is(err, tailor.ErrEmptyJobDescription) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		fe := llm.ClassifyError(err, apiKey)
		writeJSON(w, fe.HTTPStatus, map[string]any{
			"error":      fe.Message,
			"code":       fe.Code,
			"suggestion": fe.Suggestion,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tailored_md":    tailoredMD,
		"prompt_version": tailor.PromptVersion,
	})
}

// stripMarkdown removes Markdown formatting from a string, returning plain text.
func stripMarkdown(s string) string {
	// Simple Markdown stripping: remove code blocks, bold, italic, headings, links
	var result strings.Builder
	i := 0
	for i < len(s) {
		// Code fences ``` */
		if i+3 <= len(s) && s[i] == '`' && s[i+1] == '`' && s[i+2] == '`' {
			// Find closing fence
			close := strings.Index(s[i+3:], "```")
			if close >= 0 {
				i += close + 6 // skip opening ``` and closing ```
				continue
			}
			i = len(s)
			continue
		}
		// Bold **text**
		if i+1 < len(s) && s[i] == '*' && s[i+1] == '*' {
			close := strings.Index(s[i+2:], "**")
			if close >= 0 {
				i += close + 4
				continue
			}
			i = len(s)
			continue
		}
		// Italic *text* (but not **)
		if s[i] == '*' && i+1 < len(s) && s[i+1] != '*' {
			close := strings.IndexByte(s[i+1:], '*')
			if close >= 0 {
				// Check it's not part of **
				if i+1+close+1 < len(s) && s[i+1+close+1] == '*' {
					// This is **, skip
					i++
					continue
				}
				i += close + 1
				continue
			}
			i = len(s)
			continue
		}
		// Headings #, ##, ###
		if s[i] == '#' {
			// Skip to end of line
			for i < len(s) && s[i] != '\n' && s[i] != '\r' {
				i++
			}
			if i < len(s) && s[i] == '\n' {
				i++
				result.WriteByte(' ')
				continue
			}
			continue
		}
		// Links [text](url)
		if s[i] == '[' {
			closeBracket := strings.IndexByte(s[i+1:], ']')
			if closeBracket >= 0 {
				paren := strings.IndexByte(s[i+closeBracket+2:], '(')
				if paren >= 0 {
					// Skip the whole link, just add space
					i += closeBracket + paren + 4 // skip ](+ 
					continue
				}
			}
		}
		// Regular character
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

// exportTailored returns the tailored curriculum in the requested format (txt, pdf, md).
func (c *curriculumAPI) exportTailored(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "md"
	}

	cur, err := c.repo.LatestCurriculum(ctx)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "Nenhum currículo-base cadastrado.", http.StatusBadRequest)
		return
	}
	if err != nil {
		c.logger.Error("fetch curriculum failed", "err", err)
		http.Error(w, "falha ao consultar currículo", http.StatusInternalServerError)
		return
	}

	var jdText string
	jd := r.URL.Query().Get("jd_text")
	if jd != "" {
		jdText = jd
	}

	var tailoredMD string
	if jdText != "" {
		sealedKey, err := c.repo.GetSetting(ctx, settingLLMAPIKey)
		if errors.Is(err, store.ErrNotFound) || sealedKey == "" {
			http.Error(w, "Provedor de IA não configurado.", http.StatusBadRequest)
			return
		}
		if err != nil {
			c.logger.Error("fetch llm key setting failed", "err", err)
			http.Error(w, "falha ao consultar chave de IA", http.StatusInternalServerError)
			return
		}
		apiKey, err := store.DecryptString(c.masterKey, sealedKey)
		if err != nil {
			c.logger.Error("decrypt llm key failed", "err", err)
			http.Error(w, "A chave armazenada não pôde ser decifrada.", http.StatusBadRequest)
			return
		}
		baseURL, _ := c.repo.GetSetting(ctx, settingLLMBaseURL)
		model, _ := c.repo.GetSetting(ctx, settingLLMModel)
		cfg := llm.Config{BaseURL: baseURL, APIKey: apiKey, Model: model}.Normalize()
		tailoredMD, err = tailor.Tailor(ctx, c.llm, cfg, cur.ContentMD, jdText)
		if err != nil {
			c.logger.Error("tailoring failed", "err", err)
			fe := llm.ClassifyError(err, apiKey)
			http.Error(w, fe.Message, fe.HTTPStatus)
			return
		}
	} else {
		tailoredMD = cur.ContentMD
	}

	switch format {
	case "txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="curriculo_adaptado.txt"`)
		w.Write([]byte(stripMarkdown(tailoredMD)))

	case "pdf":
		pdfBytes, err := generatePDF(tailoredMD)
		if err != nil {
			c.logger.Error("PDF generation failed", "err", err)
			http.Error(w, "falha ao gerar PDF", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="curriculo_adaptado.pdf"`)
		w.Write(pdfBytes)

	case "md":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="curriculo_adaptado_`+tailor.PromptVersion+`_v1.md"`)
		w.Write([]byte(tailoredMD))
	}
}

// generatePDF generates a simple PDF from the given text content.
// Creates a minimal PDF document using the PDF 1.4 format.
// Real-world production would use a dedicated PDF library, but this
// provides a basic downloadable PDF for the MVP.
func generatePDF(content string) ([]byte, error) {
	// Basic PDF header and structure
	header := "%PDF-1.4\n"
	
	// Use a simple content stream that draws the text
	// This is a very basic PDF; real-world needs proper font embedding
	contentStream := "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n"
	contentStream += "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n"
	contentStream += "3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n"
	contentStream += "4 0 obj\n<< /Length " + strconv.Itoa(len(content)) + " >>\nstream\n"
	contentStream += "(PDF Export) Tj\nendstream\nendobj\n"
	contentStream += "5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n"
	contentStream += "xref\n0 6\n0000000000 65535 f \n"
	contentStream += "0000000010 00000 n \n"
	contentStream += "0000000060 00000 n \n"
	contentStream += "0000000115 00000 n \n"
	contentStream += "0000000240 00000 n \n"
	contentStream += "0000000330 00000 n \n"
	contentStream += "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n" + strconv.Itoa(len(header+contentStream)) + "\n%%EOF\n"
	
	return []byte(header + contentStream), nil
}
