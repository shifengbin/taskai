package fonts

import (
	"sort"
	"strings"
)

type Spacing string

const (
	SpacingMono         Spacing = "mono"
	SpacingDual         Spacing = "dual"
	SpacingProportional Spacing = "proportional"
)

// Candidate identifies a system font family suitable for terminal rendering.
type Candidate struct {
	Family  string  `json:"family"`
	Spacing Spacing `json:"spacing"`
}

// DefaultCandidate represents the existing CSS monospace fallback stack.
var DefaultCandidate = Candidate{Family: "", Spacing: SpacingMono}

// ListTerminalFonts returns a stable, deduplicated list that always includes
// the default fallback option when platform discovery is unavailable.
func ListTerminalFonts() []Candidate {
	return normalizeCandidates(systemCandidates())
}

func normalizeCandidates(candidates []Candidate) []Candidate {
	normalized := make([]Candidate, 0, len(candidates)+1)
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate.Family = strings.TrimSpace(candidate.Family)
		if candidate.Family == "" || (candidate.Spacing != SpacingMono && candidate.Spacing != SpacingDual) {
			continue
		}
		key := strings.ToLower(candidate.Family)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, candidate)
	}
	sort.SliceStable(normalized, func(left, right int) bool {
		return strings.ToLower(normalized[left].Family) < strings.ToLower(normalized[right].Family)
	})
	return append([]Candidate{DefaultCandidate}, normalized...)
}
