// RRC-derived unified extensions, AGPL-3.0-only; original renderer MIT notices retained.
package render

import (
	"fmt"
	"io"
	"serviceunlock/internal/extra"
	"strings"
)

func NewMachineOutput(w io.Writer) *Output {
	return &Output{W: w, Theme: newTheme(nil), Color: true, Width: 100}
}
func (o *Output) extraRows(results []extra.Result) []displayRow {
	out := []displayRow{}
	for _, r := range results {
		label, style := "○ Unknown", o.Theme.Error
		switch r.Status {
		case "available":
			label, style = "● Yes", o.Theme.Good
		case "blocked":
			label, style = "● No", o.Theme.Bad
		case "restricted":
			label, style = "● Limited", o.Theme.Warn
		}
		region := ""
		if r.Region != nil {
			region = regionCode(*r.Region)
		}
		out = append(out, displayRow{r.Name, label, region, style})
	}
	return out
}
func extraSummary(results []extra.Result) string {
	counts := map[string]int{}
	for _, r := range results {
		counts[r.Status]++
	}
	lines := []string{}
	for _, p := range []struct{ key, label, meaning string }{{"available", "Yes", "source access criterion met"}, {"blocked", "No", "source access criterion refused"}, {"restricted", "Limited", "partial catalogue or pending availability"}, {"error", "Unknown", "inconclusive, unsupported or safety-limited"}} {
		if counts[p.key] > 0 {
			lines = append(lines, fmt.Sprintf("%s %d: %s", p.label, counts[p.key], p.meaning))
		}
	}
	lines = append(lines, "Access, edit permission and game API observations are not playback/login guarantees.")
	return strings.Join(lines, "\n")
}
