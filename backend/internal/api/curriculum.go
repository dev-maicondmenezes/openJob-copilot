package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	ctx := r.Context()

	// 1. Resolve Job Description (from JobID or direct JDText)
	var jdText string
	if req.JobID > 0 {
		job, err := c.repo.GetJob(ctx, req.JobID)
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "vaga não encontrada"})
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
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": fmt.Sprintf("Falha ao gerar currículo adaptado pelo provedor de IA: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tailored_md":    tailoredMD,
		"prompt_version": tailor.PromptVersion,
	})
}
