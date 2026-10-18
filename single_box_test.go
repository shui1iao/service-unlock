package main

import (
	"bytes"
	"encoding/json"
	"github.com/charmbracelet/x/ansi"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestSingleBoxCompleteScopesAndMachineJSONUnchanged(t *testing.T) {
	titles := []string{"Service regions & access", "Service availability", "AI endpoints", "Global services", "Regional services"}
	for _, region := range []string{"global", "jp", "eu", "na"} {
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
			if strings.Count(text, "Service                Result") > 1 || strings.Count(text, "Result") != 1 || strings.Count(text, "Region │") != 1 {
				t.Fatal("duplicate headers", text)
			}
			nsections := len(titles)
			if region == "global" {
				nsections--
			}
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
			if region == "global" && strings.Contains(text, "Regional services") {
				t.Fatal("empty local section")
			}
			count := len(r.Results) + len(r.Services) + len(r.AIEndpoints) + len(r.ExtraChecks)
			statusLines := regexp.MustCompile(`(?m)^│.*[●○] (?:Yes|No|Limited|Unknown)`).FindAllString(text, -1)
			if len(statusLines) != count {
				t.Fatal("missing service rows", len(statusLines), count)
			}
			expectedRows := make([][]string, 5)
			for _, row := range m.Services {
				expectedRows[0] = append(expectedRows[0], row.Check.Name)
			}
			for _, row := range m.Access {
				expectedRows[1] = append(expectedRows[1], row.Check.Name)
			}
			for _, row := range m.AI {
				expectedRows[2] = append(expectedRows[2], row.Check.Name)
			}
			for _, row := range m.Extra {
				group := 3
				if row.Scope != "global" {
					group = 4
				}
				expectedRows[group] = append(expectedRows[group], row.Name)
			}
			for i, title := range titles[:nsections] {
				sectionText := strings.SplitN(text, title, 2)[1]
				if i+1 < nsections {
					sectionText = strings.SplitN(sectionText, titles[i+1], 2)[0]
				}
				remaining := sectionText
				for _, name := range expectedRows[i] {
					pos := strings.Index(remaining, name)
					if pos < 0 {
						t.Fatal("missing/out of order row", title, name)
					}
					remaining = remaining[pos+len(name):]
				}
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
