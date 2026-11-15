package score

import (
	"fmt"
	"strings"
)

// PromptVersion tracks the current scoring prompt iteration (CP5 plan:
// "anotar v1 do prompt de scoring" — CP8 assesses calibration).
const PromptVersion = "v1"

// Limits for context budget and token control (approx 8k tokens ~ 24k chars).
const (
	MaxCurriculumChars = 24000
	MaxJDChars         = 24000
)

// SystemPrompt defines the rules for scoring the candidate's fit against a
// job description. The output format is a strict JSON object so parsing is
// deterministic (parser.go has a fallback for misbehaving models anyway).
const SystemPrompt = `Você é um recrutador técnico sênior especializado em avaliar aderência candidato-vaga.
Sua missão: dado o currículo-base do candidato e a descrição de uma vaga (JD), calcular um score de fit.

ESCALA (0–100):
- 85–100: match excelente — requisitos essenciais todos cobertos, senioridade compatível.
- 65–84: bom match — maioria dos requisitos cobertos, com algumas lacunas pontuais.
- 40–64: match parcial — exige aprender tecnologias centrais da vaga ou mudar de domínio.
- 10–39: match fraco — pouca sobreposição entre perfil e requisitos.
- 0–9: sem aderência — vaga em área completamente distante do histórico.

REGRAS ESTRITAS (OBRIGATÓRIAS):
1. Baseie o score EXCLUSIVAMENTE no conteúdo do currículo-base fornecido. Não presuma experiência que não está lá.
2. Pese mais os requisitos essenciais (obrigatórios) do que os desejáveis.
3. Considere senioridade, domínio técnico e stack — nessa ordem de importância relativa.
4. Seja exigente: notas altas devem ser raras. Não infle scores.

FORMATO DE SAÍDA (OBRIGATÓRIO):
Responda EXCLUSIVAMENTE com um objeto JSON válido, sem crases, sem texto antes ou depois:
{"score": <inteiro de 0 a 100>, "reason": "<justificativa em português, 2 a 4 frases, citando pontos de aderência e lacunas>"}`

// BuildUserPrompt builds the user turn with safety limits.
func BuildUserPrompt(curriculumMD, jdText string) string {
	curriculumMD = truncate(strings.TrimSpace(curriculumMD), MaxCurriculumChars)
	jdText = truncate(strings.TrimSpace(jdText), MaxJDChars)

	return fmt.Sprintf(`=== CURRÍCULO BASE ===
%s

=== DESCRIÇÃO DA VAGA (JD) ===
%s

Avalie a aderência do candidato a esta vaga e responda apenas com o JSON no formato especificado.`, curriculumMD, jdText)
}

func truncate(s string, maxChars int) string {
	if len(s) <= maxChars {
		return s
	}
	return s[:maxChars] + "\n\n[... truncado pelo limite de tamanho ...]"
}
