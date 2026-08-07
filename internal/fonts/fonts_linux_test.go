//go:build linux

package fonts

import "testing"

func TestFontconfigSpacingClassifiesCharcellAsMonospace(t *testing.T) {
	tests := []struct {
		spacing int
		want    Spacing
	}{
		{fontconfigSpacingDual, SpacingDual},
		{fontconfigSpacingMono, SpacingMono},
		{fontconfigSpacingCharcell, SpacingMono},
		{0, SpacingProportional},
	}

	for _, test := range tests {
		if got := fontconfigSpacing(test.spacing); got != test.want {
			t.Errorf("fontconfigSpacing(%d) = %q, want %q", test.spacing, got, test.want)
		}
	}
}
