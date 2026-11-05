// MIT License; Copyright (c) 2026 Remnawave. Offline single-probe directory tests.
package render

import (
	"encoding/json"
	"reflect"
	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/extra"
	"serviceunlock/internal/geo"
	"testing"
)

func platformNamed(t *testing.T, groups []displayGroup, name string) displayRow {
	t.Helper()
	var found []displayRow
	for _, g := range groups {
		for _, r := range g.rows {
			if r.service == name {
				found = append(found, r)
			}
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s: want one row, got %d", name, len(found))
	}
	return found[0]
}
func TestPurposeCatalogueCovers156DistinctMeasurementsIncludingSafety(t *testing.T) {
	r := Report{}
	seen := map[string]bool{}
	safety := 0
	record := func(source, id string) {
		key := source + ":" + id
		if seen[key] || presentationFor(source, id).name == "" {
			t.Fatal(key)
		}
		seen[key] = true
	}
	for _, c := range geo.ServiceChecks() {
		record("geo", c.ID)
		r.Services = append(r.Services, geo.Result{Check: c})
	}
	for _, c := range access.Checks() {
		record("access", c.ID)
		r.Access = append(r.Access, access.Result{Check: c})
	}
	for _, c := range ai.Checks() {
		record("ai", c.ID)
		r.AI = append(r.AI, ai.Result{Check: c})
	}
	for _, c := range extra.Catalogue() {
		record("extra", c.ID)
		r.Extra = append(r.Extra, extra.Result{ID: c.ID, Name: c.Name, Status: "error"})
		if c.Safety != "" {
			safety++
		}
	}
	if len(seen) != 156 || safety != 9 {
		t.Fatal(len(seen), safety)
	}
	before, _ := json.Marshal(r)
	groups := (&Output{Theme: newTheme(nil)}).groups(r)
	after, _ := json.Marshal(r)
	if string(before) != string(after) {
		t.Fatal("render mutated raw")
	}
	n := 0
	for _, g := range groups {
		n += len(g.rows)
	}
	if n != 156 {
		t.Fatal("hidden or aggregated probes", n)
	}
	for _, c := range extra.Catalogue() {
		if c.Safety != "" {
			row := platformNamed(t, groups, terminalName("extra", c.ID, c.Name))
			if row.result != "○ Unknown" {
				t.Fatal("unsafe became success", row)
			}
		}
	}
}
func TestEveryRegionSelectedOrderAndOriginalMediaNames(t *testing.T) {
	totals := map[string]int{"global": 28, "tw": 39, "hk": 36, "jp": 54, "na": 65, "sa": 31, "eu": 58, "oc": 47, "kr": 36, "sea": 39, "af": 31}
	global, _ := extra.Selected("global")
	globals := map[string]bool{}
	for _, c := range global {
		globals[c.ID] = true
	}
	for _, region := range extra.Regions() {
		selected, _ := extra.Selected(region)
		if len(selected)+22 != totals[region] {
			t.Fatal(region, len(selected))
		}
		r := completeMock(t)
		r.Region = region
		want := []string{}
		for _, c := range selected {
			r.Extra = append(r.Extra, extra.Result{ID: c.ID, Name: c.Name, Status: "error"})
			p := presentationFor("extra", c.ID)
			if !globals[c.ID] && (p.purpose == video || p.purpose == music) {
				want = append(want, terminalName("extra", c.ID, c.Name))
			}
		}
		groups := (&Output{Theme: newTheme(nil)}).groups(r)
		got := []string{}
		n := 0
		for _, g := range groups {
			n += len(g.rows)
			if g.title == "Regional media — "+regionTitles[region] {
				for _, row := range g.rows {
					got = append(got, row.service)
				}
			}
		}
		if n != totals[region] || !reflect.DeepEqual(got, want) {
			t.Fatal(region, n, got, want)
		}
		if region == "global" && len(groups) != 6 {
			t.Fatal("phantom regional", groups)
		}
		if len(want) > 0 && groups[len(groups)-1].title != "Regional media — "+regionTitles[region] {
			t.Fatal("regional missing", region)
		}
	}
}
func TestUnknownCallersStaySourceQualified(t *testing.T) {
	r := Report{Extra: []extra.Result{{ID: "unknown", Name: "Netflix", Status: "error"}}}
	row := (&Output{Theme: newTheme(nil)}).groups(r)[0].rows[0]
	if row.service != "extra:unknown (Netflix)" || row.result != "○ Unknown" {
		t.Fatal(row)
	}
}
