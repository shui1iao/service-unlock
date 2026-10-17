// MIT License; Copyright (c) 2026 Remnawave. Offline fixtures only.
package access

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"serviceunlock/internal/netx"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func reply(req *http.Request, status int, body, final string) *http.Response {
	if final != "" {
		copy := req.Clone(req.Context())
		copy.URL, _ = url.Parse(final)
		req = copy
	}
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), ContentLength: -1, Request: req}
}
func fixture(req *http.Request) (*http.Response, error) {
	body, final := "ok", ""
	switch req.URL.Host {
	case "www.youtube.com":
		body = "ad-free"
	case "gemini.google.com":
		body = `,3,7,200,"FRA"`
	case "notebooklm.google.com":
		final = "https://accounts.google.com/login"
	case "claude.ai":
		if req.URL.Path == "/cdn-cgi/trace" {
			body = "loc=FR\n"
		}
	}
	return reply(req, 200, body, final), nil
}
func mockEnv(t *testing.T, f roundTripFunc) Env {
	t.Helper()
	env := testEnv(t)
	env.Stack.HTTP(0).Transport = f
	return env
}
func TestStandaloneExactScopeAndAvailable(t *testing.T) {
	env := mockEnv(t, fixture)
	ids := []string{"chatgpt_web", "chatgpt_app", "youtube_premium_access", "netflix_access", "claude_access", "tiktok_access", "gemini_access", "notebooklm_access"}
	checks := Checks()
	if len(checks) != 8 {
		t.Fatal(len(checks))
	}
	for i, c := range checks {
		if c.ID != ids[i] {
			t.Fatal(c.ID)
		}
		r := c.Run(context.Background(), env)
		if r.State != StateAvailable {
			t.Fatalf("%s: %+v", c.ID, r)
		}
	}
}
func TestStandaloneHTTPFailuresOverrideMarkers(t *testing.T) {
	body := `ad-free unsupported_country disallowed isp been blocked app unavailable in region not available in your region youtube premium is not available in your country ,3,7,200,"CHN"`
	for _, status := range []int{429, 500, 502, 503, 599} {
		env := mockEnv(t, func(req *http.Request) (*http.Response, error) { return reply(req, status, body, ""), nil })
		for _, c := range Checks() {
			if r := c.Run(context.Background(), env); r.State != StateError {
				t.Fatalf("%s HTTP%d: %+v", c.ID, status, r)
			}
		}
	}
}
func TestStandaloneChallengesAndNetworkFailures(t *testing.T) {
	for _, mode := range []string{"challenge", "network", "timeout"} {
		env := mockEnv(t, func(req *http.Request) (*http.Response, error) {
			if mode == "network" {
				return nil, fmt.Errorf("network unreachable")
			}
			if mode == "timeout" {
				return nil, context.DeadlineExceeded
			}
			return reply(req, 403, "cf_chl_opt /app-unavailable-in-region unsupported_country disallowed isp not available in your region", ""), nil
		})
		for _, c := range Checks() {
			if r := c.Run(context.Background(), env); r.State != StateError {
				t.Fatalf("%s %s: %+v", c.ID, mode, r)
			}
		}
	}
}
func TestStandaloneService403Semantics(t *testing.T) {
	markers := map[string]string{"chatgpt_web": "unsupported_country", "chatgpt_app": "disallowed isp", "youtube_premium_access": "youtube premium is not available in your country", "claude_access": "unfortunately, claude isn&#39;t available here.", "tiktok_access": "not available in your region"}
	for _, c := range Checks() {
		if marker, ok := markers[c.ID]; ok {
			env := mockEnv(t, func(req *http.Request) (*http.Response, error) { return reply(req, 403, marker, ""), nil })
			if r := c.Run(context.Background(), env); r.State != StateBlocked {
				t.Fatalf("%s: %+v", c.ID, r)
			}
		}
		env := mockEnv(t, func(req *http.Request) (*http.Response, error) { return reply(req, 403, "Forbidden", ""), nil })
		if r := c.Run(context.Background(), env); r.State != StateError {
			t.Fatalf("bare403 %s: %+v", c.ID, r)
		}
	}
}
func TestStandaloneNetflixFourStates(t *testing.T) {
	for _, tc := range []struct {
		licensed, original int
		want               State
	}{{200, 200, StateAvailable}, {404, 200, StateRestricted}, {404, 404, StateBlocked}, {500, 200, StateError}, {404, 503, StateError}, {403, 200, StateError}} {
		env := mockEnv(t, func(req *http.Request) (*http.Response, error) {
			status := tc.original
			if strings.HasSuffix(req.URL.Path, netflixLicensedTitle) {
				status = tc.licensed
			}
			return reply(req, status, "ok", ""), nil
		})
		if r := netflix().Run(context.Background(), env); r.State != tc.want {
			t.Fatalf("%+v: %+v", tc, r)
		}
	}
}
func TestStandaloneNetflixInvalidDestinations(t *testing.T) {
	for _, final := range []string{"https://www.netflix.com/login", "https://www.netflix.com/fr/login", "https://www.netflix.com/title/123", "https://example.com/title/70143836"} {
		env := mockEnv(t, func(req *http.Request) (*http.Response, error) { return reply(req, 200, "ok", final), nil })
		if r := netflix().Run(context.Background(), env); r.State != StateError {
			t.Fatalf("%s: %+v", final, r)
		}
	}
}
func TestStandaloneEmptyTextAndOptionalTrace(t *testing.T) {
	env := mockEnv(t, func(req *http.Request) (*http.Response, error) { return reply(req, 200, "", ""), nil })
	for _, c := range Checks() {
		if c.ID == "netflix_access" {
			continue
		}
		if r := c.Run(context.Background(), env); r.State != StateError {
			t.Fatalf("%s: %+v", c.ID, r)
		}
	}
	env = mockEnv(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/cdn-cgi/trace" {
			return nil, context.DeadlineExceeded
		}
		return fixture(req)
	})
	if r := claude().Run(context.Background(), env); r.State != StateAvailable || r.Region != "" {
		t.Fatalf("%+v", r)
	}
}
func TestStandaloneNotebookLMGuards(t *testing.T) {
	for _, final := range []string{"https://accounts.google.com/login", "https://notebook.google/?location=unsupported"} {
		for _, status := range []int{429, 500, 503} {
			if r := classifyNotebookLM(status, final, ""); r.State != StateError {
				t.Fatalf("%+v", r)
			}
		}
	}
	for _, final := range []string{"", "https://[", "/relative", "https://user@accounts.google.com/login", "https://accounts.google.com:99999/login"} {
		if r := classifyNotebookLM(200, final, ""); r.State != StateError {
			t.Fatalf("%s: %+v", final, r)
		}
	}
}
func TestStandaloneConcurrencyAndCancellation(t *testing.T) {
	var active, peak atomic.Int32
	checks := make([]Check, 8)
	for i := range checks {
		checks[i] = Check{ID: fmt.Sprint(i), Run: func(ctx context.Context, _ Env) Result {
			n := active.Add(1)
			defer active.Add(-1)
			for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
			}
			time.Sleep(5 * time.Millisecond)
			return Result{State: StateAvailable}
		}}
	}
	results := Run(context.Background(), nil, 0, checks, 2)
	if peak.Load() != 2 || active.Load() != 0 {
		t.Fatalf("peak=%d active=%d", peak.Load(), active.Load())
	}
	for i, r := range results {
		if r.Check.ID != fmt.Sprint(i) || r.CheckedAt.IsZero() || r.RTT <= 0 {
			t.Fatalf("%+v", r)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, r := range Run(ctx, nil, 0, checks, 2) {
		if r.State != StateError {
			t.Fatalf("cancelled check: %+v", r)
		}
	}
}
func TestValidDestinationInvalidPort(t *testing.T) {
	// url.Parse accepts numeric ports outside uint16: do not treat these as login evidence.
	if _, err := validDestination("https://accounts.google.com:99999/login"); err == nil {
		t.Fatal("invalid port accepted")
	}
	_ = netx.V4
}
