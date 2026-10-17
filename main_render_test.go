package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"serviceunlock/internal/ai"
	"serviceunlock/internal/geo"
)

func TestTerminalRemovesTimesDetailsButJSONKeepsOriginalValues(t *testing.T) {
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	m := measurements{Access: fixtureResults()}
	m.Access[0].Region = "USA"
	m.Access[0].Detail = "DETAIL_ACCESS"
	m.Access[0].RTT = 999 * time.Second
	m.Access[0].CheckedAt = when
	m.Services = []geo.Result{{Check: geo.Check{ID: "google", Name: "Google", Kind: geo.KindCountry}, Value: "US", Detail: "DETAIL_SERVICE", RTT: 888 * time.Second, CheckedAt: when}}
	m.AI = []ai.Result{{Check: ai.Check{ID: "openai", Name: "OpenAI", Vendor: "OpenAI"}, State: ai.StateReachable, Status: 401, Detail: "DETAIL_API", RTT: 777 * time.Second, CheckedAt: when}}
	run := func(context.Context, time.Duration, int) (measurements, error) { return m, nil }
	out := new(bytes.Buffer)
	if code := runCLI(context.Background(), nil, out, new(bytes.Buffer), run); code != 0 {
		t.Fatal(code)
	}
	for _, bad := range []string{"DETAIL_", "completed in", "耗时", "说明", "Time", " ms", "999", "888", "777"} {
		if strings.Contains(out.String(), bad) {
			t.Fatal("removed terminal content", bad, out.String())
		}
	}
	if !strings.Contains(out.String(), "United States") || !strings.Contains(out.String(), "Yes") {
		t.Fatal(out.String())
	}
	out.Reset()
	if code := runCLI(context.Background(), []string{"--json"}, out, new(bytes.Buffer), run); code != 0 {
		t.Fatal(code)
	}
	var got report
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if *got.Results[0].Region != "USA" || got.Results[0].Detail != "DETAIL_ACCESS" || got.Results[0].ElapsedMS != 999000 || got.Results[0].Status != "available" {
		t.Fatal(got.Results[0])
	}
	if *got.Services[0].Value != "US" || got.Services[0].Detail != "DETAIL_SERVICE" || got.Services[0].ElapsedMS != 888000 || got.Services[0].Type != "country" {
		t.Fatal(got.Services[0])
	}
	if got.AIEndpoints[0].Detail != "DETAIL_API" || got.AIEndpoints[0].ElapsedMS != 777000 || got.AIEndpoints[0].HTTPStatus != 401 || got.AIEndpoints[0].Status != "reachable" {
		t.Fatal(got.AIEndpoints[0])
	}
	// The same measurements must serialize to identical JSON value bytes after
	// rendering. Ignore only the independent generated_at timestamp.
	expected := makeCompleteReport(m, 6, 4)
	expected.GeneratedAt = got.GeneratedAt
	a, _ := json.Marshal(expected)
	b, _ := json.Marshal(got)
	if !bytes.Equal(a, b) {
		t.Fatal("rendering mutated JSON", string(a), string(b))
	}
}
func TestJSONUnchangedUnderForcedColor(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1")
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("NO_COLOR", "")
	out := new(bytes.Buffer)
	code := runCLI(context.Background(), []string{"--json"}, out, new(bytes.Buffer), func(context.Context, time.Duration, int) (measurements, error) {
		return measurements{Access: fixtureResults()}, nil
	})
	if code != 0 || strings.Contains(out.String(), "\x1b") || strings.Contains(out.String(), "Service availability") {
		t.Fatal(code, out.String())
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &object); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"schema_version", "source", "generated_at", "network", "results", "services", "ai_endpoints"} {
		if _, ok := object[name]; !ok {
			t.Fatal(name)
		}
	}
	if len(object) != 7 {
		t.Fatal("changed JSON top-level fields")
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(object["results"], &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 8 {
		t.Fatal(len(rows))
	}
	for _, row := range rows {
		if len(row) != 9 {
			t.Fatal("changed access result fields", row)
		}
	}
}
