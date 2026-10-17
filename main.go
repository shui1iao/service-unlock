// Service Unlock v0.4.2. MIT License; Copyright (c) 2026 Remnawave.
// Derived from remnawave/geocheck v0.3.0; see LICENSE and README.md.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/geo"
	"serviceunlock/internal/netx"
	"serviceunlock/internal/render"
)

//go:embed LICENSE
var licenseText string

const help = `Service Unlock v0.4.2 — 37 checks / 3 groups
Usage: service-unlock [--json] [--timeout SECONDS] [--workers N]
  --json       Output JSON only; error means inconclusive, not blocked
  --timeout N  Total budget per HTTP request in seconds, default 6
               Includes redirects and response body
  --workers N  Bounded concurrency from 1 to 8, default 4
  --help       Show help without running checks
  --license    Show bundled licenses without running checks
Running checks contacts real services; no Node/Python/Go runtime is needed.
`

type item struct {
	ID        string  `json:"id"`
	Platform  string  `json:"platform"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Detail    string  `json:"detail"`
	Region    *string `json:"region"`
	Error     *string `json:"error"`
	CheckedAt string  `json:"checked_at"`
	ElapsedMS float64 `json:"elapsed_ms"`
}
type report struct {
	SchemaVersion int               `json:"schema_version"`
	Source        map[string]string `json:"source"`
	GeneratedAt   string            `json:"generated_at"`
	Network       networkInfo       `json:"network"`
	Results       []item            `json:"results"`
	Services      []serviceItem     `json:"services"`
	AIEndpoints   []endpointItem    `json:"ai_endpoints"`
}
type networkInfo struct {
	Mode           string  `json:"mode"`
	AddressFamily  string  `json:"address_family"`
	TimeoutSeconds float64 `json:"timeout_seconds"`
	TimeoutScope   string  `json:"timeout_scope"`
	Workers        int     `json:"workers"`
}

func utc(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }
func makeReport(results []access.Result, timeout float64, workers int) report {
	r := report{SchemaVersion: 1, GeneratedAt: utc(time.Now()),
		Source: map[string]string{"repository": "https://github.com/remnawave/geocheck", "tag": "v0.3.0",
			"commit": "50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc", "port": "Go v0.4.0 standalone three-group extraction with conservative error handling"},
		Network: networkInfo{"direct", "system-selected", timeout, "request including redirects and body", workers},
		Results: make([]item, 0, len(results)),
	}
	for _, res := range results {
		platform := res.Check.Name
		if res.Check.ID == "chatgpt_web" || res.Check.ID == "chatgpt_app" {
			platform = "ChatGPT"
		}
		out := item{ID: res.Check.ID, Platform: platform, Name: res.Check.Name, Status: res.State.String(), Detail: res.Detail,
			CheckedAt: utc(res.CheckedAt), ElapsedMS: math.Round(float64(res.RTT)/float64(time.Millisecond)*1000) / 1000}
		if res.Region != "" {
			region := res.Region
			out.Region = &region
		}
		if res.State == access.StateError {
			message := res.Detail
			if res.Err != nil {
				message = res.Err.Error()
			}
			if message == "" {
				message = "could not reach a conclusion"
			}
			out.Error = &message
		}
		r.Results = append(r.Results, out)
	}
	return r
}

type measurements struct {
	Access   []access.Result
	Services []geo.Result
	AI       []ai.Result
}
type runner func(context.Context, time.Duration, int) (measurements, error)

func realRun(ctx context.Context, timeout time.Duration, workers int) (measurements, error) {
	stack, err := netx.New(ctx, netx.Options{Timeout: timeout})
	if err != nil {
		return measurements{}, err
	}
	defer stack.Close()
	return runGroups(ctx, stack, workers)
}

// Groups run consecutively through the same direct stack. This preserves the
// existing access path and makes --workers a GLOBAL bound, not a per-group multiplier.
func runGroups(ctx context.Context, stack *netx.Stack, workers int) (measurements, error) {
	m := measurements{Access: access.Run(ctx, stack, 0, access.Checks(), workers)}
	if ctx.Err() != nil {
		return m, ctx.Err()
	}
	m.Services = geo.Run(ctx, &geo.Env{Stack: stack}, geo.ServiceChecks(), workers)
	if ctx.Err() != nil {
		return m, ctx.Err()
	}
	m.AI = ai.Run(ctx, stack, 0, ai.Checks(), workers)
	return m, ctx.Err()
}

func runCLI(ctx context.Context, args []string, stdout, stderr io.Writer, run runner) int {
	flags := flag.NewFlagSet("service-unlock", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	asJSON := flags.Bool("json", false, "JSON")
	showLicense := flags.Bool("license", false, "show MIT license")
	timeout := flags.Float64("timeout", 6, "per-request timeout seconds")
	workers := flags.Int("workers", 4, "concurrency")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, help)
			return 0
		}
		fmt.Fprintln(stderr, "Invalid arguments:", err)
		return 2
	}
	if flags.NArg() != 0 || math.IsNaN(*timeout) || math.IsInf(*timeout, 0) || *timeout <= 0 || *timeout > 2147483.647 || *workers < 1 || *workers > 8 {
		fmt.Fprintln(stderr, "Invalid arguments: timeout must be in (0,2147483.647] seconds, workers must be 1-8, and positional arguments are not allowed")
		return 2
	}
	if *showLicense {
		fmt.Fprint(stdout, licenseText, render.ThirdPartyLicenses)
		return 0
	}
	duration := time.Duration(math.Ceil(*timeout*1000)) * time.Millisecond
	started := time.Now()
	results, err := run(ctx, duration, *workers)
	elapsed := time.Since(started)
	if ctx.Err() != nil {
		fmt.Fprintln(stderr, "Checks interrupted")
		return 130
	}
	if err != nil {
		fmt.Fprintln(stderr, "Check execution failed:", err)
		return 1
	}
	r := makeCompleteReport(results, *timeout, *workers)
	if *asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		err = encoder.Encode(r)
	} else {
		err = render.WriteAccess(stdout, render.Report{Access: results.Access, Services: results.Services, AI: results.AI, Duration: elapsed})
	}
	if err != nil {
		fmt.Fprintln(stderr, "Output failed:", err)
		return 1
	}
	for _, r := range results.Access {
		if r.State == access.StateError {
			return 1
		}
	}
	for _, r := range results.Services {
		if r.Err != nil {
			return 1
		}
	}
	for _, r := range results.AI {
		if r.State == ai.StateError {
			return 1
		}
	}
	return 0
}
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	var interrupted atomic.Int32
	go func() {
		select {
		case sig := <-signals:
			interrupted.Store(int32(128 + int(sig.(syscall.Signal))))
			cancel()
		case <-ctx.Done():
		}
	}()
	code := runCLI(ctx, os.Args[1:], os.Stdout, os.Stderr, realRun)
	cancel()
	signal.Stop(signals)
	if value := interrupted.Load(); value != 0 {
		code = int(value)
	}
	os.Exit(code)
}
