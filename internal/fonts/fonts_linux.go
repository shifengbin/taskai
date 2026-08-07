//go:build linux

package fonts

import (
	"os/exec"
	"strconv"
	"strings"
)

const (
	fontconfigSpacingDual     = 90
	fontconfigSpacingMono     = 100
	fontconfigSpacingCharcell = 110
)

func systemCandidates() []Candidate {
	output, err := exec.Command("fc-list", "--format=%{family}\\t%{spacing}\\n").Output()
	if err != nil {
		return nil
	}

	candidates := make([]Candidate, 0)
	for _, line := range strings.Split(string(output), "\n") {
		familyList, spacingValue, found := strings.Cut(line, "\t")
		if !found {
			continue
		}
		spacing, err := strconv.Atoi(strings.TrimSpace(spacingValue))
		if err != nil {
			continue
		}
		candidateSpacing := fontconfigSpacing(spacing)
		for _, family := range strings.Split(familyList, ",") {
			candidates = append(candidates, Candidate{Family: family, Spacing: candidateSpacing})
		}
	}
	return candidates
}

func fontconfigSpacing(spacing int) Spacing {
	switch spacing {
	case fontconfigSpacingMono, fontconfigSpacingCharcell:
		return SpacingMono
	case fontconfigSpacingDual:
		return SpacingDual
	default:
		return SpacingProportional
	}
}
