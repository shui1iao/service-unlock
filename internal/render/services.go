// Single system-selected outcome adapter for geocheck's consumer-service group.
// MIT License; Copyright (c) 2026 Remnawave.
package render

import (
	"fmt"
	"serviceunlock/internal/geo"
	"strings"
)

func (o *Output) servicesTable(r Report) {
	o.section("Popular services")
	rows := make([][]string, 0, len(r.Services))
	observed, unknown := 0, 0
	for _, res := range r.Services {
		kind := res.Check.Kind.String()
		if res.Check.Kind == geo.KindBlocked {
			kind = "captcha/block"
		}
		value, detail, elapsed := res.Value, res.Detail, o.rttCell(res.RTT)
		if res.Err != nil {
			value = o.Theme.Error.Render("○ error")
			detail = res.Err.Error()
			elapsed = "—"
			unknown++
		} else {
			observed++
			if res.Check.Kind == geo.KindAvailability {
				if value == "yes" {
					value = o.Theme.Good.Render("● yes")
				} else {
					value = o.Theme.Bad.Render("● no")
				}
			}
			if res.Check.Kind == geo.KindBlocked {
				if value == "yes" {
					value = o.Theme.Bad.Render("● yes")
				} else {
					value = o.Theme.Good.Render("● no")
				}
			}
		}
		rows = append(rows, []string{res.Check.Name, kind, value, clip(detail, 52), elapsed})
	}
	o.line("%s", o.table([]string{"Service", "Type", "Result", "Detail", "Time"}, rows))
	var parts []string
	if observed > 0 {
		parts = append(parts, fmt.Sprintf("%d observations", observed))
	}
	if unknown > 0 {
		parts = append(parts, o.Theme.Error.Render(fmt.Sprintf("%d inconclusive", unknown)))
	}
	if len(parts) > 0 {
		o.line("  %s", strings.Join(parts, " · "))
	}
	o.line("")
}
