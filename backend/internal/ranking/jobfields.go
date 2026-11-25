package ranking

import (
	"regexp"
	"strings"
)

// JobFields is the field decomposition used for weighted BM25. Description is
// always the full text; requirements and responsibilities are the subsets
// recognized by the section heuristics (possibly empty).
type JobFields struct {
	Title            string
	Requirements     string
	Responsibilities string
	Description      string
}

var (
	requirementsHeading = regexp.MustCompile(`(?i)^\s*(requisitos|requirements|qualifica[çc][õo]es|qualifications|o que voc[êe] precisa|must have|voc[êe] ter[áa]|esperamos)\b`)
	responsibilitiesHeading = regexp.MustCompile(`(?i)^\s*(responsabilidades|responsibilities|atividades|duties|o que voc[êe] far[áa]|day to day|sobre a vaga|a fun[çc][ãa]o)\b`)
)

// SplitJobFields decomposes a job description into weighted fields using a
// deterministic heading heuristic. It never calls an LLM or a vocabulary.
func SplitJobFields(title, description string) JobFields {
	fields := JobFields{Title: title, Description: description}

	var req, resp []string
	bucket := ""
	for _, line := range strings.Split(strings.ReplaceAll(description, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case requirementsHeading.MatchString(trimmed):
			bucket = "requirements"
			continue
		case responsibilitiesHeading.MatchString(trimmed):
			bucket = "responsibilities"
			continue
		}
		switch bucket {
		case "requirements":
			if trimmed != "" {
				req = append(req, trimmed)
			}
		case "responsibilities":
			if trimmed != "" {
				resp = append(resp, trimmed)
			}
		}
	}
	fields.Requirements = strings.Join(req, "\n")
	fields.Responsibilities = strings.Join(resp, "\n")
	return fields
}
