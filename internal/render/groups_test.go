package render

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/geo"
)

func completeMock(t *testing.T) Report {
	r := mockReport(t)
	for _, c := range geo.ServiceChecks() {
		value := "US"
		if c.Kind == geo.KindAvailability {
			value = "yes"
		}
		if c.Kind == geo.KindBlocked {
			value = "no"
		}
		res := geo.Result{Check: c, Value: value, Detail: "offline observation; not an unlock verdict", RTT: 20 * time.Millisecond}
		if c.ID == "apple" {
			res.Value = ""
			res.Err = errors.New("mock timeout; result unknown")
		}
		r.Services = append(r.Services, res)
	}
	for _, c := range ai.Checks() {
		res := ai.Result{Check: c, State: ai.StateReachable, Status: 401, Detail: "answered, credentials not supplied", RTT: 20 * time.Millisecond}
		if c.ID == "anthropic" {
			res.State = ai.StateBlocked
			res.Status = 403
			res.Detail = "the API refused this region"
		}
		if c.ID == "gemini" {
			res.Status = 403
		}
		if c.ID == "zhipu" {
			res.State = ai.StateError
			res.Status = 429
			res.Detail = "unexpected HTTP 429"
		}
		r.AI = append(r.AI, res)
	}
	return r
}
func TestThreeSeparateTables37RowsAndNoFamilyClaim(t *testing.T) {
	r := completeMock(t)
	for _, profile := range []colorprofile.Profile{colorprofile.NoTTY, colorprofile.TrueColor, colorprofile.Ascii} {
		raw := renderProfile(t, r, profile)
		text := ansi.Strip(raw)
		popular, stash, api := strings.Index(text, "Popular services"), strings.Index(text, "Stash checks"), strings.Index(text, "AI endpoints")
		if popular < 0 || stash <= popular || api <= stash || strings.Count(text, "completed in") != 1 {
			t.Fatal(text)
		}
		if strings.Contains(text, "IPv4") || strings.Contains(text, "IPv6") || strings.Contains(text, "GeoIP") || strings.Contains(text, "speed.bin") {
			t.Fatal("out-of-scope output")
		}
		if !strings.Contains(text, "Type") || !strings.Contains(text, "Result") || !strings.Contains(text, "captcha/block") || !strings.Contains(text, "HTTP") || !strings.Contains(text, "● reachable") {
			t.Fatal("semantics hidden", text)
		}
		if !strings.Contains(text, "20 observations · 1 inconclusive") || !strings.Contains(text, "6 reachable · 1 blocked · 1 inconclusive") {
			t.Fatal("group statistics", text)
		}
		rows, separators := 0, 0
		for _, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "│") {
				rows++
				if strings.Count(line, "│") != 2 {
					t.Fatal("column divider", line)
				}
			}
			if strings.HasPrefix(line, "├") {
				separators++
			}
		}
		if rows != 40 || separators != 3 || strings.Count(text, "╭") != 3 || strings.Count(text, "╰") != 3 {
			t.Fatal("37 rows plus 3 headers", rows, separators)
		}
		last := -1
		for _, res := range r.Services {
			pos := strings.Index(text[:stash], res.Check.Name)
			if pos <= last {
				t.Fatal("service order", res.Check.Name)
			}
			last = pos
		}
		last = -1
		for _, res := range r.AI {
			pos := strings.Index(text[api:], res.Check.Name)
			if pos <= last {
				t.Fatal("AI order", res.Check.Name)
			}
			last = pos
		}
		if strings.Contains(raw, "38;2;") || strings.Contains(raw, "\x1b]") {
			t.Fatal("fixed RGB or OSC query")
		}
		if profile == colorprofile.NoTTY && strings.Contains(raw, "\x1b") {
			t.Fatal("non-TTY ANSI leak")
		}
	}
}
func TestPaletteIndependentOfBackgroundAndReadableInformation(t *testing.T) {
	var expected string
	for _, bg := range []string{"", "15;0", "0;15"} {
		t.Setenv("COLORFGBG", bg)
		text := renderProfile(t, completeMock(t), colorprofile.TrueColor)
		if expected != "" && text != expected {
			t.Fatal("unknown background changed foreground choices")
		}
		expected = text
		if !strings.Contains(text, "\x1b[1m○ error") || strings.Contains(text, "38;5;8m") {
			t.Fatal("error remains dim")
		}
		// Information cells render independently of any colour palette.
		theme := newTheme(nil)
		for _, value := range []string{"FR", "US", "a request failed", "ChatGPT (web)", "served region; not an unlock verdict"} {
			if strings.Contains(theme.Muted.Render(value), "\x1b") || strings.Contains(theme.TableCell.Render(value), "\x1b") {
				t.Fatal("information has fixed foreground", value)
			}
		}
	}
}
