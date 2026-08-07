package fonts

import (
	"reflect"
	"testing"
)

func TestNormalizeCandidatesKeepsTerminalFontsAndAddsDefault(t *testing.T) {
	got := normalizeCandidates([]Candidate{
		{Family: " Fira Code ", Spacing: SpacingMono},
		{Family: "fira code", Spacing: SpacingMono},
		{Family: "Noto Sans CJK SC", Spacing: SpacingDual},
		{Family: "Noto Sans", Spacing: SpacingProportional},
		{Family: "", Spacing: SpacingMono},
	})

	want := []Candidate{
		DefaultCandidate,
		{Family: "Fira Code", Spacing: SpacingMono},
		{Family: "Noto Sans CJK SC", Spacing: SpacingDual},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeCandidates() = %#v, want %#v", got, want)
	}
}

func TestNormalizeCandidatesReturnsDefaultWhenDiscoveryHasNoTerminalFonts(t *testing.T) {
	if got := normalizeCandidates([]Candidate{{Family: "Noto Sans", Spacing: SpacingProportional}}); !reflect.DeepEqual(got, []Candidate{DefaultCandidate}) {
		t.Fatalf("normalizeCandidates() = %#v, want default candidate", got)
	}
}
