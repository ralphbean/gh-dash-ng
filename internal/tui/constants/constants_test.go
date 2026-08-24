package constants

import (
	"strings"
	"testing"
)

func TestLogoContainsForkBranding(t *testing.T) {
	lines := strings.Split(Logo, "\n")
	if len(lines) != 2 {
		t.Fatalf("Logo should have 2 lines, got %d", len(lines))
	}

	topRunes := []rune(lines[0])
	bottomRunes := []rune(lines[1])

	if len(topRunes) != len(bottomRunes) {
		t.Errorf("Logo lines should be the same rune width, got %d and %d",
			len(topRunes), len(bottomRunes))
	}

	// The upstream "dash" logo is 11 runes wide per line.
	// The fork "dash-ng" logo must be wider to include the "-ng" suffix.
	if len(topRunes) <= 11 {
		t.Errorf("Logo should be wider than upstream (11 runes) to include "+
			"'-ng' branding, got %d runes", len(topRunes))
	}
}
