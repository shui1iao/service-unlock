package main

import (
	"bytes"
	"context"
	"encoding/json"
	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/extra"
	"serviceunlock/internal/geo"
	"strings"
	"testing"
	"time"
)

func TestRemovedNotebookLMNotRegisteredRequestedOrRendered(t *testing.T) {
	if len(access.Checks()) != 6 {
		t.Fatal("access count")
	}
	for _, c := range access.Checks() {
		if strings.Contains(strings.ToLower(c.ID+c.Name), "notebook") {
			t.Fatal("removed probe registered")
		}
	}
	// Existing actual-request-path mock is a strict endpoint allowlist; removed
	// Notebook endpoints are not in it, so any attempt would fail the test.
	stack, trace := mockCompleteStack(t)
	m, err := runGroups(context.Background(), stack, 3)
	if err != nil {
		t.Fatal(err)
	}
	for endpoint := range trace.seen {
		if strings.Contains(strings.ToLower(endpoint), "notebook") {
			t.Fatal("removed endpoint requested", endpoint)
		}
	}
	core := len(access.Checks()) + len(geo.ServiceChecks()) + len(ai.Checks())
	if core != 35 {
		t.Fatal(core)
	}
	for _, region := range extra.Regions() {
		checks, e := extra.Selected(region)
		if e != nil {
			t.Fatal(e)
		}
		fixture := unifiedFixture(region, true)
		if len(fixture.Access) != 6 || len(fixture.Extra) != len(checks) {
			t.Fatal("catalogue changed")
		}
		var out, stderr bytes.Buffer
		code := runCLI(context.Background(), []string{"--machine", "--region", region, "--extra-family", "4"}, &out, &stderr, func(context.Context, time.Duration, int) (measurements, error) { return fixture, nil })
		if code != 1 {
			t.Fatal(code, stderr.String())
		}
		if strings.Contains(strings.ToLower(out.String()), "notebook") {
			t.Fatal("removed probe still in JSON/ANSI")
		}
		var doc struct {
			Report report `json:"report"`
		}
		if e = json.Unmarshal(out.Bytes(), &doc); e != nil {
			t.Fatal(e)
		}
		if len(doc.Report.Results)+len(doc.Report.Services)+len(doc.Report.AIEndpoints)+len(doc.Report.ExtraChecks) != core+len(checks) {
			t.Fatal("count not derived from actual selection")
		}
	}
	if len(m.Access) != 6 {
		t.Fatal("actual run includes removed item")
	}
}
