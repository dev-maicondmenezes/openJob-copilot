package keywords

import "sort"

// consolidate merges per-section results, deduplicates expressions and applies
// the top-N limit. It never compares raw scores across algorithms: only the
// per-section rank and source priority decide which duplicate survives.
func consolidate(groups [][]Keyword, topN int) []Keyword {
	if topN <= 0 {
		topN = 20
	}
	best := make(map[string]Keyword)
	order := make([]string, 0)

	for _, group := range groups {
		for _, kw := range group {
			if kw.Normalized == "" {
				continue
			}
			current, ok := best[kw.Normalized]
			if !ok {
				best[kw.Normalized] = kw
				order = append(order, kw.Normalized)
				continue
			}
			if betterDuplicate(kw, current) {
				best[kw.Normalized] = kw
			}
		}
	}

	merged := make([]Keyword, 0, len(best))
	for _, key := range order {
		merged = append(merged, best[key])
	}

	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].Rank != merged[j].Rank {
			return merged[i].Rank < merged[j].Rank
		}
		if sourcePriority(merged[i].Source) != sourcePriority(merged[j].Source) {
			return sourcePriority(merged[i].Source) < sourcePriority(merged[j].Source)
		}
		if merged[i].Section != merged[j].Section {
			return merged[i].Section < merged[j].Section
		}
		return merged[i].Normalized < merged[j].Normalized
	})

	if len(merged) > topN {
		merged = merged[:topN]
	}
	for i := range merged {
		merged[i].Rank = i + 1
	}
	return merged
}

func betterDuplicate(candidate, current Keyword) bool {
	if candidate.Rank != current.Rank {
		return candidate.Rank < current.Rank
	}
	if sourcePriority(candidate.Source) != sourcePriority(current.Source) {
		return sourcePriority(candidate.Source) < sourcePriority(current.Source)
	}
	return candidate.Section < current.Section
}

func sourcePriority(source string) int {
	if source == SourceYAKE {
		return 0
	}
	return 1
}
