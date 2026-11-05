// Consumer-service execution adapted from geocheck v0.3.0 internal/geo/geo.go.
// MIT License; Copyright (c) 2026 Remnawave.
// Deliberately no public-IP discovery, GeoIP/CDN catalogues, or family probing.
package geo

import (
	"context"
	"fmt"
	"serviceunlock/internal/countries"
	"serviceunlock/internal/netx"
	"strings"
	"sync"
	"time"
)

type Group string

const GroupServices Group = "services"

type Kind int

const (
	KindCountry Kind = iota
	KindAvailability
	KindBlocked
)

func (k Kind) String() string {
	switch k {
	case KindCountry:
		return "country"
	case KindAvailability:
		return "availability"
	default:
		return "blocked"
	}
}

type RunFunc func(context.Context, *Env, netx.Family) (string, error)
type Check struct {
	ID, Name string
	Group    Group
	Kind     Kind
	Run      RunFunc
}
type Result struct {
	Check         Check
	Value, Detail string
	Region        string
	Err           error
	CheckedAt     time.Time
	RTT           time.Duration
}
type Env struct {
	Stack   *netx.Stack
	mu      sync.RWMutex
	details map[string]string
}

func (e *Env) note(key, text string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.details == nil {
		e.details = map[string]string{}
	}
	e.details[key] = text
}
func (e *Env) detail(key string) string { e.mu.RLock(); defer e.mu.RUnlock(); return e.details[key] }

// Guard only the new service group: existing access and endpoint-specific AI
// status handling remains untouched. Exceptions must be explicit at call sites.
func (e *Env) request(ctx context.Context, f netx.Family, req netx.Request, allowed ...int) (*netx.Response, error) {
	resp, err := e.Stack.Do(ctx, f, req)
	if err != nil {
		return nil, err
	}
	lower := strings.ToLower(resp.Text())
	if resp.Status == 403 || resp.Status == 429 || resp.Status == 503 {
		for _, marker := range []string{"cf_chl_opt", "_cf_chl", "just a moment", "cf-browser-verification", "enable javascript and cookies to continue"} {
			if strings.Contains(lower, marker) {
				return nil, fmt.Errorf("challenged before reaching the service")
			}
		}
	}
	if resp.Status == 429 || resp.Status >= 500 || resp.Status < 200 {
		return nil, fmt.Errorf("unexpected HTTP %d", resp.Status)
	}
	if !resp.OK() {
		ok := false
		for _, code := range allowed {
			if resp.Status == code {
				ok = true
			}
		}
		if !ok {
			return nil, fmt.Errorf("unexpected HTTP %d", resp.Status)
		}
	}
	if !req.HeadOnly && strings.TrimSpace(resp.Text()) == "" {
		return nil, fmt.Errorf("empty response; no service evidence")
	}
	return resp, nil
}
func normalize(value string, kind Kind) string {
	value = strings.TrimSpace(value)
	if kind != KindCountry {
		value = strings.ToLower(value)
		if value == "yes" || value == "no" {
			return value
		}
		return ""
	}
	// Keep malformed values and Worldwide from being reported as a country.
	value = strings.ToUpper(value)
	if len(value) == 2 && countries.Known(value) && value != "WW" {
		return value
	}
	return ""
}

// Run executes only the registered independent probes, with one system-selected outcome.
func Run(ctx context.Context, env *Env, checks []Check, workers int) []Result {
	if workers < 1 {
		workers = 4
	}
	results := make([]Result, len(checks))
	indices := make([]int, len(checks))
	for i, c := range checks {
		results[i].Check = c
		indices[i] = i
	}
	wave := func(indices []int) {
		sem := make(chan struct{}, workers)
		var wg sync.WaitGroup
		for _, i := range indices {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					results[i].Err = ctx.Err()
					return
				}
				defer func() { <-sem }()
				if ctx.Err() != nil {
					results[i].Err = ctx.Err()
					return
				}
				start := time.Now()
				c := checks[i]
				raw, err := c.Run(ctx, env, 0)
				r := Result{Check: c, CheckedAt: start, RTT: time.Since(start), Detail: env.detail(c.ID), Err: err}
				if err == nil {
					r.Value = normalize(raw, c.Kind)
					if r.Value == "" {
						r.Err = fmt.Errorf("could not parse a %s result", c.Kind)
					}
				}
				if r.Detail == "" {
					switch c.Kind {
					case KindCountry:
						r.Detail = "served region; not an access verdict"
					case KindAvailability:
						r.Detail = "service availability observation; not a login/playback test"
					case KindBlocked:
						r.Detail = "yes = captcha/block observed; no = marker not observed"
					}
				}
				if c.Kind != KindCountry {
					r.Region = RegionMetadata(ctx, env.Stack, 0, c.ID)
				}
				results[i] = r
			}(i)
		}
		wg.Wait()
	}
	wave(indices)
	return results
}
