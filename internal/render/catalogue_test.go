// MIT License; Copyright (c) 2026 Remnawave. Offline fixture assertions only.
package render

import (
	"reflect"
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
	record := func(source, id string) {
		key := source + ":" + id
		if seen[key] {
			t.Fatal("duplicate measurement", key)
		}
		seen[key] = true
		counts[presentationFor(source, id).purpose]++
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
	if len(seen) != 168 || safety != 9 || counts != [6]int{127, 8, 13, 10, 3, 7} {
		t.Fatal(len(seen), safety, counts)
	}
	o := &Output{Theme: newTheme(nil)}
	groups := o.groups(r)
	if len(groups) != 6 {
		t.Fatal("missing purpose group", groups)
	}
	for i, group := range groups {
		if group.title != purposeTitles[i] || len(group.rows) != counts[i] {
			t.Fatal("lost/misclassified rows", group.title, len(group.rows), counts[i])
		}
	}
}

func TestPurposeStablePlatformAdjacencyAndCapabilityLabels(t *testing.T) {
	r := completeMock(t)
	r.Extra = []extra.Result{
		{ID: "MediaUnlockTest_Dazn", Name: "Dazn"},
		{ID: "MediaUnlockTest_PrimeVideo", Name: "Amazon Prime Video (access)"},
		{ID: "WebTest_MetaAI", Name: "Meta AI"},
		{ID: "MediaUnlockTest_DMM", Name: "DMM"},
	}
	want := [][]string{
		{"YouTube (region)", "YouTube Premium", "Twitch (region)", "Netflix (Fast region)", "Netflix (catalogue)", "Amazon Prime Video (region)", "Amazon Prime Video (access)", "TikTok (region)", "TikTok (Web)", "Dazn"},
		{"YouTube Music", "Spotify (region)", "Spotify (legacy signup endpoint)", "Deezer (region)"},
		{"ChatGPT (region)", "ChatGPT (Web)", "OpenAI (API)", "Claude (Web)", "Anthropic (API)", "Gemini (Web)", "Google Gemini (API)", "DeepSeek (API)", "Qwen (Singapore) (API)", "Qwen (Beijing) (API)", "Kimi / Moonshot (API)", "Zhipu GLM (API)", "Meta AI (Web)"},
		{"Steam (store region)", "PlayStation (region)"},
		{"Reddit (region)", "Reddit (guest access)"},
		{"Google (region)", "Google Search captcha", "Apple (region)", "Ookla Speedtest (region)", "JetBrains (region)", "Microsoft (Bing) (region)", "DMM (financial site)"},
	}
	o := &Output{Theme: newTheme(nil)}
	groups := o.groups(r)
	for i, group := range groups {
		got := []string{}
		for _, row := range group.rows {
			got = append(got, row.service)
		}
		if !reflect.DeepEqual(got, want[i]) {
			t.Fatal(group.title, got, want[i])
		}
	}
	// The input's raw names and order must not be rewritten to implement labels.
	if r.Services[0].Check.Name != "Google" || r.Access[0].Check.Name != "ChatGPT (web)" || r.Extra[2].Name != "Meta AI" {
		t.Fatal("mutated raw names")
	}
}

func TestPurposeEmptyGroupsOmittedAndSamplesStayIndependent(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	r := Report{}
	for _, id := range []string{"MediaUnlockTest_BGlobalSEA", "MediaUnlockTest_BGlobalTH", "MediaUnlockTest_BGlobalID", "MediaUnlockTest_BGlobalVN"} {
		r.Extra = append(r.Extra, extra.Result{ID: id, Name: id, Status: "error"})
	}
	groups := o.groups(r)
	if len(groups) != 1 || groups[0].title != "Video, live & sports" || len(groups[0].rows) != 4 {
		t.Fatal("empty groups or lost copyright sample", groups)
	}
	for i, row := range groups[0].rows {
		if row.service != r.Extra[i].Name || row.result != "○ Unknown" {
			t.Fatal(row)
		}
	}
}
