package ranking

import (
	"fmt"
	"strings"
)

// buildReason produces a deterministic, human-readable explanation in PT-BR.
// It is intentionally lexical: it states which terms matched and which are
// missing, without implying semantic or hiring-probability judgement.
func buildReason(score int, raw float64, matched, missing []string, total int) string {
	if total == 0 {
		return "Aderência de 0/100 — não foi possível identificar termos relevantes."
	}
	if len(matched) == 0 {
		reason := fmt.Sprintf("Aderência de %d/100 — nenhum termo da vaga foi encontrado no currículo.", score)
		if len(missing) > 0 {
			reason += " Lacunas: " + joinLimit(missing, 5) + "."
		}
		return reason
	}
	reason := fmt.Sprintf("Aderência de %d/100 — termos encontrados: %s (%d/%d termos-chave).",
		score, joinLimit(matched, 5), len(matched), total)
	if len(missing) > 0 {
		reason += " Lacunas: " + joinLimit(missing, 5) + "."
	}
	return reason
}

func joinLimit(values []string, limit int) string {
	if len(values) > limit {
		values = values[:limit]
	}
	return strings.Join(values, ", ")
}
