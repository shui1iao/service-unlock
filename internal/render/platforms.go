// Platform-only terminal aggregation; original source outcomes remain independent.
// MIT License; Copyright (c) 2026 Remnawave. See LICENSE.
package render

import (
	"strings"

	lg "charm.land/lipgloss/v2"
	"serviceunlock/internal/geo"
)

// Each status has its own style. Labels and separators retain the default
// foreground; no platform-wide verdict or style is computed.
type cellPart struct {
	label, value string
	style        lg.Style
}

func (p cellPart) plain() string {
	if p.label == "" {
		return p.value
	}
	return p.label + " " + p.value
}
func joinParts(parts []cellPart, styled bool) string {
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if styled {
			value := part.style.Render(part.value)
			if part.label != "" {
				value = part.label + " " + value
			}
			values = append(values, value)
		} else {
			values = append(values, part.plain())
		}
	}
	return strings.Join(values, " · ")
}
func (r displayRow) resultCell() string {
	if r.resultParts != nil {
		return joinParts(r.resultParts, true)
	}
	return r.style.Render(r.result)
}
func (r displayRow) regionCell() string {
	if r.regionParts != nil {
		return joinParts(r.regionParts, true)
	}
	return r.region
}

type regionPart struct {
	cellPart
	first, forceLabel bool
}
type platformRow struct {
	row                displayRow
	results, countries []cellPart
	regions            []regionPart
}

// groups is the shared normal/machine terminal path. It builds exactly one
// logical displayRow per explicit platform. Traversal/first platform order is
// geo/access/AI/extra, independent of execution, selection or raw array order.
func (o *Output) groups(r Report) []displayGroup {
	var buckets [len(purposeTitles)][]*platformRow
	platforms := map[string]*platformRow{}
	add := func(source, id string, row displayRow, country, hasRegion bool) {
		p := presentationFor(source, id)
		target := platforms[p.platform]
		// No mapped identity means no authority to combine caller rows, even
		// when the caller omitted/reused an ID. Keep each fallback visible.
		if p.name == "" {
			target = nil
		}
		if target == nil {
			target = &platformRow{row: displayRow{service: row.service}}
			platforms[p.platform] = target
			buckets[p.purpose] = append(buckets[p.purpose], target)
		}
		label := p.capability
		if country {
			// A country success is not an availability vote. Only a country-only
			// platform needs this explicitly qualified status in its Result cell.
			if label == "" {
				label = "Region"
			}
			target.countries = append(target.countries, cellPart{label, row.result, row.style})
		} else {
			target.results = append(target.results, cellPart{label, row.result, row.style})
		}
		if !hasRegion && !p.regionExpected {
			return
		}
		regionLabel := p.regionLabel
		if regionLabel == "" {
			regionLabel = "Geo"
		}
		value, style := row.region, lg.NewStyle()
		if value == "" {
			value, style = "Unknown", o.Theme.Error
		} else if country && row.result == "○ Unknown" {
			// Preserve both an observed code and the source's failed/inconclusive
			// status. Another capability can neither erase the error nor fill it.
			value, style = value+" (Unknown)", o.Theme.Error
		}
		target.regions = append(target.regions, regionPart{
			cellPart: cellPart{regionLabel, value, style}, first: p.regionFirst,
			forceLabel: p.platform == "netflix" || p.platform == "prime",
		})
	}
	for i, row := range o.serviceRows(r.Services) {
		country := r.Services[i].Check.Kind == geo.KindCountry
		add("geo", r.Services[i].Check.ID, row, country, country)
	}
	for i, row := range o.accessRows(r.Access) {
		add("access", r.Access[i].Check.ID, row, false, r.Access[i].Region != "")
	}
	for i, row := range o.aiRows(r.AI) {
		add("ai", r.AI[i].Check.ID, row, false, false)
	}
	for i, row := range o.extraRows(r.Extra) {
		// Extra country fields are optional: absent/empty is not a separate
		// failed region check. An explicitly provided invalid code stays Unknown.
		hasRegion := r.Extra[i].Region != nil && *r.Extra[i].Region != ""
		add("extra", r.Extra[i].ID, row, false, hasRegion)
	}
	var groups []displayGroup
	for category, rows := range buckets {
		if len(rows) == 0 {
			continue
		}
		group := displayGroup{title: purposeTitles[category]}
		for _, platform := range rows {
			row := platform.row
			row.resultParts = platform.results
			if len(row.resultParts) == 0 {
				row.resultParts = platform.countries
			}
			row.result = joinParts(row.resultParts, false)
			// The two explicitly ordered country pairs are Catalogue/Fast and
			// Access/Geo. Stable partitioning never changes values or combines them.
			for _, first := range []bool{true, false} {
				for _, region := range platform.regions {
					if region.first != first {
						continue
					}
					part := region.cellPart
					if len(platform.regions) == 1 && !region.forceLabel && !strings.Contains(part.value, "Unknown") {
						part.label = ""
					}
					row.regionParts = append(row.regionParts, part)
				}
			}
			row.region = joinParts(row.regionParts, false)
			group.rows = append(group.rows, row)
		}
		groups = append(groups, group)
	}
	return groups
}
