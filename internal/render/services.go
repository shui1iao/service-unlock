// Terminal-only interpretation of consumer-service observations.
// MIT License; Copyright (c) 2026 Remnawave.
package render

import "serviceunlock/internal/geo"

func (o *Output) serviceRows(results []geo.Result) []displayRow {
	rows := make([]displayRow, 0, len(results))
	for _, r := range results {
		label, region, style := "○ Unknown", "", o.Theme.Error
		if r.Check.Kind == geo.KindCountry {
			region = regionCode(r.Value)
		}
		if r.Err == nil {
			switch r.Check.Kind {
			case geo.KindCountry:
				if region != "" {
					label, style = "● Yes", o.Theme.Good
				}
			case geo.KindAvailability:
				if r.Value == "yes" {
					label, style = "● Yes", o.Theme.Good
				} else if r.Value == "no" {
					label, style = "● No", o.Theme.Bad
				}
			case geo.KindBlocked:
				if r.Value == "yes" {
					label, style = "● No", o.Theme.Bad
				} else if r.Value == "no" {
					label, style = "● Yes", o.Theme.Good
				}
			}
		}
		rows = append(rows, displayRow{r.Check.Name, label, region, style})
	}
	return rows
}
