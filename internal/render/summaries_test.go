package render

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/geo"
)

// Only table content: detailed group summaries intentionally contain words
// such as available/reachable, but these must not reappear as per-row labels.
func tableText(text string) string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "│") {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
func TestDetailedServiceSummariesAndReverseCaptcha(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	fixtures := []struct {
		kind  geo.Kind
		value string
		err   error
	}{
		{geo.KindCountry, "US", nil}, {geo.KindCountry, "ESP", nil},
		{geo.KindAvailability, "yes", nil}, {geo.KindBlocked, "no", nil},
		{geo.KindAvailability, "no", nil}, {geo.KindBlocked, "yes", nil},
		{geo.KindCountry, "HK", errors.New("mock timeout")},
		{geo.KindCountry, "WW", nil}, {geo.KindAvailability, "yes", errors.New("mock HTTP 500")},
	}
	var results []geo.Result
	for _, f := range fixtures {
		results = append(results, geo.Result{Check: geo.Check{Kind: f.kind, Name: "Fixture"}, Value: f.value, Err: f.err})
	}
	rows := o.serviceRows(results)
	want := "Yes 4: 2 regions identified, 1 accessible, 1 without restriction markers\nNo 2: 1 access denied, 1 with restriction markers\nUnknown 3: inconclusive"
	if actual := serviceSummary(rows, results); actual != want {
		t.Fatalf("got %q want %q", actual, want)
	}
	if rows[3].result != "● Yes" || rows[5].result != "● No" {
		t.Fatal("raw captcha direction changed")
	}
	if rows[6].result != "○ Unknown" || rows[6].region != "HK" {
		t.Fatal("country code overrode unknown", rows[6])
	}
	// Singular grammar and omission of zero categories.
	one := results[:1]
	if got := serviceSummary(o.serviceRows(one), one); got != "Yes 1: 1 region identified" {
		t.Fatal(got)
	}
	no := results[4:5]
	if got := serviceSummary(o.serviceRows(no), no); got != "No 1: 1 access denied" {
		t.Fatal(got)
	}
}
func TestDetailedAvailabilitySummariesUseDisplayedStatus(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	results := []access.Result{
		{State: access.StateAvailable},
		{State: access.StateBlocked, Region: "USA"},
		{Check: access.Check{ID: "netflix_access"}, State: access.StateRestricted, Region: "FRA"},
		{Check: access.Check{ID: "other"}, State: access.StateRestricted},
		{State: access.StateBlocked, Region: "JPN", Err: errors.New("timeout")},
		{State: access.StateError, Region: "HK"},
	}
	rows := o.accessRows(results)
	want := "Yes 1: available\nNo 1: blocked\nLimited 2: 1 originals-only, 1 restricted\nUnknown 2: inconclusive"
	if got := accessSummary(rows, results); got != want {
		t.Fatalf("%q", got)
	}
	if rows[1].result != "● No" || rows[1].region != "US" || rows[4].result != "○ Unknown" || rows[4].region != "JP" {
		t.Fatal(rows)
	}
	for _, tc := range []struct {
		r    access.Result
		want string
	}{
		{results[2], "Limited 1: originals-only"}, {results[3], "Limited 1: restricted"}, {results[4], "Unknown 1: inconclusive"},
	} {
		r := []access.Result{tc.r}
		if got := accessSummary(o.accessRows(r), r); got != tc.want {
			t.Fatal(got)
		}
	}
}
func TestDetailedAPISummaryNeverClaimsInvocationSuccess(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	rows := o.aiRows([]ai.Result{{State: ai.StateReachable, Status: 401}, {State: ai.StateBlocked, Status: 403}, {State: ai.StateError}, {State: ai.StateReachable, Err: errors.New("timeout")}})
	want := "Yes 1: API reachable\nNo 1: region/request refused\nUnknown 2: inconclusive"
	if got := aiSummary(rows); got != want {
		t.Fatal(got)
	}
	for _, row := range rows {
		if row.region != "" {
			t.Fatal("API node inferred country")
		}
	}
}
func TestDetailedSummaryTotalsMatchAll35Rows(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	pattern := regexp.MustCompile(`^(Yes|No|Limited|Unknown) ([0-9]+): (.+)$`)
	for _, negative := range []bool{false, true} {
		groups := o.groups(englishFixture(t, negative))
		total := 0
		if len(groups) != 3 {
			t.Fatal("group count")
		}
		for _, g := range groups {
			expected := map[string]int{}
			for _, row := range g.rows {
				expected[rowLabel(row)]++
			}
			got := map[string]int{}
			sum := 0
			for _, line := range strings.Split(g.summary, "\n") {
				m := pattern.FindStringSubmatch(line)
				if m == nil {
					t.Fatal("unclear summary", line)
				}
				if _, exists := got[m[1]]; exists {
					t.Fatal("duplicate status")
				}
				n, _ := strconv.Atoi(m[2])
				if n == 0 {
					t.Fatal("zero category")
				}
				got[m[1]] = n
				sum += n
			}
			if !reflect.DeepEqual(got, expected) || sum != len(g.rows) {
				t.Fatal(g.title, got, expected)
			}
			total += sum
		}
		if total != 35 {
			t.Fatal(total)
		}
	}
}
func TestDetailedSummaryWrapPreservesAllMeaning(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	for _, g := range o.groups(englishFixture(t, true)) {
		for _, width := range []int{14, 18, 20, 24, 30, 40, 80, 100} {
			wrapped := wrapSummary(g.summary, width)
			if compact(wrapped) != compact(g.summary) {
				t.Fatal("summary meaning lost", width, wrapped)
			}
			for _, line := range strings.Split(wrapped, "\n") {
				if ansi.StringWidth(line) > width {
					t.Fatal("overflow", width, line)
				}
			}
		}
	}
}
func TestRegionColumnsContainCodesNotNames(t *testing.T) {
	text := ansi.Strip(renderProfile(t, englishFixture(t, true), colorprofile.TrueColor))
	for _, name := range []string{"United States", "France", "Hong Kong", "Japan", "China"} {
		if strings.Contains(tableText(text), name) {
			t.Fatal("country name remains", name)
		}
	}
	for _, code := range []string{"US", "HK", "JP", "FR", "CN"} {
		if !strings.Contains(tableText(text), " "+code+" ") {
			t.Fatal("country code missing", code)
		}
	}
}
