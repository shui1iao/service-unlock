// MIT License; Copyright (c) 2026 Remnawave. Offline endpoint checks.
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"serviceunlock/internal/netx"
	"strings"
	"testing"
	"time"
)

type tripFunc func(*http.Request) (*http.Response, error)

func (f tripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestAllEightEndpointsReachableWithoutCredentials(t *testing.T) {
	data, err := os.ReadFile("testdata/endpoints.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Checks []struct {
			ID, Name, Vendor, URL string
			Headers               map[string]string
		}
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	checks := Checks()
	if len(checks) != 8 || len(fixture.Checks) != 8 {
		t.Fatal("not eight")
	}
	for i, c := range checks {
		f := fixture.Checks[i]
		t.Run(c.ID, func(t *testing.T) {
			if c.ID != f.ID || c.Name != f.Name || c.Vendor != f.Vendor {
				t.Fatalf("%+v", c)
			}
			stack, err := netx.New(context.Background(), netx.Options{Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer stack.Close()
			calls := 0
			stack.HTTP(0).Transport = tripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.String() != f.URL {
					t.Error("wrong endpoint", req.Method, req.URL)
				}
				for name, value := range f.Headers {
					if req.Header.Get(name) != value {
						t.Error("wrong header", name)
					}
				}
				if req.Header.Get("Authorization") != "" || req.Header.Get("X-Api-Key") != "" || req.Header.Get("Cookie") != "" || req.Body != nil {
					t.Error("unexpected credentials/body")
				}
				return &http.Response{StatusCode: 401, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"missing credentials"}`)), ContentLength: -1, Request: req}, nil
			})
			r := Run(context.Background(), stack, 0, []Check{c}, 1)[0]
			if calls != 1 || r.State != StateReachable || r.Status != 401 || r.CheckedAt.IsZero() || r.RTT <= 0 {
				t.Fatalf("calls=%d %+v", calls, r)
			}
		})
	}
}
func TestAllAIStatusAndJSONGuards(t *testing.T) {
	for _, e := range catalog {
		t.Run(e.id, func(t *testing.T) {
			for _, status := range []int{200, 201, 401} {
				if r := e.classify(status, `{"data":[]}`); r.State != StateReachable {
					t.Fatalf("status%d: %+v", status, r)
				}
			}
			for _, status := range []int{429, 500, 502, 503, 599, 404, 418} {
				body := `{"error":"unsupported_country_region_territory Request not allowed"}`
				if status == 404 || status == 418 {
					body = `{"error":"unexpected"}`
				}
				if r := e.classify(status, body); r.State != StateError {
					t.Fatalf("status%d: %+v", status, r)
				}
			}
			for _, body := range []string{"", "{broken", "<html>upstream unavailable</html>"} {
				for _, status := range []int{200, 401} {
					if r := e.classify(status, body); r.State != StateError {
						t.Fatalf("status%d body%q: %+v", status, body, r)
					}
				}
			}
			want := StateError
			if e.id == "gemini" {
				want = StateReachable
			}
			if r := e.classify(403, `{"error":"unregistered caller"}`); r.State != want {
				t.Fatalf("403: %+v", r)
			}
			if r := e.classify(403, "cf_chl_opt request not allowed"); r.State != StateError {
				t.Fatalf("challenge: %+v", r)
			}
		})
	}
}
func TestAITransportFailureAndCancellation(t *testing.T) {
	for _, failure := range []error{errors.New("network failed"), context.DeadlineExceeded} {
		stack, _ := netx.New(context.Background(), netx.Options{Timeout: time.Second})
		stack.HTTP(0).Transport = tripFunc(func(*http.Request) (*http.Response, error) { return nil, failure })
		for _, r := range Run(context.Background(), stack, 0, Checks(), 2) {
			if r.State != StateError || r.Err == nil {
				t.Fatalf("%+v", r)
			}
		}
		stack.Close()
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, r := range Run(ctx, nil, 0, Checks(), 2) {
		if r.State != StateError || r.Err == nil {
			t.Fatal("cancelled check not unknown")
		}
	}
}
