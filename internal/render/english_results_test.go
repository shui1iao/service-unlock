package render

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/geo"
)

func englishFixture(t *testing.T, negative bool) Report {
	r := completeMock(t)
	for i := range r.Services {
		switch r.Services[i].Check.ID {
		case "google_captcha":
			if negative {
				r.Services[i].Value = "yes"
			}
		case "spotify_signup":
			if negative {
				r.Services[i].Value = "no"
				r.Services[i].Err = nil
			} else {
				r.Services[i].Value = ""
				r.Services[i].Err = errors.New("legacy signup endpoint inconclusive")
			}
		case "steam":
			r.Services[i].Value = "HK"
		case "playstation":
			r.Services[i].Value = "JP"
		}
	}
	return r
}

// Old34-row .golden files remain historical, not falsely regenerated here.
// Exact platform/capability/region cells replace the superseded row-per-probe
// snapshot; full frame/wrapping/ANSI assertions remain in the layout tests.
func TestEnglishPlatformCellsAndProfiles(t *testing.T) {
	type fixtureGroup struct {
		Title string      `json:"title"`
		Rows  [][3]string `json:"rows"`
	}
	for _, tc := range []struct {
		name     string
		negative bool
	}{{"positive", false}, {"negative", true}} {
		raw, err := os.ReadFile("testdata/platform-" + tc.name + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var want []fixtureGroup
		if err = json.Unmarshal(raw, &want); err != nil {
			t.Fatal(err)
		}
		r := englishFixture(t, tc.negative)
		var got []fixtureGroup
		for _, g := range (&Output{Theme: newTheme(nil)}).groups(r) {
			f := fixtureGroup{Title: g.title}
			for _, row := range g.rows {
				f.Rows = append(f.Rows, [3]string{row.service, row.result, row.region})
			}
			got = append(got, f)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: logical platform cells differ\ngot%+v\nwant%+v", tc.name, got, want)
		}
		plain := renderProfile(t, r, colorprofile.NoTTY)
		for _, profile := range []colorprofile.Profile{colorprofile.NoTTY, colorprofile.Ascii, colorprofile.TrueColor} {
			if actual := ansi.Strip(renderProfile(t, r, profile)); actual != plain {
				t.Fatalf("%s: profile changed layout/text", tc.name)
			}
		}
	}
}
func TestResultWholeWordsHaveDistinctColours(t *testing.T) {
	raw := renderProfile(t, englishFixture(t, true), colorprofile.TrueColor)
	// These exact contiguous runs prove the word itself, not merely the bullet,
	// is inside its intended ANSI colour/bold span in the final table output.
	for _, sequence := range []string{
		"\x1b[1;38;5;2m● Yes\x1b[m",
		"\x1b[1;38;5;1m● No\x1b[m",
		"\x1b[1;38;5;3m● Limited\x1b[m",
		"\x1b[1m○ Unknown\x1b[m",
	} {
		if !strings.Contains(raw, sequence) {
			t.Fatalf("result word not fully coloured/bold: %q", sequence)
		}
	}
	if strings.Contains(raw, "38;5;8m") || strings.Contains(raw, "38;2;") {
		t.Fatal("forced dark/truecolor foreground")
	}
}
func TestEmptyRegionsAndKnownRegionNeverOverrideStatus(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	for _, state := range []access.State{access.StateAvailable, access.StateRestricted, access.StateBlocked, access.StateError} {
		rows := o.accessRows([]access.Result{{Check: access.Check{ID: "netflix_access", Name: "Netflix"}, State: state, Region: "FRA"}})
		if rows[0].region != "FR" {
			t.Fatal("lost known region")
		}
		expected := map[access.State]string{access.StateAvailable: "● Yes", access.StateRestricted: "● Limited", access.StateBlocked: "● No", access.StateError: "○ Unknown"}[state]
		if rows[0].result != expected {
			t.Fatal("region overrode result", rows[0])
		}
	}
	unknown := o.accessRows([]access.Result{{State: access.StateBlocked, Region: "USA", Err: errors.New("timeout")}})[0]
	if unknown.result != "○ Unknown" || unknown.region != "US" {
		t.Fatal(unknown)
	}
	// Neither yes/no observation nor a provider's node name supplies a region.
	rows := append(o.serviceRows([]geo.Result{{Check: geo.Check{Name: "Spotify signup", Kind: geo.KindAvailability}, Value: "yes"}}),
		o.aiRows([]ai.Result{{Check: ai.Check{Name: "Qwen (Singapore)"}, State: ai.StateReachable}, {Check: ai.Check{Name: "Qwen (Beijing)"}, State: ai.StateBlocked}})...)
	rows = append(rows, o.accessRows([]access.Result{{Check: access.Check{Name: "Claude"}, State: access.StateError, Region: "WW"}})...)
	groups := []displayGroup{{rows: rows}}
	widths, padding := sharedWidths(groups, 100)
	table := ansi.Strip(o.threeColumnTable(rows, widths, padding))
	start := 1 + widths[0] + widths[1] + 4*padding + padding
	inBody := false
	for _, line := range strings.Split(table, "\n") {
		if strings.HasPrefix(line, "├") {
			inBody = true
			continue
		}
		if !inBody || !strings.HasPrefix(line, "│") {
			continue
		}
		if strings.TrimSpace(ansi.Cut(line, start, start+widths[2])) != "" {
			t.Fatal("region not blank", line)
		}
	}
	if strings.Contains(table, "—") || strings.Contains(table, "N/A") {
		t.Fatal("missing-region placeholder")
	}
}
