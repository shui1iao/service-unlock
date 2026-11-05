// MIT License; Copyright (c) 2026 Remnawave. Offline platform assertions.
package render

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/extra"
	"serviceunlock/internal/geo"
)

func TestPurposeCatalogueCoversAll168IDsIncludingSafetyUnknowns(t *testing.T) {
	r := Report{}
	counts := [len(purposeTitles)]int{}
	seen := map[string]bool{}
	platforms := map[string]presentation{}
	record := func(source, id string) {
		key := source + ":" + id
		if seen[key] {
			t.Fatal("duplicate measurement", key)
		}
		seen[key] = true
		p := presentationFor(source, id)
		if p.name == "" {
			t.Fatal("unmapped current measurement", key)
		}
		if before, ok := platforms[p.platform]; ok && (p.purpose != before.purpose || p.name != before.name) {
			t.Fatal("platform has multiple purposes/names", key)
		}
		platforms[p.platform] = p
		counts[p.purpose]++
	}
	for _, c := range geo.ServiceChecks() {
		if c.ID == "youtube_premium" {
			t.Fatal("retired probe still active")
		}
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
	safety := 0
	for _, c := range extra.Catalogue() {
		record("extra", c.ID)
		r.Extra = append(r.Extra, extra.Result{ID: c.ID, Name: c.Name, Status: "error"})
		if c.Safety != "" {
			safety++
		}
	}
	if len(seen) != 168 || safety != 9 || counts != [6]int{127, 8, 13, 10, 3, 7} || len(platforms) != 145 {
		t.Fatal(len(seen), safety, counts, len(platforms))
	}
	o := &Output{Theme: newTheme(nil)}
	groups := o.groups(r)
	wantRows := [6]int{112, 7, 8, 10, 2, 6}
	if len(groups) != 6 {
		t.Fatal("missing group", groups)
	}
	emitted := map[string]bool{}
	for i, g := range groups {
		if g.title != purposeTitles[i] || len(g.rows) != wantRows[i] {
			t.Fatal(g.title, len(g.rows), wantRows[i])
		}
		for _, row := range g.rows {
			if emitted[row.service] {
				t.Fatal("platform emitted twice", row.service)
			}
			emitted[row.service] = true
		}
	}
	for _, c := range extra.Catalogue() {
		if c.Safety == "" {
			continue
		}
		p := presentationFor("extra", c.ID)
		row := platformNamed(t, groups, p.name)
		want := cellPart{label: p.capability, value: "○ Unknown"}.plain()
		if !strings.Contains(row.result, want) {
			t.Fatal("safety Unknown lost", c.ID, row)
		}
	}
	// Raw names/statuses/IDs/order stay byte-identical even for safety rows.
	before, _ := json.Marshal(r.Extra)
	o.groups(r)
	after, _ := json.Marshal(r.Extra)
	if string(before) != string(after) {
		t.Fatal("mutated extra raw")
	}
}

func platformNamed(t *testing.T, groups []displayGroup, name string) displayRow {
	t.Helper()
	var found []displayRow
	for _, g := range groups {
		for _, row := range g.rows {
			if row.service == name {
				found = append(found, row)
			}
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s: want one logical row, got%d", name, len(found))
	}
	return found[0]
}

func TestPlatformFirstAppearanceAndCanonicalNames(t *testing.T) {
	r := completeMock(t)
	r.Extra = []extra.Result{
		{ID: "MediaUnlockTest_Dazn", Name: "Dazn"},
		{ID: "MediaUnlockTest_PrimeVideo", Name: "Amazon Prime Video (access)"},
		{ID: "WebTest_MetaAI", Name: "Meta AI"},
		{ID: "MediaUnlockTest_DMM", Name: "DMM"},
	}
	want := [][]string{
		{"YouTube", "Twitch", "Netflix", "Amazon Prime Video", "TikTok", "Dazn"},
		{"YouTube Music", "Spotify", "Deezer"},
		{"ChatGPT", "Claude", "Gemini", "DeepSeek", "Qwen", "Kimi / Moonshot", "Zhipu GLM", "Meta AI"},
		{"Steam", "PlayStation"}, {"Reddit"},
		{"Google", "Apple", "Ookla Speedtest", "JetBrains", "Microsoft (Bing)", "DMM (financial site)"},
	}
	groups := (&Output{Theme: newTheme(nil)}).groups(r)
	for i, g := range groups {
		var got []string
		for _, row := range g.rows {
			got = append(got, row.service)
		}
		if !reflect.DeepEqual(got, want[i]) {
			t.Fatal(g.title, got, want[i])
		}
	}
	if r.Services[0].Check.Name != "Google" || r.Access[0].Check.Name != "ChatGPT (web)" || r.Extra[2].Name != "Meta AI" {
		t.Fatal("mutated raw names")
	}
}

func TestAll16FamiliesUseOneRowAndRetainEveryCapability(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	r := completeMock(t)
	// All markets are intentionally together in an offline display fixture.
	for _, c := range extra.Catalogue() {
		r.Extra = append(r.Extra, extra.Result{ID: c.ID, Name: c.Name, Status: "error"})
	}
	groups := o.groups(r)
	for name, labels := range map[string][]string{
		"ChatGPT": {"Web", "API"}, "Claude": {"Web", "API"}, "Gemini": {"Web", "API"},
		"Qwen": {"API SG", "API CN"}, "Google": {"Search"}, "YouTube": {"Premium"},
		"Netflix": {"Catalogue"}, "TikTok": {"Web"}, "Amazon Prime Video": {"Access"},
		"Spotify": {"Legacy signup"}, "Reddit": {"Guest"},
		"Bilibili": {"TW", "HK-MO-TW", "Global SEA", "Global TH", "Global ID", "Global VN"},
		"Sky":      {"Go UK", "DE", "CH", "Go NZ"}, "Hulu": {"US", "JP"},
		"Rakuten TV": {"EU", "JP"}, "DIRECTV": {"Stream", "Go"},
	} {
		row := platformNamed(t, groups, name)
		if len(row.resultParts) != len(labels) {
			t.Fatal("lost/extra capability", name, row.result)
		}
		for i, label := range labels {
			// Bilibili's two metadata entries precede SEA in the opposite market order;
			// identity is deterministic but capabilities retain actual input order.
			found := false
			for _, part := range row.resultParts {
				if part.label == label {
					found = true
				}
			}
			if !found {
				t.Fatal(name, i, label, row.result)
			}
		}
	}
	for _, g := range groups {
		for _, row := range g.rows {
			if row.service == "OpenAI" || row.service == "Anthropic" {
				t.Fatal("API alias emitted as another platform", row.service)
			}
		}
	}
	for _, name := range []string{"YouTube Music", "SkyShowTime", "DMM (financial site)", "DMM TV", "Viu.TV", "Viu.com", "TVBAnywhere+", "MyTVSuper", "encoreTVB"} {
		platformNamed(t, groups, name)
	}
}

func TestPurposeEmptyGroupsAndCopyrightSamplesKeepSeparateResults(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	r := Report{}
	for i, id := range []string{"MediaUnlockTest_BGlobalSEA", "MediaUnlockTest_BGlobalTH", "MediaUnlockTest_BGlobalID", "MediaUnlockTest_BGlobalVN"} {
		r.Extra = append(r.Extra, extra.Result{ID: id, Name: id, Status: []string{"available", "blocked", "restricted", "error"}[i]})
	}
	groups := o.groups(r)
	if len(groups) != 1 || groups[0].title != "Video, live & sports" || len(groups[0].rows) != 1 {
		t.Fatal(groups)
	}
	row := groups[0].rows[0]
	if row.service != "Bilibili" || row.result != "Global SEA ● Yes · Global TH ● No · Global ID ● Limited · Global VN ○ Unknown" || row.region != "" {
		t.Fatal(row)
	}
}

func TestUnknownCallersSourceQualifiedVisibleAndNotGuessed(t *testing.T) {
	r := Report{Extra: []extra.Result{
		{ID: "MediaUnlockTest_NewSky", Name: "Sky", Status: "available"},
		{ID: "MediaUnlockTest_NewSky", Name: "Other", Status: "blocked"},
	}}
	groups := (&Output{Theme: newTheme(nil)}).groups(r)
	if len(groups) != 1 || groups[0].title != "General websites" || len(groups[0].rows) != 2 {
		t.Fatal(groups)
	}
	for i, row := range groups[0].rows {
		if !strings.HasPrefix(row.service, "extra:MediaUnlockTest_NewSky (") || !strings.Contains(row.service, r.Extra[i].Name) {
			t.Fatal(row)
		}
	}
}
