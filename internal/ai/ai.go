// Derived from remnawave/geocheck v0.3.0, commit 50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc.
// MIT License; Copyright (c) 2026 Remnawave.
// Package ai retains the four independent unauthenticated API probes without
// an existing web-client counterpart: DeepSeek, Qwen International, Kimi and GLM.
// Reachable means the endpoint answered without refusing the region; it is not
// evidence of authenticated model use. Web counterparts are checked only by access.
package ai

import (
	"context"
	"sync"
	"time"

	"serviceunlock/internal/netx"
)

// State is what the endpoint said about this address.
type State int

const (
	// StateReachable means the API answered and did not refuse the region.
	StateReachable State = iota
	// StateBlocked means it refused, naming the region or country.
	StateBlocked
	// StateError means the probe could not reach a conclusion.
	StateError
)

func (s State) String() string {
	switch s {
	case StateReachable:
		return "reachable"
	case StateBlocked:
		return "blocked"
	default:
		return "error"
	}
}

// Check is one endpoint probe.
type Check struct {
	ID   string
	Name string
	// Vendor groups endpoints that belong to the same company.
	Vendor string
	Run    func(ctx context.Context, env Env) Result
}

// Env is what a check needs to make its request.
type Env struct {
	Stack  *netx.Stack
	Family netx.Family
}

// Result is one endpoint's answer.
type Result struct {
	Check  Check
	State  State
	Status int
	// Detail is the reason, in the service's own words where it gave one.
	Detail    string
	RTT       time.Duration
	CheckedAt time.Time
	Err       error
}

// Run probes every endpoint concurrently.
func Run(ctx context.Context, stack *netx.Stack, f netx.Family, checks []Check, concurrency int) []Result {
	if concurrency <= 0 {
		concurrency = 6
	}

	results := make([]Result, len(checks))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, c := range checks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results[i] = Result{Check: c, State: StateError, Err: ctx.Err()}
				return
			}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				results[i] = Result{Check: c, State: StateError, Err: ctx.Err()}
				return
			}
			start := time.Now()
			res := c.Run(ctx, Env{Stack: stack, Family: f})
			res.Check = c
			res.CheckedAt = start
			if res.RTT == 0 {
				res.RTT = time.Since(start)
			}
			results[i] = res
		}()
	}
	wg.Wait()
	return results
}

// Summary counts the outcomes for the footer line.
type Summary struct {
	Total     int
	Reachable int
	Blocked   int
	Failed    int
}

// Summarize tallies results.
func Summarize(results []Result) Summary {
	s := Summary{Total: len(results)}
	for _, r := range results {
		switch r.State {
		case StateReachable:
			s.Reachable++
		case StateBlocked:
			s.Blocked++
		default:
			s.Failed++
		}
	}
	return s
}
