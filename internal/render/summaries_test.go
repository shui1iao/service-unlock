package render

import (
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

// Select the entire frame, including section titles but no outside notices.
func tableText(text string) string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "│") {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
func TestRegionColumnsContainCodesNotNames(t *testing.T) {
	text := ansi.Strip(renderProfile(t, englishFixture(t, true), colorprofile.TrueColor))
	for _, name := range []string{"United States", "France", "Hong Kong", "Japan", "China"} {
		if strings.Contains(tableText(text), name) {
			t.Fatal("country name remains", name)
		}
	}
	for _, code := range []string{"US", "HK", "JP", "FR", "CN"} {
		if !strings.Contains(tableText(text), " "+code+" ") {
			t.Fatal("country code missing", code)
		}
	}
}
