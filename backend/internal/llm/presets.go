package llm

import "strings"

// PresetsVersion is bumped whenever the catalog changes shape or content in a
// way clients might care about. Presets are only a prefill: every field stays
// editable in the UI, so a stale default never locks anyone out.
const PresetsVersion = 2

// Preset is one homologated OpenAI-compatible provider (backlog B3/B4).
//
// "Homologated" means the endpoint speaks the /chat/completions dialect that
// internal/llm/openai.go uses (docs/adr/0005). Model names are suggestions and
// providers rotate them; the user can always type another one.
type Preset struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	// Model is the suggested default; Models lists every model the UI should
	// offer in the selector. The first entry is always the suggested default.
	Model    string   `json:"model"`
	Models   []string `json:"models"`
	FreeTier bool     `json:"free_tier"`
	// FreeNote is a stable code the UI translates (free_rate_limited,
	// free_local) so the free offer is described honestly, never as unlimited.
	FreeNote string `json:"free_note,omitempty"`
	// KeyURL is the official page where the user creates the API key (B1).
	// Empty for providers that need no key.
	KeyURL string `json:"key_url,omitempty"`
	// DocsURL points at the provider's official documentation/models page.
	DocsURL string `json:"docs_url,omitempty"`
	// RequiresKey is false for local runtimes (Ollama, LM Studio).
	RequiresKey bool `json:"requires_key"`
	// Local marks runtimes that run on the user's machine.
	Local bool `json:"local"`
}

// PresetCatalog is the payload of GET /api/settings/llm/presets.
type PresetCatalog struct {
	Version int      `json:"version"`
	Presets []Preset `json:"presets"`
	// DefaultID is the preset suggested to someone who wants to start for free.
	DefaultID string `json:"default_id"`
}

// Presets returns a fresh copy of the catalog (callers may mutate it).
func Presets() PresetCatalog {
	return PresetCatalog{
		Version: PresetsVersion,
		Presets: []Preset{
			{
				ID: "gemini", Name: "Google Gemini (AI Studio)",
				BaseURL:     "https://generativelanguage.googleapis.com/v1beta/openai",
				Model:       "gemini-2.5-flash",
				Models:      []string{"gemini-2.5-flash", "gemini-2.5-pro", "gemini-2.0-flash"},
				FreeTier:    true,
				FreeNote:    "free_rate_limited",
				KeyURL:      "https://aistudio.google.com/apikey",
				DocsURL:     "https://ai.google.dev/gemini-api/docs/models",
				RequiresKey: true,
			},
			{
				ID: "groq", Name: "Groq",
				BaseURL:     "https://api.groq.com/openai/v1",
				Model:       "llama-3.3-70b-versatile",
				Models:      []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "openai/gpt-oss-120b", "qwen/qwen3-32b"},
				FreeTier:    true,
				FreeNote:    "free_rate_limited",
				KeyURL:      "https://console.groq.com/keys",
				DocsURL:     "https://console.groq.com/docs/models",
				RequiresKey: true,
			},
			{
				ID: "openrouter", Name: "OpenRouter (modelos :free)",
				BaseURL: "https://openrouter.ai/api/v1",
				Model:   "meta-llama/llama-3.3-70b-instruct:free",
				Models: []string{
					"meta-llama/llama-3.3-70b-instruct:free",
					"google/gemma-3-27b-it:free",
					"deepseek/deepseek-r1:free",
					"qwen/qwen3-235b-a22b:free",
				},
				FreeTier:    true,
				FreeNote:    "free_rate_limited",
				KeyURL:      "https://openrouter.ai/keys",
				DocsURL:     "https://openrouter.ai/docs",
				RequiresKey: true,
			},
			{
				ID: "ollama", Name: "Ollama (local)",
				BaseURL:  "http://localhost:11434/v1",
				Model:    "llama3.2",
				Models:   []string{"llama3.2", "qwen2.5", "mistral", "gemma3"},
				FreeTier: true,
				FreeNote: "free_local",
				KeyURL:   "https://ollama.com",
				DocsURL:  "https://docs.ollama.com/",
				Local:    true,
			},
			{
				ID: "lmstudio", Name: "LM Studio (local)",
				BaseURL:  "http://localhost:1234/v1",
				Model:    "local-model",
				Models:   []string{"local-model", "qwen2.5-7b-instruct", "llama-3.2-3b-instruct"},
				FreeTier: true,
				FreeNote: "free_local",
				KeyURL:   "https://lmstudio.ai",
				DocsURL:  "https://lmstudio.ai/docs",
				Local:    true,
			},
			{
				ID: "openai", Name: "OpenAI",
				BaseURL:     "https://api.openai.com/v1",
				Model:       "gpt-4o-mini",
				Models:      []string{"gpt-4o-mini", "gpt-4o", "gpt-4.1-mini", "gpt-4.1", "o4-mini"},
				KeyURL:      "https://platform.openai.com/api-keys",
				DocsURL:     "https://platform.openai.com/docs/models",
				RequiresKey: true,
			},
			{
				ID: "together", Name: "Together AI",
				BaseURL: "https://api.together.xyz/v1",
				Model:   "meta-llama/Llama-3.3-70B-Instruct-Turbo",
				Models: []string{
					"meta-llama/Llama-3.3-70B-Instruct-Turbo",
					"meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo",
					"Qwen/Qwen2.5-72B-Instruct-Turbo",
					"mistralai/Mixtral-8x7B-Instruct-v0.1",
				},
				KeyURL:      "https://api.together.ai/settings/api-keys",
				DocsURL:     "https://docs.together.ai/docs/serverless-models",
				RequiresKey: true,
			},
			{
				ID: "fireworks", Name: "Fireworks AI",
				BaseURL: "https://api.fireworks.ai/inference/v1",
				Model:   "accounts/fireworks/models/llama-v3p3-70b-instruct",
				Models: []string{
					"accounts/fireworks/models/llama-v3p3-70b-instruct",
					"accounts/fireworks/models/llama-v3p1-8b-instruct",
					"accounts/fireworks/models/qwen3-235b-a22b",
					"accounts/fireworks/models/deepseek-r1",
				},
				KeyURL:      "https://fireworks.ai/account/api-keys",
				DocsURL:     "https://docs.fireworks.ai/",
				RequiresKey: true,
			},
		},
		DefaultID: "gemini",
	}
}

// PresetByID looks a preset up by its stable id.
func PresetByID(id string) (Preset, bool) {
	for _, p := range Presets().Presets {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}

// LocalKeyPlaceholder is stored in place of an API key for local runtimes that
// ignore authentication. The OpenAI-compatible client always sends a bearer
// token, and every consumer treats "no key" as "not configured".
const LocalKeyPlaceholder = "local-no-key"

// IsLocalBaseURL reports whether baseURL points at the user's own machine, the
// only case where an empty API key is acceptable.
func IsLocalBaseURL(baseURL string) bool {
	u := strings.ToLower(strings.TrimSpace(baseURL))
	for _, prefix := range []string{"http://", "https://"} {
		u = strings.TrimPrefix(u, prefix)
	}
	host := u
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host = host[:i]
	}
	if strings.HasPrefix(host, "[") { // [::1]:11434
		if j := strings.Index(host, "]"); j > 0 {
			return host[1:j] == "::1"
		}
		return false
	}
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	return host == "localhost" || host == "127.0.0.1" || host == "host.docker.internal"
}
