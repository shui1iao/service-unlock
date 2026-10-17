// Terminal-only interpretation of consumer-service observations.
// MIT License; Copyright (c) 2026 Remnawave.
package render

import "serviceunlock/internal/geo"

func (o *Output) serviceRows(results []geo.Result) []displayRow {
	rows := make([]displayRow, 0, len(results))
	for _, r := range results {
		label, region, style := "○ 未知", "—", o.Theme.Error
		if r.Err == nil {
			switch r.Check.Kind {
			case geo.KindCountry:
				region = regionName(r.Value)
				if region != "—" {
					label, style = "● 已识别地区", o.Theme.Value
				}
			case geo.KindAvailability:
				if r.Value == "yes" {
					label, style = "● 可访问", o.Theme.Good
				} else if r.Value == "no" {
					label, style = "● 不可访问", o.Theme.Bad
				}
			case geo.KindBlocked:
				if r.Value == "yes" {
					label, style = "● 发现限制", o.Theme.Bad
				} else if r.Value == "no" {
					label, style = "● 未发现限制", o.Theme.Good
				}
			}
		}
		rows = append(rows, displayRow{r.Check.Name, label, region, style})
	}
	return rows
}
