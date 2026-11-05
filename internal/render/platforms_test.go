// MIT License; Copyright (c) 2026 Remnawave. Offline platform regressions.
package render

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/extra"
	"serviceunlock/internal/geo"
)

func TestPlatformConflictsSourceRegionsAndNoCountryBackfill(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	r := Report{
		Services: []geo.Result{
			{Check: geo.Check{ID: "chatgpt", Kind: geo.KindCountry}, Value: "JP"},
			{Check: geo.Check{ID: "netflix", Kind: geo.KindCountry}, Value: "US"},
			{Check: geo.Check{ID: "prime", Kind: geo.KindCountry}, Err: errors.New("geo timeout")},
			{Check: geo.Check{ID: "spotify", Kind: geo.KindCountry}, Value: "JP"},
			{Check: geo.Check{ID: "spotify_signup", Kind: geo.KindAvailability}, Err: errors.New("inconclusive legacy endpoint")},
			{Check: geo.Check{ID: "google", Kind: geo.KindCountry}, Value: "US"},
			{Check: geo.Check{ID: "google_captcha", Kind: geo.KindBlocked}, Value: "yes"},
		},
		Access: []access.Result{
			{Check: access.Check{ID: "chatgpt_web"}, State: access.StateAvailable},
			{Check: access.Check{ID: "netflix_access"}, State: access.StateRestricted, Region: "JP"},
			{Check: access.Check{ID: "claude_access"}, State: access.StateAvailable},
			{Check: access.Check{ID: "gemini_access"}, State: access.StateBlocked, Region: "CHN"},
		},
		AI: []ai.Result{
			{Check: ai.Check{ID: "openai"}, State: ai.StateBlocked},
			{Check: ai.Check{ID: "anthropic"}, State: ai.StateReachable},
			{Check: ai.Check{ID: "gemini"}, State: ai.StateReachable},
			{Check: ai.Check{ID: "qwen_intl"}, State: ai.StateReachable},
			{Check: ai.Check{ID: "qwen_cn"}, State: ai.StateError},
		},
	}
	jp := "JP"
	r.Extra = []extra.Result{{ID: "MediaUnlockTest_PrimeVideo", Status: "available", Region: &jp}}
	groups := o.groups(r)
	for name, want := range map[string][2]string{
		"ChatGPT":            {"Web ● Yes · API ● No", "JP"},
		"Netflix":            {"Catalogue ● Limited", "Catalogue JP · Fast US"},
		"Amazon Prime Video": {"Access ● Yes", "Access JP · Geo Unknown"},
		"Spotify":            {"Legacy signup ○ Unknown", "JP"},
		"Google":             {"Search ● No", "US"}, // Captcha=yes is NOT access Yes.
		"Claude":             {"Web ● Yes · API ● Yes", "Geo Unknown"},
		"Gemini":             {"Web ● No · API ● Yes", "CN"},
		"Qwen":               {"API SG ● Yes · API CN ○ Unknown", ""},
	} {
		row := platformNamed(t, groups, name)
		if row.result != want[0] || row.region != want[1] {
			t.Fatal(name, row.result, row.region, want)
		}
	}
	row := platformNamed(t, groups, "ChatGPT")
	colored := row.resultCell()
	if ansi.Strip(colored) != row.result || !strings.Contains(colored, "\x1b[1;38;5;2m● Yes\x1b[m") || !strings.Contains(colored, "\x1b[1;38;5;1m● No\x1b[m") {
		t.Fatal("independent status colours lost", colored)
	}
	// A successful other-source country must not fill a missing Netflix catalogue.
	r.Access[1].Region = ""
	r.Access[1].State = access.StateBlocked
	row = platformNamed(t, o.groups(r), "Netflix")
	if row.result != "Catalogue ● No" || row.region != "Catalogue Unknown · Fast US" {
		t.Fatal(row)
	}
	// Both observations stay explicitly qualified even when their country agrees.
	r.Access[1].Region = "US"
	row = platformNamed(t, o.groups(r), "Netflix")
	if row.region != "Catalogue US · Fast US" {
		t.Fatal(row)
	}
	// A nil/empty optional extra country creates no phantom failed region probe.
	r.Extra[0].Region = nil
	row = platformNamed(t, o.groups(r), "Amazon Prime Video")
	if row.region != "Geo Unknown" {
		t.Fatal(row)
	}
}

func TestCountryOnlyNeverLooksLikeAnUnlockVerdict(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	for _, tc := range []struct{ id, cap string }{{"twitch", "Region"}, {"apple", "Region"}, {"steam", "Store region"}, {"playstation", "Region"}, {"deezer", "Region"}, {"ookla", "Region"}, {"jetbrains", "Region"}, {"bing", "Region"}} {
		r := Report{Services: []geo.Result{{Check: geo.Check{ID: tc.id, Kind: geo.KindCountry}, Value: "JP"}}}
		row := o.groups(r)[0].rows[0]
		if row.result != tc.cap+" ● Yes" || row.region != "JP" {
			t.Fatal(tc, row)
		}
		r.Services[0].Err = errors.New("timeout with a retained observed country")
		row = o.groups(r)[0].rows[0]
		if row.result != tc.cap+" ○ Unknown" || row.region != "Geo JP (Unknown)" {
			t.Fatal("country error erased", tc, row)
		}
		r.Services[0].Value = "ZZ"
		row = o.groups(r)[0].rows[0]
		if row.result != tc.cap+" ○ Unknown" || row.region != "Geo Unknown" {
			t.Fatal(tc, row)
		}
	}
}

func TestEverySelectedScopeHasOneRowPerPlatform(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	counts := map[string]int{"global": 28, "tw": 39, "hk": 36, "jp": 54, "na": 65, "sa": 31, "eu": 56, "oc": 47, "kr": 36, "sea": 36, "af": 31}
	for _, region := range extra.Regions() {
		r := completeMock(t)
		checks, err := extra.Selected(region)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range checks {
			r.Extra = append(r.Extra, extra.Result{ID: c.ID, Name: c.Name, Status: "error"})
		}
		seen := map[string]bool{}
		for _, g := range o.groups(r) {
			for _, row := range g.rows {
				if seen[row.service] {
					t.Fatal(region, "duplicate platform", row.service)
				}
				seen[row.service] = true
			}
		}
		if len(seen) != counts[region] {
			t.Fatal(region, len(seen), counts[region])
		}
	}
}

// All regional Sky samples share a platform but never share a verdict/country.
func TestSkyMarketStatesAndCountriesRemainIndependent(t *testing.T) {
	r := Report{}
	for i, id := range []string{"MediaUnlockTest_SkyGo", "MediaUnlockTest_SKY_DE", "MediaUnlockTest_Sky_CH", "MediaUnlockTest_SkyGONZ"} {
		code := []string{"GB", "DE", "CH", "NZ"}[i]
		r.Extra = append(r.Extra, extra.Result{ID: id, Status: []string{"available", "blocked", "restricted", "error"}[i], Region: &code})
	}
	groups := (&Output{Theme: newTheme(nil)}).groups(r)
	row := platformNamed(t, groups, "Sky")
	if len(groups) != 1 || len(groups[0].rows) != 1 || row.result != "Go UK ● Yes · DE ● No · CH ● Limited · Go NZ ○ Unknown" || row.region != "Go UK GB · DE DE · CH CH · Go NZ NZ" {
		t.Fatal(row)
	}
}
