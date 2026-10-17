// Offline CLI tests. MIT; Copyright (c) 2026 Remnawave.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"serviceunlock/internal/access"
)

func fixtureResults() []access.Result {
	results := []access.Result{}
	for _, check := range access.Checks() {
		results = append(results, access.Result{Check: check, State: access.StateAvailable, CheckedAt: time.Now(), RTT: time.Millisecond})
	}
	return results
}
func TestCLIJSONScopeFieldsAndExitCodes(t *testing.T) {
	for _, hasError := range []bool{false, true} {
		out, errout := new(bytes.Buffer), new(bytes.Buffer)
		runner := func(_ context.Context, timeout time.Duration, workers int) ([]access.Result, error) {
			if timeout != 100*time.Millisecond || workers != 2 {
				t.Fatal(timeout, workers)
			}
			results := fixtureResults()
			results[0].State = access.StateBlocked
			results[3].State = access.StateRestricted
			results[3].Region = "FR"
			if hasError {
				results[7].State = access.StateError
				results[7].Detail = "parse failure"
			}
			return results, nil
		}
		code := runCLI(context.Background(), []string{"--json", "--timeout", ".1", "--workers", "2"}, out, errout, runner)
		want := 0
		if hasError {
			want = 1
		}
		if code != want {
			t.Fatal(code)
		}
		var r report
		if err := json.Unmarshal(out.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if r.SchemaVersion != 1 || len(r.Results) != 8 || r.Network.Workers != 2 || errout.Len() != 0 {
			t.Fatalf("%+v %s", r, errout)
		}
		platforms := map[string]bool{}
		for _, item := range r.Results {
			platforms[item.Platform] = true
			if item.ID == "" || item.CheckedAt == "" || item.ElapsedMS <= 0 {
				t.Fatalf("%+v", item)
			}
		}
		if len(platforms) != 7 || r.Results[0].ID != "chatgpt_web" || r.Results[1].ID != "chatgpt_app" {
			t.Fatalf("%+v", r.Results)
		}
		if hasError && r.Results[7].Error == nil {
			t.Fatal("missing error")
		}
		var raw map[string]json.RawMessage
		json.Unmarshal(out.Bytes(), &raw)
		if _, ok := raw["status"]; ok {
			t.Fatal("invented aggregate")
		}
	}
}
func TestCLIHelpValidationAndHumanOutput(t *testing.T) {
	never := func(context.Context, time.Duration, int) ([]access.Result, error) {
		t.Fatal("help/invalid args attempted measurement")
		return nil, nil
	}
	for _, args := range [][]string{{"--help"}, {"-h"}, {"--workers", "0"}, {"--workers", "9"}, {"--timeout", "NaN"}, {"--timeout", "0"}, {"--timeout", "-1"}, {"--unknown"}, {"extra"}} {
		out, errout := new(bytes.Buffer), new(bytes.Buffer)
		code := runCLI(context.Background(), args, out, errout, never)
		if args[0] == "--help" || args[0] == "-h" {
			if code != 0 || !strings.Contains(out.String(), "7个平台") {
				t.Fatal(code, out)
			}
		} else if code != 2 || out.Len() != 0 {
			t.Fatal(code, out)
		}
	}
	out := new(bytes.Buffer)
	code := runCLI(context.Background(), nil, out, new(bytes.Buffer), func(context.Context, time.Duration, int) ([]access.Result, error) { return fixtureResults(), nil })
	if code != 0 || !strings.Contains(out.String(), "ChatGPT (app)") || !strings.Contains(out.String(), "error = 未知") {
		t.Fatal(code, out)
	}
}
func TestCLICancelNoPartialJSON(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, errout := new(bytes.Buffer), new(bytes.Buffer)
	code := runCLI(ctx, []string{"--json"}, out, errout, func(context.Context, time.Duration, int) ([]access.Result, error) { return nil, context.Canceled })
	if code != 130 || out.Len() != 0 || !strings.Contains(errout.String(), "中断") {
		t.Fatal(code, out, errout)
	}
}
