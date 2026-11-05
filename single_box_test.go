package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/charmbracelet/x/ansi"
	"os"
	"path/filepath"
	"regexp"
	"serviceunlock/internal/extra"
	"strings"
	"testing"
	"time"
)

func TestSingleBoxCompleteScopesAndMachineJSONUnchanged(t *testing.T) {
	titles := []string{"Video, live & sports", "Music & audio", "AI services", "Gaming", "Community", "General websites"}
	for _, region := range extra.Regions() {
		t.Run(region, func(t *testing.T) {
			m := unifiedFixture(region, false)
			if region == "jp" {
				m.Selection.RequestedRegion = "auto"
				m.Selection.DetectedCountry = "JP"
				m.Selection.DetectionSource = "https://www.cloudflare.com/cdn-cgi/trace"
				m.Selection.ResolutionReason = "country_mapped"
			}
			r := makeCompleteReport(m, 6, 4)
			r.GeneratedAt = "2026-10-18T08:00:00.000Z"
			encode := func(value report) []byte {
				var b bytes.Buffer
				enc := json.NewEncoder(&b)
				enc.SetEscapeHTML(false)
				if e := enc.Encode(value); e != nil {
					t.Fatal(e)
				}
				return bytes.TrimSuffix(b.Bytes(), []byte("\n"))
			}
			before := encode(r)
			var e error
			var out bytes.Buffer
			if e = writeMachine(&out, r, m, time.Second); e != nil {
				t.Fatal(e)
			}
			var envelope struct {
				Format   string          `json:"format"`
				Report   json.RawMessage `json:"report"`
				Terminal string          `json:"terminal"`
			}
			if e = json.Unmarshal(out.Bytes(), &envelope); e != nil {
				t.Fatal(e)
			}
			after := encode(r)
			if !bytes.Equal(before, after) || !bytes.Equal(before, envelope.Report) || envelope.Format != "guko-unified-v1" {
				t.Fatal("machine changed report JSON bytes")
			}
			text := ansi.Strip(envelope.Terminal)
			for _, corner := range []string{"╭", "╮", "╰", "╯"} {
				if strings.Count(text, corner) != 1 {
					t.Fatal("not one frame", corner)
				}
			}
			// Aggregated source-region text can widen Region; do not assume
			// the old six-cell column or its exact padding in header matching.
			if strings.Count(text, "Result") != 1 || len(regexp.MustCompile(`(?m)^│\s*Service\s+Result\s+Region\s*│$`).FindAllString(text, -1)) != 1 {
				t.Fatal("duplicate/missing headers", text)
			}
			nsections := len(titles)
			last := -1
			for _, title := range titles[:nsections] {
				p := strings.Index(text, title)
				if p <= last {
					t.Fatal("section order", title)
				}
				last = p
				for _, line := range strings.Split(text, "\n") {
					if strings.Contains(line, title) && !strings.HasPrefix(line, "│") {
						t.Fatal("section outside frame", line)
					}
				}
			}
			// Every primary probe has one status; long names may wrap naturally.
			for _, name := range []string{"ChatGPT", "Claude", "Gemini", "Qwen Intl (API)", "Netflix", "TikTok", "Spotify (Legacy signup endpoint)", "Reddit (guest)"} {
				pattern := regexp.MustCompile(`(?m)^│\s*` + regexp.QuoteMeta(name) + `\s{2,}`)
				if len(pattern.FindAllString(text, -1)) != 1 {
					t.Fatal("primary not exactly one Service cell", name, text)
				}
			}
			for _, row := range m.Extra {
				name := row.Name
				if row.ID == "MediaUnlockTest_PrimeVideo" {
					name = "Amazon Prime Video"
				}
				if row.ID == "MediaUnlockTest_DMM" {
					name = "DMM (financial site)"
				}
				// Default width fits these original names; no brand aliases.
				if !strings.Contains(text, name) {
					t.Fatal("missing original extra name", row.ID, name)
				}
			}
			if region != "global" && !strings.Contains(text, "Regional media — ") {
				t.Fatal("missing regional section", region)
			}
			// CLI and machine must use the same direct layout and existing one
			// completed measurement, not perform a second request for merging.
			var ordinary, stderr bytes.Buffer
			calls := 0
			code := runCLI(context.Background(), []string{"--region", region, "--extra-family", "4"}, &ordinary, &stderr, func(context.Context, time.Duration, int) (measurements, error) { calls++; return m, nil })
			if code != 1 || calls != 1 || stderr.Len() != 0 || ordinary.String() != text {
				t.Fatal("normal/machine presentation diverged or reran", code, calls, stderr.String())
			}
			if regexp.MustCompile(`(?m)^(?:│\s*)?(Yes|No|Limited|Unknown) [0-9]+:`).MatchString(text) {
				t.Fatal("retired summaries remain")
			}
			for _, bad := range []string{"not playback/login guarantees", "source access criterion", "inconclusive, unsupported", "originals-only", "Detail", "Duration", "completed in"} {
				if strings.Contains(text, bad) {
					t.Fatal("retired explanation", bad)
				}
			}
			expectedNotices := 0
			if region == "jp" {
				expectedNotices = 1
			}
			if strings.Count(text, "Automatic region:") != expectedNotices {
				t.Fatal("selection notice duplication")
			}
			if dir := os.Getenv("SINGLE_BOX_ARTIFACT_DIR"); dir != "" {
				if e = os.MkdirAll(dir, 0700); e != nil {
					t.Fatal(e)
				}
				os.WriteFile(filepath.Join(dir, region+".machine.json"), out.Bytes(), 0600)
				os.WriteFile(filepath.Join(dir, region+".report.json"), before, 0600)
				os.WriteFile(filepath.Join(dir, region+".ansi.txt"), []byte("OFFLINE MOCK - NOT REAL MEASUREMENT\n"+strings.TrimRight(strings.ReplaceAll(envelope.Terminal, "\r\n", "\n"), "\r\n")+"\n"), 0600)
				os.WriteFile(filepath.Join(dir, region+".txt"), []byte("OFFLINE MOCK - NOT REAL MEASUREMENT\n"+text), 0600)
			}
		})
	}
}
