package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"serviceunlock/internal/geo"
)

func TestGoogleYouTubeCNHasOneTerminalJSONMachineConclusion(t *testing.T) {
	m := measurements{Services: []geo.Result{{Check: geo.Check{ID: "google_captcha", Name: "Google", Kind: geo.KindBlocked}, Err: errors.New("Search timeout"), Detail: "Search failed"}}}
	geo.ApplyYouTubeCountry(m.Services, geo.YouTubeCountryEvidence{Region: "CN", Source: "https://www.youtube.com/premium countryCode"})
	run := func(context.Context, time.Duration, int) (measurements, error) { return m, nil }
	for _, args := range [][]string{nil, {"--json"}, {"--machine"}} {
		out := new(bytes.Buffer)
		if code := runCLI(context.Background(), args, out, new(bytes.Buffer), run); code != 0 {
			t.Fatal(code, out.String())
		}
		if len(args) == 0 {
			if !strings.Contains(out.String(), "Google") || !strings.Contains(out.String(), "No") || !strings.Contains(out.String(), "CN") {
				t.Fatal(out.String())
			}
			continue
		}
		var got report
		if args[0] == "--machine" {
			var envelope struct {
				Report   report `json:"report"`
				Terminal string `json:"terminal"`
			}
			if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			got = envelope.Report
			if !strings.Contains(envelope.Terminal, "Google") || !strings.Contains(envelope.Terminal, "No") || !strings.Contains(envelope.Terminal, "CN") {
				t.Fatal(envelope.Terminal)
			}
		} else if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if len(got.Services) != 1 {
			t.Fatal(out.String())
		}
		r := got.Services[0]
		if r.Type != "blocked" || r.Status != "observed" || r.Value == nil || *r.Value != "yes" || r.Region == nil || *r.Region != "CN" || r.Error != nil || !strings.Contains(r.Detail, "Search timeout") || !strings.Contains(r.Detail, "YouTube") {
			t.Fatal(r)
		}
	}
}
