package render

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode"

	lg "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/geo"
)

func TestCLDREnglishISO2AndISO3(t *testing.T) {
	for _, tc := range []struct{ a, b, want string }{{"US", "USA", "United States"}, {"FR", "FRA", "France"}, {"CN", "CHN", "China"}, {"GB", "GBR", "United Kingdom"}, {"JP", "JPN", "Japan"}, {"DE", "DEU", "Germany"}} {
		if got := regionName(tc.a); got != tc.want || regionName(tc.b) != got {
			t.Fatal(tc, got, regionName(tc.b))
		}
	}
	for _, code := range []string{"", "—", "WW", "ZZ", "ZZZ", "XXX", "XA", "QZ", "EU", "001", "US123", "America", "en-US", "<US>", "U S", "\x1b[31mUS"} {
		if got := regionName(code); got != "" {
			t.Fatalf("invalid region %q -> %q", code, got)
		}
	}
	if regionName(" usa ") != "United States" {
		t.Fatal("normalization")
	}
	// Exercise the library's full country/autonomous-area catalogue, not a
	// hand-picked dictionary. ISO3 canonicalization must use the same CLDR name.
	count := 0
	for _, r := range language.Supported.Regions() {
		if !r.IsCountry() || r.IsPrivateUse() {
			continue
		}
		got := regionName(r.String())
		want := display.Regions(language.English).Name(r)
		if want == "" || want == r.String() {
			if got != "" {
				t.Fatal("unnamed region invented", r, got)
			}
			continue
		}
		count++
		if got == "" || got != want {
			t.Fatalf("%s: got %q want CLDR %q", r, got, want)
		}
		hasHan := strings.IndexFunc(got, func(c rune) bool { return unicode.Is(unicode.Han, c) }) >= 0
		if hasHan {
			t.Fatalf("not English: %s %s", r, got)
		}
		if iso3 := r.ISO3(); len(iso3) == 3 && iso3 != "ZZZ" && regionName(iso3) != got {
			t.Fatalf("ISO3 mismatch %s/%s", r, iso3)
		}
	}
	t.Logf("CLDR %s: %d named country/area entries verified, including ISO3 equivalence", display.CLDRVersion, count)
	if count < 240 {
		t.Fatal("incomplete region coverage", count)
	}
}
func TestUserDefinedDisplayMappingsPreserveNegativeAndUnknown(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	for _, tc := range []struct {
		kind                geo.Kind
		value, want, region string
		err                 error
	}{
		{geo.KindCountry, "US", "● Yes", "United States", nil},
		{geo.KindCountry, "FRA", "● Yes", "France", nil},
		{geo.KindCountry, "WW", "○ Unknown", "", nil},
		{geo.KindCountry, "US", "○ Unknown", "United States", errors.New("timeout")},
		{geo.KindAvailability, "yes", "● Yes", "", nil},
		{geo.KindAvailability, "no", "● No", "", nil},
		{geo.KindAvailability, "invalid", "○ Unknown", "", nil},
		{geo.KindBlocked, "yes", "● No", "", nil},
		{geo.KindBlocked, "no", "● Yes", "", nil},
		{geo.KindBlocked, "no", "○ Unknown", "", errors.New("HTTP 500")},
	} {
		r := o.serviceRows([]geo.Result{{Check: geo.Check{Name: "Fixture", Kind: tc.kind}, Value: tc.value, Err: tc.err}})[0]
		if r.result != tc.want || r.region != tc.region {
			t.Fatalf("%+v -> %+v", tc, r)
		}
	}
	for _, tc := range []struct {
		id    string
		state access.State
		want  string
	}{{"netflix_access", access.StateAvailable, "● Yes"}, {"netflix_access", access.StateRestricted, "● Limited"}, {"other", access.StateRestricted, "● Limited"}, {"netflix_access", access.StateBlocked, "● No"}, {"netflix_access", access.StateError, "○ Unknown"}} {
		r := o.accessRows([]access.Result{{Check: access.Check{ID: tc.id}, State: tc.state, Region: "FRA"}})[0]
		if r.result != tc.want || r.region != "France" {
			t.Fatal(tc, r)
		}
	}
	for _, tc := range []struct {
		state ai.State
		want  string
	}{{ai.StateReachable, "● Yes"}, {ai.StateBlocked, "● No"}, {ai.StateError, "○ Unknown"}} {
		r := o.aiRows([]ai.Result{{Check: ai.Check{Name: "Qwen (Beijing)"}, State: tc.state, Status: 401}})[0]
		if r.result != tc.want || r.region != "" {
			t.Fatal(tc, r)
		}
	}
	if r := o.accessRows([]access.Result{{State: access.StateBlocked, Err: errors.New("timeout")}})[0]; r.result != "○ Unknown" {
		t.Fatal("network error became block")
	}
	if r := o.aiRows([]ai.Result{{State: ai.StateBlocked, Err: errors.New("timeout")}})[0]; r.result != "○ Unknown" {
		t.Fatal("network error became API refusal")
	}
}
func TestThreeAlignedEnglishColumnsNoDetailsOrTime(t *testing.T) {
	r := completeMock(t)
	r.Duration = 999 * time.Second
	for i := range r.Access {
		r.Access[i].Detail = "HIDDEN_ACCESS_DETAIL"
		r.Access[i].RTT = 888 * time.Second
	}
	for i := range r.Services {
		r.Services[i].Detail = "HIDDEN_SERVICE_DETAIL"
	}
	for i := range r.AI {
		r.AI[i].Detail = "HIDDEN_AI_DETAIL"
	}
	for _, profile := range []colorprofile.Profile{colorprofile.NoTTY, colorprofile.TrueColor, colorprofile.Ascii} {
		raw := renderProfile(t, r, profile)
		text := ansi.Strip(raw)
		if strings.IndexFunc(text, func(c rune) bool { return unicode.Is(unicode.Han, c) }) >= 0 {
			t.Fatal("Chinese terminal text", text)
		}
		if strings.Contains(text, "—") || strings.Contains(text, "N/A") {
			t.Fatal("placeholder region", text)
		}
		headers := []string{}
		width, rows, top, bottom, separator := 0, 0, 0, 0, 0
		for _, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "╭") {
				top++
				width = lg.Width(line)
			}
			if strings.HasPrefix(line, "╰") {
				bottom++
			}
			if strings.HasPrefix(line, "├") {
				separator++
			}
			if strings.ContainsAny(line, "┬┴┼") {
				t.Fatal("unrequested cell lines")
			}
			if strings.HasPrefix(line, "│") {
				rows++
				if strings.Count(line, "│") != 2 || lg.Width(line) != width {
					t.Fatal("broken border", line)
				}
				if strings.Contains(line, "Service") && strings.Contains(line, "Result") && strings.Contains(line, "Region") {
					headers = append(headers, line)
				}
			}
		}
		if rows != 40 || top != 3 || bottom != 3 || separator != 3 || len(headers) != 3 {
			t.Fatalf("counts=%d/%d/%d/%d/%d", rows, top, bottom, separator, len(headers))
		}
		if headers[0] != headers[1] || headers[1] != headers[2] {
			t.Fatal("three header positions differ", headers)
		}
		for _, word := range []string{"Time", "Detail", "Type", "Status", "HTTP", "completed in", "HIDDEN_", " ms", "耗时", "说明", "available", "reachable", "restricted", "blocked", "error", "country", "yes", " no ", " US ", " FRA ", " CHN ", "IPv4", "IPv6"} {
			if strings.Contains(text, word) {
				t.Fatalf("removed content %q survived", word)
			}
		}
		popular, stash, api := strings.Index(text, "Service regions & access"), strings.Index(text, "Service availability"), strings.Index(text, "AI endpoints")
		if popular < 0 || stash <= popular || api <= stash {
			t.Fatal("wrong group order")
		}
		for _, word := range []string{"20 Yes · 1 Unknown", "4 Yes · 2 No · 1 Limited · 1 Unknown", "6 Yes · 1 No · 1 Unknown"} {
			if !strings.Contains(text, word) {
				t.Fatal("missing correct summary", word, text)
			}
		}
		last := -1
		for _, res := range r.Services {
			p := strings.Index(text[:stash], res.Check.Name)
			if p <= last {
				t.Fatal("service order changed", res.Check.Name)
			}
			last = p
		}
		if profile == colorprofile.NoTTY && strings.Contains(raw, "\x1b") {
			t.Fatal("non-TTY ANSI leak")
		}
	}
}
func compact(s string) string { return strings.Join(strings.Fields(s), "") }
func TestCJKSharedWidthsAndLosslessWrappingOnNarrowScreens(t *testing.T) {
	o := &Output{Theme: newTheme(nil)}
	r := completeMock(t)
	r.Services[0].Value = "GS"                      // CLDR long English name; no fabricated translations.
	r.Services[0].Check.Name = "Fixture 中文 e\u0301" // Synthetic width-only fixture, not a real brand.
	r.Access[0].Region = "HKG"
	groups := o.groups(r)
	for _, terminal := range []int{14, 18, 20, 24, 30, 40, 60, 80, 100, 160} {
		widths, padding := sharedWidths(groups, terminal)
		outer := 2 + widths[0] + widths[1] + widths[2] + 6*padding
		if outer > terminal {
			t.Fatal("layout exceeds terminal", terminal, outer)
		}
		var headers []string
		for _, g := range groups {
			text := ansi.Strip(o.threeColumnTable(g.rows, widths, padding))
			for _, line := range strings.Split(text, "\n") {
				if lg.Width(line) != outer {
					t.Fatalf("screen=%d lineWidth=%d outer=%d: %q", terminal, lg.Width(line), outer, line)
				}
			}
			var head [3]string
			for _, line := range strings.Split(text, "\n") {
				if strings.HasPrefix(line, "├") {
					break
				}
				if !strings.HasPrefix(line, "│") {
					continue
				}
				start := 1
				for col := 0; col < 3; col++ {
					head[col] += strings.TrimSpace(ansi.Cut(line, start+padding, start+padding+widths[col]))
					start += widths[col] + 2*padding
				}
			}
			if head != [3]string{"Service", "Result", "Region"} {
				t.Fatal("header truncated", terminal, head)
			}
			headers = append(headers, strings.Join(head[:], "|"))
			// Render each row individually to reconstruct wrapped column contents and
			// prove no country/result/brand was truncated or replaced by an ellipsis.
			for _, row := range g.rows {
				text := ansi.Strip(o.threeColumnTable([]displayRow{row}, widths, padding))
				var reconstructed [3]string
				inBody := false
				for _, line := range strings.Split(text, "\n") {
					if strings.HasPrefix(line, "├") {
						inBody = true
						continue
					}
					if !inBody || !strings.HasPrefix(line, "│") {
						continue
					}
					start := 1
					for col := 0; col < 3; col++ {
						reconstructed[col] += strings.TrimSpace(ansi.Cut(line, start+padding, start+padding+widths[col]))
						start += widths[col] + 2*padding
					}
				}
				for col, want := range []string{row.service, row.result, row.region} {
					if compact(reconstructed[col]) != compact(want) {
						t.Fatalf("screen=%d col=%d lost text: got %q want %q", terminal, col, reconstructed[col], want)
					}
				}
			}
		}
		if headers[0] != headers[1] || headers[1] != headers[2] {
			t.Fatal("narrow header alignment differs")
		}
		var b bytes.Buffer
		o.W = &b
		o.Width = terminal
		o.printGroups(groups)
		for _, line := range strings.Split(ansi.Strip(b.String()), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "·") || strings.HasSuffix(strings.TrimSpace(line), "·") {
				t.Fatal("dangling summary separator", line)
			}
			if lg.Width(line) > terminal {
				t.Fatal("title/summary overflow", terminal, line)
			}
		}
	}
}
func TestTerminalPresentationDoesNotMutateReport(t *testing.T) {
	r := completeMock(t)
	before := Report{Duration: r.Duration, Access: append([]access.Result(nil), r.Access...), Services: append([]geo.Result(nil), r.Services...), AI: append([]ai.Result(nil), r.AI...)}
	// reflect.DeepEqual cannot compare functions in Check, so compare fields that
	// rendering uses/omits and all serialized outcome values via explicit copies.
	renderProfile(t, r, colorprofile.TrueColor)
	for i, x := range r.Access {
		b := before.Access[i]
		if x.Region != b.Region || x.Detail != b.Detail || x.State != b.State || x.RTT != b.RTT || x.Err != b.Err {
			t.Fatal("mutated access")
		}
	}
	for i, x := range r.Services {
		b := before.Services[i]
		if x.Value != b.Value || x.Detail != b.Detail || x.Err != b.Err || x.RTT != b.RTT {
			t.Fatal("mutated services")
		}
	}
	for i, x := range r.AI {
		b := before.AI[i]
		if x.Status != b.Status || x.State != b.State || x.Detail != b.Detail || x.RTT != b.RTT {
			t.Fatal("mutated AI")
		}
	}
}
func TestDefaultForegroundAndNOColorAcrossBackgrounds(t *testing.T) {
	var expected string
	for _, bg := range []string{"", "15;0", "0;15"} {
		t.Setenv("COLORFGBG", bg)
		o := &Output{Theme: newTheme(nil)}
		raw := renderProfile(t, completeMock(t), colorprofile.TrueColor)
		if expected != "" && raw != expected {
			t.Fatal("background changed text colour")
		}
		expected = raw
		for _, style := range []lg.Style{o.Theme.TableCell, o.Theme.Value, o.Theme.Border} {
			if strings.Contains(style.Render("Hong Kong SAR China"), "\x1b") {
				t.Fatal("information foreground forced")
			}
		}
		if !strings.Contains(raw, "\x1b[1m○ Unknown") || strings.Contains(raw, "38;5;8m") || strings.Contains(raw, "38;2;") || strings.Contains(raw, "\x1b]") {
			t.Fatal("unreadable status or OSC query")
		}
		for _, r := range o.accessRows([]access.Result{{State: access.StateAvailable}, {State: access.StateRestricted}, {State: access.StateBlocked}}) {
			if !strings.Contains(r.style.Render(r.result), "38;5;") {
				t.Fatal("semantic colour lost")
			}
		}
	}
	for _, name := range []string{"NO_COLOR", "CLICOLOR_FORCE", "CLICOLOR", "COLORTERM", "COLORFGBG", "CI"} {
		t.Setenv(name, "")
	}
	t.Setenv("TERM", "xterm-256color")
	for _, noColor := range []bool{false, true} {
		if noColor {
			t.Setenv("NO_COLOR", "anything")
			t.Setenv("CLICOLOR_FORCE", "1")
			t.Setenv("COLORTERM", "truecolor")
		}
		f, err := os.CreateTemp(t.TempDir(), "render")
		if err != nil {
			t.Fatal(err)
		}
		out := NewOutput(f)
		if out.Color || out.Width != 100 {
			t.Fatal("non-TTY profile")
		}
		if err = out.PrintAccess(completeMock(t)); err != nil {
			t.Fatal(err)
		}
		f.Close()
		data, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "38;") {
			t.Fatal("NO_COLOR still colour")
		}
		if !noColor && strings.Contains(string(data), "\x1b") {
			t.Fatal("nonTTY ANSI leak")
		}
		if !reflect.DeepEqual(ansi.Strip(string(data)), renderProfile(t, completeMock(t), colorprofile.NoTTY)) {
			t.Fatal("profile changed layout/text")
		}
	}
}
