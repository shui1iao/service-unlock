// MIT License; Copyright (c) 2026 Remnawave. Offline rendering tests only.
package render

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	lg "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"serviceunlock/internal/access"
)

func mockReport(t *testing.T) Report {
	t.Helper()
	raw, err := os.ReadFile("testdata/input.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Duration int64 `json:"duration_ns"`
		Results  []struct {
			ID, Name, Region, Detail, Error string
			State                           access.State
			RTT                             int64 `json:"rtt_ns"`
		}
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	r := Report{Duration: time.Duration(fixture.Duration)}
	for _, x := range fixture.Results {
		res := access.Result{Check: access.Check{ID: x.ID, Name: x.Name}, State: x.State, Region: x.Region, Detail: x.Detail, RTT: time.Duration(x.RTT)}
		if x.Error != "" {
			res.Err = errors.New(x.Error)
		}
		r.Access = append(r.Access, res)
	}
	return r
}
func renderProfile(t *testing.T, r Report, profile colorprofile.Profile) string {
	t.Helper()
	var buf bytes.Buffer
	w := colorprofile.NewWriter(&buf, nil)
	w.Profile = profile
	o := &Output{W: w, Theme: newTheme(lg.LightDark(true)), Color: profile > colorprofile.Ascii, Width: 100}
	if err := o.PrintAccess(r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
func TestUpstreamGoldenExactBytes(t *testing.T) {
	// These expected files were generated from the ORIGINAL report.go/theme.go
	// functions in an independent reference package, not from this candidate.
	for _, tc := range []struct {
		name    string
		profile colorprofile.Profile
	}{{"plain", colorprofile.NoTTY}, {"ansi", colorprofile.TrueColor}, {"nocolor", colorprofile.Ascii}} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := os.ReadFile("testdata/upstream-" + tc.name + ".golden")
			if err != nil {
				t.Fatal(err)
			}
			actual := renderProfile(t, mockReport(t), tc.profile)
			if actual != string(expected) {
				t.Fatalf("output differs from original source golden\nactual:\n%s\nexpected:\n%s", actual, expected)
			}
		})
	}
}
func TestOriginalTableShapeAndOrder(t *testing.T) {
	text := renderProfile(t, mockReport(t), colorprofile.NoTTY)
	if strings.Contains(text, "\x1b") {
		t.Fatal("plain profile contains ANSI")
	}
	if !strings.HasPrefix(text, "  Stash checks ") || strings.Contains(text, "7个平台") {
		t.Fatal(text)
	}
	order := []string{"Service", "Status", "Region", "Detail", "Time", "ChatGPT (web)", "ChatGPT (app)", "YouTube Premium", "Netflix", "Claude", "TikTok", "Gemini", "NotebookLM"}
	last := -1
	for _, marker := range order {
		index := strings.Index(text, marker)
		if index <= last {
			t.Fatalf("out of order: %s", marker)
		}
		last = index
	}
	rows, headLines, tops, bottoms := 0, 0, 0, 0
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "│") {
			rows++
			if strings.Count(line, "│") != 2 {
				t.Fatalf("inter-column border: %s", line)
			}
		}
		if strings.HasPrefix(line, "├") {
			headLines++
		}
		if strings.HasPrefix(line, "╭") {
			tops++
		}
		if strings.HasPrefix(line, "╰") {
			bottoms++
		}
		if strings.ContainsAny(line, "┬┴┼") {
			t.Fatalf("unexpected cell separator: %s", line)
		}
	}
	if rows != 9 || headLines != 1 || tops != 1 || bottoms != 1 {
		t.Fatalf("rows=%d separators=%d corners=%d/%d", rows, headLines, tops, bottoms)
	}
	if !strings.Contains(text, "4 available · 1 restricted · 2 blocked · 1 inconclusive") || !strings.Contains(text, "completed in 6.2s") {
		t.Fatal(text)
	}
}
func TestOriginalStatusesColorsAndTimes(t *testing.T) {
	o := &Output{Theme: newTheme(lg.LightDark(true))}
	for _, tc := range []struct {
		state         access.State
		symbol, color string
	}{{access.StateAvailable, "● available", "38;2;94;224;138"}, {access.StateRestricted, "● restricted", "38;2;245;196;83"}, {access.StateBlocked, "● blocked", "38;2;255;123;114"}, {access.StateError, "○ error", "38;2;138;138;138"}} {
		cell := o.accessStateCell(tc.state)
		if ansi.Strip(cell) != tc.symbol || !strings.Contains(cell, tc.color) {
			t.Fatalf("%q", cell)
		}
	}
	if cell := ansi.Strip(o.accessTimeCell(access.Result{State: access.StateError, Err: errors.New("mock timeout"), RTT: 6 * time.Second})); cell != "—" {
		t.Fatal(cell)
	}
	// Match upstream exactly: only an actual Err (or zero duration) hides time.
	if cell := ansi.Strip(o.accessTimeCell(access.Result{State: access.StateError, RTT: 10 * time.Millisecond})); cell != "10.0 ms" {
		t.Fatal(cell)
	}
	for _, tc := range []struct {
		d           time.Duration
		text, color string
	}{
		{0, "—", "38;2;138;138;138"}, {5678 * time.Microsecond, "5.68 ms", "38;2;94;224;138"},
		{30 * time.Millisecond, "30.0 ms", "38;2;94;224;138"}, {31 * time.Millisecond, "31.0 ms", "38;2;111;211;247"},
		{80 * time.Millisecond, "80.0 ms", "38;2;111;211;247"}, {81 * time.Millisecond, "81.0 ms", "38;2;245;196;83"},
		{160 * time.Millisecond, "160 ms", "38;2;245;196;83"}, {161 * time.Millisecond, "161 ms", "38;2;255;123;114"},
	} {
		cell := o.rttCell(tc.d)
		if ansi.Strip(cell) != tc.text || !strings.Contains(cell, tc.color) {
			t.Fatalf("%v: %q", tc.d, cell)
		}
	}
}
func TestOriginalClippingMissingRegionAndCounts(t *testing.T) {
	if clip(strings.Repeat("x", 52), 52) != strings.Repeat("x", 52) {
		t.Fatal("clipped exact limit")
	}
	if clip(strings.Repeat("x", 53), 52) != strings.Repeat("x", 51)+"…" {
		t.Fatal("wrong clip")
	}
	if clip(strings.Repeat("界", 18), 52) != strings.Repeat("界", 17)+"…" {
		t.Fatal("must preserve upstream byte slicing")
	}
	r := mockReport(t)
	text := renderProfile(t, r, colorprofile.NoTTY)
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "ChatGPT (web)") && !strings.Contains(line, "—") {
			t.Fatal("missing region not em dash")
		}
		if strings.Contains(line, "Claude") && (!strings.Contains(line, "○ error") || strings.Contains(line, "6000")) {
			t.Fatal(line)
		}
	}
	for i := range r.Access {
		r.Access[i].State = access.StateAvailable
	}
	text = renderProfile(t, r, colorprofile.NoTTY)
	if !strings.Contains(text, "  8 available\n") || strings.Contains(text, "0 restricted") || strings.Contains(text, " · ") {
		t.Fatal("zero categories must be omitted")
	}
}
func TestSectionWidthAndFooter(t *testing.T) {
	for _, width := range []int{20, 100, 120} {
		var b bytes.Buffer
		o := &Output{W: &b, Theme: Theme{}, Width: width}
		o.section("Stash checks")
		content := width - 4
		if content < 20 {
			content = 20
		}
		if got := strings.Count(b.String(), "─"); got != content-lg.Width("Stash checks")-3 {
			t.Fatal(width, got)
		}
	}
	var b bytes.Buffer
	o := &Output{W: &b}
	o.footer(Report{})
	if b.String() != "\n" {
		t.Fatal("zero footer duration", b.String())
	}
}
func TestNonTTYAndNOColorOriginalBehavior(t *testing.T) {
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
		o := NewOutput(f)
		if o.Width != 100 || o.Color {
			t.Fatal(o.Width, o.Color)
		}
		if err = o.PrintAccess(mockReport(t)); err != nil {
			t.Fatal(err)
		}
		f.Close()
		data, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		mode := "plain"
		if noColor {
			mode = "nocolor"
		}
		expected, _ := os.ReadFile("testdata/upstream-" + mode + ".golden")
		if string(data) != string(expected) {
			t.Fatalf("%s not upstream behavior", mode)
		}
		if bytes.Contains(data, []byte("38;2;")) {
			t.Fatal("NO_COLOR failed")
		}
	}
}
func TestOriginalBackgroundDetection(t *testing.T) {
	for _, tc := range []struct {
		env  string
		dark bool
	}{{"", true}, {"15;0", true}, {"0;7", false}, {"0;8", true}, {"0;15", false}, {"invalid", true}, {"0;x", true}} {
		t.Setenv("COLORFGBG", tc.env)
		if backgroundIsDark() != tc.dark {
			t.Fatal(tc)
		}
	}
	o := &Output{Theme: newTheme(lg.LightDark(false))}
	if !strings.Contains(o.accessStateCell(access.StateAvailable), "38;2;31;138;68") {
		t.Fatal("light palette changed")
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestAdapterPreservesWriteErrors(t *testing.T) {
	if err := WriteAccess(brokenWriter{}, mockReport(t)); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}
