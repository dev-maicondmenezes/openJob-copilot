package discovery

import (
	"strings"

	"golang.org/x/net/html"
)

// cardSpec declares the CSS-ish hooks used to read a server-rendered listing.
// Scrapers must be resilient to markup drift: every field has a fallback and a
// card without title+link is skipped instead of poisoning the run.
type cardSpec struct {
	CardClasses     []string
	CardTags        []string
	TitleClasses    []string
	CompanyClasses  []string
	LocationClasses []string
	SalaryClasses   []string
	LinkClasses     []string
}

func parseListingCards(doc *html.Node, baseURL string, spec cardSpec) []DiscoveredJob {
	cards := findCards(doc, spec)
	jobs := make([]DiscoveredJob, 0, len(cards))
	for _, card := range cards {
		title := pickText(card, spec.TitleClasses...)
		if title == "" {
			// Fallback: first heading or anchor text with reasonable length.
			for _, tag := range []string{"h1", "h2", "h3", "h4"} {
				if n := firstByTag(card, tag); n != nil {
					if t := strings.TrimSpace(textNode(n)); t != "" {
						title = t
						break
					}
				}
			}
		}
		href := pickLink(card, spec.LinkClasses...)
		if href == "" {
			href = firstLink(card)
		}
		if href == "" {
			href = attrVal(card, "data-href")
		}
		url := absoluteURL(baseURL, href)
		if title == "" || url == "" {
			continue
		}
		company := pickText(card, spec.CompanyClasses...)
		location := pickText(card, spec.LocationClasses...)
		salary := pickText(card, spec.SalaryClasses...)
		jobs = append(jobs, DiscoveredJob{
			Title:       title,
			Company:     company,
			URL:         url,
			Description: composeDescription("", location, "", salary),
			Location:    location,
			Salary:      salary,
			ExternalID:  extractExternalID(url),
		})
	}
	return jobs
}

func findCards(doc *html.Node, spec cardSpec) []*html.Node {
	for _, class := range spec.CardClasses {
		if nodes := allByClass(doc, class); len(nodes) > 0 {
			return nodes
		}
	}
	for _, tag := range spec.CardTags {
		if nodes := allByTag(doc, tag); len(nodes) > 0 {
			return nodes
		}
	}
	// Last resort: <article> cards (semantic listings).
	return allByTag(doc, "article")
}

func pickText(root *html.Node, classes ...string) string {
	for _, class := range classes {
		if n := firstByClass(root, class); n != nil {
			if text := strings.TrimSpace(textNode(n)); text != "" {
				return text
			}
		}
	}
	return ""
}

func pickLink(root *html.Node, classes ...string) string {
	for _, class := range classes {
		if n := firstByClass(root, class); n != nil {
			if n.Data == "a" {
				if href := attrVal(n, "href"); href != "" {
					return href
				}
			}
			if href := firstLink(n); href != "" {
				return href
			}
		}
	}
	return ""
}

// slugifyKeyword turns a keyword into the path segment used by the sources
// (lowercase, spaces → dashes). Accents are kept so local listings match.
func slugifyKeyword(keyword string) string {
	k := strings.ToLower(strings.TrimSpace(keyword))
	k = strings.Join(strings.Fields(k), "-")
	replacer := strings.NewReplacer("/", "-", "?", "", "#", "", "&", "")
	return replacer.Replace(k)
}

// appendMetaDescription merges a detail description with the card metadata
// prefix without duplicating the listing fields.
func appendMetaDescription(meta, detail string) string {
	meta = strings.TrimSpace(meta)
	detail = CleanHTML(detail)
	switch {
	case meta == "":
		return detail
	case detail == "" || strings.Contains(detail, meta):
		return detail
	default:
		return meta + "\n\n" + detail
	}
}
