package main

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"serviceunlock/internal/access"
)

func TestTerminalFooterUsesRealRunInterval(t *testing.T) {
	var measured time.Duration
	out := new(bytes.Buffer)
	code := runCLI(context.Background(), nil, out, new(bytes.Buffer), func(context.Context, time.Duration, int) ([]access.Result, error) {
		start := time.Now()
		time.Sleep(120 * time.Millisecond)
		results := fixtureResults()
		for i := range results {
			results[i].RTT = 999 * time.Second
		}
		measured = time.Since(start)
		return results, nil
	})
	match := regexp.MustCompile(`completed in ([0-9]+\.[0-9])s`).FindStringSubmatch(out.String())
	if code != 0 || len(match) != 2 {
		t.Fatal(code, out.String())
	}
	shown, err := strconv.ParseFloat(match[1], 64)
	if err != nil || math.Abs(shown-measured.Seconds()) > 0.1 {
		t.Fatal("footer not real run duration", shown, measured)
	}
	if strings.Contains(out.String(), "7个平台") || strings.Contains(out.String(), "GeoIP") {
		t.Fatal("unrequested heading/section")
	}
}
func TestJSONUnchangedUnderForcedColor(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1")
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("NO_COLOR", "")
	out := new(bytes.Buffer)
	code := runCLI(context.Background(), []string{"--json"}, out, new(bytes.Buffer), func(context.Context, time.Duration, int) ([]access.Result, error) { return fixtureResults(), nil })
	if code != 0 || strings.Contains(out.String(), "\x1b") || strings.Contains(out.String(), "Stash checks") {
		t.Fatal(code, out.String())
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &object); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"schema_version", "source", "generated_at", "network", "results"} {
		if _, ok := object[name]; !ok {
			t.Fatal(name)
		}
	}
	if len(object) != 5 {
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
			t.Fatal("changed result fields", row)
		}
	}
}
