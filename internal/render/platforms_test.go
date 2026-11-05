// MIT License; Copyright (c) 2026 Remnawave. Direct rows, no capability aggregation.
package render

import (
	"errors"
	"github.com/charmbracelet/x/ansi"
	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/extra"
	"serviceunlock/internal/geo"
	"strings"
	"testing"
)

func TestSinglePrimaryStatusAndNoRegionBackfill(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	jp := "JP"
	r := Report{Services: []geo.Result{{Check: geo.Check{ID: "spotify_signup", Kind: geo.KindAvailability}, Err: errors.New("legacy endpoint inconclusive")}, {Check: geo.Check{ID: "google_captcha", Kind: geo.KindBlocked}, Value: "yes"}}, Access: []access.Result{{Check: access.Check{ID: "chatgpt_web"}, State: access.StateAvailable}, {Check: access.Check{ID: "netflix_access"}, State: access.StateRestricted, Region: "JP"}, {Check: access.Check{ID: "claude_access"}, State: access.StateBlocked}, {Check: access.Check{ID: "gemini_access"}, State: access.StateError}}, AI: []ai.Result{{Check: ai.Check{ID: "qwen_intl"}, State: ai.StateReachable}}, Extra: []extra.Result{{ID: "MediaUnlockTest_PrimeVideo", Status: "available", Region: &jp}}}
	for name, want := range map[string][2]string{"ChatGPT": {"● Yes", ""}, "Netflix": {"● Limited", "JP"}, "Claude": {"● No", ""}, "Gemini": {"○ Unknown", ""}, "Google Search (captcha)": {"● No", ""}, "Spotify (Legacy signup endpoint)": {"○ Unknown", ""}, "Qwen Intl (API)": {"● Yes", ""}, "Amazon Prime Video": {"● Yes", "JP"}} {
		row := platformNamed(t, o.groups(r), name)
		if row.result != want[0] || row.region != want[1] {
			t.Fatal(name, row, want)
		}
		if strings.Contains(row.result, " · ") || ansi.Strip(row.resultCell()) != row.result {
			t.Fatal("aggregated/mutated", row)
		}
	}
	r.Access[1].Region = ""
	row := platformNamed(t, o.groups(r), "Netflix")
	if row.region != "" || row.result != "● Limited" {
		t.Fatal("fabricated country", row)
	}
}
func TestCountryOnlyQualifiedAndBlockedMeaningRetained(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	for _, c := range geo.ServiceChecks() {
		if c.Kind != geo.KindCountry {
			continue
		}
		r := Report{Services: []geo.Result{{Check: c, Value: "JP"}}}
		row := o.groups(r)[0].rows[0]
		if !strings.Contains(row.service, "region)") || row.result != "● Yes" || row.region != "JP" {
			t.Fatal(c.ID, row)
		}
		r.Services[0].Err = errors.New("unknown")
		r.Services[0].Value = ""
		row = o.groups(r)[0].rows[0]
		if row.result != "○ Unknown" || row.region != "" {
			t.Fatal(c.ID, row)
		}
	}
}
func TestSEAOriginalCopyrightSamplesAreIndependent(t *testing.T) {
	selected, _ := extra.Selected("sea")
	r := Report{Region: "sea"}
	n := 0
	for _, c := range selected {
		status := "error"
		if strings.Contains(c.ID, "BGlobal") {
			status = []string{"available", "blocked", "restricted", "error"}[n]
			n++
		}
		r.Extra = append(r.Extra, extra.Result{ID: c.ID, Name: c.Name, Status: status})
	}
	if n != 4 {
		t.Fatal(n)
	}
	groups := (&Output{Theme: newTheme(nil)}).groups(r)
	n = 0
	for _, c := range selected {
		if !strings.Contains(c.ID, "BGlobal") {
			continue
		}
		row := platformNamed(t, groups, c.Name)
		want := []string{"● Yes", "● No", "● Limited", "○ Unknown"}[n]
		if row.result != want {
			t.Fatal(c.ID, row)
		}
		n++
	}
	for _, g := range groups {
		if strings.HasPrefix(g.title, "Regional media") {
			for _, row := range g.rows {
				if strings.Contains(row.service, "Meta AI") {
					t.Fatal("nonmedia in regional")
				}
			}
		}
	}
}
