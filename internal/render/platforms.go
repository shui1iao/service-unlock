// Direct single-probe rows; regional media retain their source sample names.
// MIT License; Copyright (c) 2026 Remnawave. See LICENSE.
package render

import "serviceunlock/internal/extra"

func (r displayRow) resultCell() string { return r.style.Render(r.result) }
func (r displayRow) regionCell() string { return r.region }

var regionTitles = map[string]string{"tw": "Taiwan", "hk": "Hong Kong", "jp": "Japan", "na": "North America", "sa": "South America", "eu": "Europe", "oc": "Oceania", "kr": "Korea", "sea": "Southeast Asia", "af": "Africa"}

func (o *Output) groups(r Report) []displayGroup {
	var buckets [len(purposeTitles)][]displayRow
	var regional []displayRow
	local := map[string]bool{}
	if _, ok := regionTitles[r.Region]; ok {
		global, _ := extra.Selected("global")
		for _, c := range global {
			local[c.ID] = false
		}
		selected, _ := extra.Selected(r.Region)
		for _, c := range selected {
			if _, global := local[c.ID]; !global {
				local[c.ID] = true
			}
		}
	}
	add := func(source, id string, row displayRow) {
		p := presentationFor(source, id)
		if source == "extra" && local[id] && (p.purpose == video || p.purpose == music) {
			regional = append(regional, row)
		} else {
			buckets[p.purpose] = append(buckets[p.purpose], row)
		}
	}
	for i, row := range o.serviceRows(r.Services) {
		add("geo", r.Services[i].Check.ID, row)
	}
	for i, row := range o.accessRows(r.Access) {
		add("access", r.Access[i].Check.ID, row)
	}
	for i, row := range o.aiRows(r.AI) {
		add("ai", r.AI[i].Check.ID, row)
	}
	for i, row := range o.extraRows(r.Extra) {
		add("extra", r.Extra[i].ID, row)
	}
	var groups []displayGroup
	for i, rows := range buckets {
		if len(rows) > 0 {
			groups = append(groups, displayGroup{purposeTitles[i], rows})
		}
	}
	if len(regional) > 0 {
		groups = append(groups, displayGroup{"Regional media — " + regionTitles[r.Region], regional})
	}
	return groups
}
