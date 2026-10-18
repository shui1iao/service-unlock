package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"serviceunlock/internal/netx"
)

func deepSeekGovernorFixture(t *testing.T) (endpoint, string) {
	t.Helper()
	body, err := os.ReadFile("testdata/deepseek-governor-auth.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != 31 || string(body) != "Authentication Fails (governor)" {
		t.Fatal("captured non-secret authentication fixture changed")
	}
	for _, e := range catalog {
		if e.id == "deepseek" {
			return e, string(body)
		}
	}
	t.Fatal("DeepSeek endpoint missing")
	return endpoint{}, ""
}

func TestDeepSeekGovernorPlainAuthentication(t *testing.T) {
	e, fixture := deepSeekGovernorFixture(t)
	for _, body := range []string{fixture, " \t\r\n" + fixture + "\n\t "} {
		r := e.classify(401, body)
		if r.State != StateReachable || r.Status != 401 || r.Detail != "authentication endpoint responded without credentials; model calls not tested" {
			t.Fatalf("expected scoped authentication reachability: %+v", r)
		}
	}
}

func TestDeepSeekGovernorStrictGuards(t *testing.T) {
	e, fixture := deepSeekGovernorFixture(t)
	for _, body := range []string{
		"", " \r\n", strings.ToLower(fixture), "Authentication fails (governor)",
		"Authentication Fails", fixture + ".", fixture + " maintenance", "prefix " + fixture,
		"<html>" + fixture + "</html>", fixture + " cf_chl_opt", fixture + " _cf_chl",
		fixture + " Just a moment", fixture + " cf-browser-verification", fixture + "\x00",
	} {
		if r := e.classify(401, body); r.State != StateError {
			t.Fatalf("non-exact body accepted: %q %+v", body, r)
		}
	}
	for _, status := range []int{0, 199, 200, 201, 301, 302, 307, 308, 400, 403, 404, 418, 429, 500, 503, 599} {
		if r := e.classify(status, fixture); r.State != StateError || r.Status != status {
			t.Fatalf("non-401 accepted: %+v", r)
		}
	}
	for _, id := range []string{"", "DeepSeek", "openai", "anthropic", "gemini", "qwen_intl", "qwen_cn", "moonshot", "zhipu"} {
		other := e
		other.id = id
		if r := other.classify(401, fixture); r.State != StateError {
			t.Fatalf("wrong id %q accepted: %+v", id, r)
		}
	}
	for _, url := range []string{"", "http://api.deepseek.com/models", "https://api.deepseek.com/v1/models", "https://api.deepseek.com/models/", "https://api.deepseek.com/models?x=1", "https://example.com/models"} {
		other := e
		other.url = url
		if r := other.classify(401, fixture); r.State != StateError {
			t.Fatalf("wrong URL %q accepted: %+v", url, r)
		}
	}
	other := e
	other.vendor = "OpenAI"
	if r := other.classify(401, fixture); r.State != StateError {
		t.Fatalf("wrong vendor accepted: %+v", r)
	}
	for _, other := range catalog {
		if other.id != "deepseek" && other.classify(401, fixture).State != StateError {
			t.Fatalf("other vendor %q accepted the plain-text exception", other.id)
		}
	}
}

func TestDeepSeekGovernorRunBoundaries(t *testing.T) {
	e, fixture := deepSeekGovernorFixture(t)
	for _, tc := range []struct {
		name string
		mode string
		want State
	}{
		{"official_401", "ok", StateReachable},
		{"header_challenge", "challenge", StateError},
		{"redirected_401", "redirect", StateError},
		{"incomplete_redirect", "incomplete", StateError},
		{"network_failure", "network", StateError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stack, err := netx.New(context.Background(), netx.Options{Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer stack.Close()
			calls := 0
			stack.HTTP(0).Transport = tripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.Header.Get("Authorization") != "" || req.Header.Get("X-Api-Key") != "" || req.Header.Get("Cookie") != "" || req.Body != nil {
					t.Error("probe must remain an unauthenticated read-only GET")
				}
				if tc.mode == "network" {
					return nil, errors.New("offline network failure")
				}
				header := make(http.Header)
				status := 401
				if tc.mode == "challenge" {
					header.Set("Cf-Mitigated", "challenge")
				}
				if tc.mode == "redirect" && calls == 1 {
					status = 302
					header.Set("Location", "https://api.deepseek.com/unexpected-login")
				}
				if tc.mode == "incomplete" {
					status = 302
				}
				return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(fixture)), ContentLength: int64(len(fixture)), Request: req}, nil
			})
			r := Run(context.Background(), stack, 0, []Check{e.check()}, 1)[0]
			if r.State != tc.want || r.CheckedAt.IsZero() || r.RTT <= 0 {
				t.Fatalf("%s: %+v", tc.name, r)
			}
			wantCalls := 1
			if tc.mode == "redirect" {
				wantCalls = 2
			}
			if calls != wantCalls {
				t.Fatalf("unexpected request count %d", calls)
			}
		})
	}
}
