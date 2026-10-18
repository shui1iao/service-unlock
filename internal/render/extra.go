// RRC-derived unified extensions, AGPL-3.0-only; original renderer MIT notices retained.
package render

import (
	"io"
	"serviceunlock/internal/extra"
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
