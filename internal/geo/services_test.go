// MIT License; Copyright (c) 2026 Remnawave. All transport is injected/offline.
package geo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"serviceunlock/internal/netx"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type tripFunc func(*http.Request) (*http.Response, error)

func (f tripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type fixture struct {
	ID, Name, Kind, URL, Method, Body, Value string
	Headers                                  map[string]string
	RequestForm                              map[string]string `json:"request_form"`
	RequestJSON                              json.RawMessage   `json:"request_json"`
	ResponseHeaders                          http.Header       `json:"response_headers"`
}

func fixtures(t *testing.T) []fixture {
	t.Helper()
	data, err := os.ReadFile("testdata/services.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct{ Checks []fixture }
	if err = json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	return f.Checks
}
func response(req *http.Request, status int, body string, headers http.Header) *http.Response {
	if headers == nil {
		headers = make(http.Header)
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), ContentLength: -1, Header: headers, Request: req}
}
func environment(t *testing.T, f tripFunc) *Env {
	t.Helper()
	stack, err := netx.New(context.Background(), netx.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	stack.HTTP(0).Transport = f
	t.Cleanup(stack.Close)
	return &Env{Stack: stack}
}
func single(t *testing.T, c Check, f tripFunc) Result {
	t.Helper()
	return Run(context.Background(), environment(t, f), []Check{c}, 1)[0]
}
func check(t *testing.T, id string) Check {
	t.Helper()
	for _, c := range ServiceChecks() {
		if c.ID == id {
			return c
		}
	}
	t.Fatal(id)
	return Check{}
}
func TestAll21CatalogueTypesMethodsPayloadsAndParsers(t *testing.T) {
	fs, checks := fixtures(t), ServiceChecks()
	if len(fs) != 21 || len(checks) != 21 {
		t.Fatal("not 21")
	}
	for i, c := range checks {
		f := fs[i]
		t.Run(c.ID, func(t *testing.T) {
			if c.ID != f.ID || c.Name != f.Name || c.Kind.String() != f.Kind || c.Group != GroupServices {
				t.Fatalf("catalogue mismatch: %+v %+v", c, f)
			}
			calls := 0
			r := single(t, c, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.String() != f.URL || req.Method != f.Method {
					t.Errorf("endpoint/method mismatch: %s %s", req.Method, req.URL)
				}
				for key, value := range f.Headers {
					if req.Header.Get(key) != value {
						t.Errorf("%s header changed", key)
					}
				}
				if !strings.Contains(req.Header.Get("User-Agent"), "Chrome/151.0.0.0") {
					t.Error("upstream UA lost")
				}
				if req.Header.Get("Authorization") != "" {
					t.Error("user credential attached")
				}
				var body []byte
				if req.Body != nil {
					body, _ = io.ReadAll(req.Body)
				}
				if len(f.RequestJSON) > 0 {
					var actual, want any
					json.Unmarshal(body, &actual)
					json.Unmarshal(f.RequestJSON, &want)
					if !reflect.DeepEqual(actual, want) {
						t.Error("POST JSON differs", string(body))
					}
				} else if len(f.RequestForm) > 0 {
					form, err := url.ParseQuery(string(body))
					if err != nil {
						t.Fatal(err)
					}
					actual := map[string]string{}
					for key, values := range form {
						if len(values) != 1 {
							t.Error("duplicate form value")
						}
						actual[key] = form.Get(key)
					}
					if !reflect.DeepEqual(actual, f.RequestForm) {
						t.Error("POST validation form differs")
					}
				} else if len(body) > 0 {
					t.Error("unexpected payload")
				}
				return response(req, 200, f.Body, f.ResponseHeaders), nil
			})
			if calls != 1 || r.Err != nil || r.Value != f.Value || r.CheckedAt.IsZero() || r.RTT <= 0 {
				t.Fatalf("calls=%d result=%+v", calls, r)
			}
		})
	}
}
func TestAllServicesErrorsCannotBecomeObservations(t *testing.T) {
	for _, c := range ServiceChecks() {
		for _, mode := range []string{"network", "timeout", "429", "500", "503", "challenge"} {
			t.Run(c.ID+"/"+mode, func(t *testing.T) {
				r := single(t, c, func(req *http.Request) (*http.Response, error) {
					switch mode {
					case "network":
						return nil, errors.New("offline network error")
					case "timeout":
						return nil, context.DeadlineExceeded
					}
					status := 500
					switch mode {
					case "429":
						status = 429
					case "503":
						status = 503
					case "challenge":
						status = 403
					}
					return response(req, status, `cf_chl_opt unusual traffic from "en_US" "countryCode":"US" {"status":1}`, nil), nil
				})
				if r.Err == nil || r.Value != "" {
					t.Fatalf("false result: %+v", r)
				}
			})
		}
	}
}
func TestServiceSpecificRefusalAndUnknownSemantics(t *testing.T) {
	cases := []struct {
		id         string
		status     int
		body, want string
		unknown    bool
	}{
		{"google_captcha", 403, "unusual traffic from this network", "yes", false}, {"google_captcha", 200, "ordinary page", "no", false}, {"google_captcha", 403, "Forbidden", "", true},
		{"youtube_premium", 403, "YouTube Premium is not available in your country", "no", false}, {"youtube_music", 403, "YouTube Music is not available in your area", "no", false},
		{"youtube_premium", 403, "Forbidden", "", true}, {"youtube_music", 200, " ", "", true},
		{"spotify_signup", 200, `{"status":120}`, "no", false}, {"spotify_signup", 200, `{"status":320}`, "no", false}, {"spotify_signup", 200, `{"is_country_launched":false}`, "no", false},
		{"spotify_signup", 200, `{"status":"invalid"}`, "", true}, {"spotify_signup", 200, `{"is_country_launched":"invalid"}`, "", true},
		{"reddit_guest", 403, "Forbidden", "no", false}, {"reddit_guest", 404, "not found", "", true}, {"reddit_guest", 200, "", "", true},
		{"reddit", 200, "blocked by network security", "", true}, {"apple", 200, "US123", "", true}, {"apple", 200, "ZZ", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.id+fmt.Sprint(tc.status, tc.body), func(t *testing.T) {
			r := single(t, check(t, tc.id), func(req *http.Request) (*http.Response, error) { return response(req, tc.status, tc.body, nil), nil })
			if (r.Err != nil) != tc.unknown || r.Value != tc.want {
				t.Fatalf("%+v", r)
			}
		})
	}
}
func TestJSONAndMissingCountryParseFailures(t *testing.T) {
	for _, id := range []string{"twitch", "chatgpt", "netflix", "spotify", "spotify_signup", "deezer", "reddit", "prime", "apple", "steam", "playstation", "tiktok", "ookla", "jetbrains", "bing"} {
		for _, body := range []string{"{}", "{broken", "", "<html>no marker</html>"} {
			r := single(t, check(t, id), func(req *http.Request) (*http.Response, error) { return response(req, 200, body, nil), nil })
			if r.Err == nil {
				t.Fatalf("%s body=%q: %+v", id, body, r)
			}
		}
	}
}
func TestUpstreamFallbacksAreExplicitAndDoNotHideHTTPFailure(t *testing.T) {
	env := environment(t, func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "www.google.com":
			return response(req, 200, "no locale marker", nil), nil
		case "play.google.com":
			return response(req, 200, `<div class="yVZQTb">United States`, nil), nil
		case "www.youtube.com":
			return response(req, 200, "no country marker", nil), nil
		}
		t.Errorf("unexpected fallback URL %s", req.URL)
		return nil, errors.New("unexpected request")
	})
	r := Run(context.Background(), env, []Check{youtube(), google()}, 2)
	if r[0].Value != "US" || r[1].Value != "US" || !strings.Contains(r[0].Detail, "Google served") || !strings.Contains(r[1].Detail, "Google Play") {
		t.Fatalf("%+v", r)
	}
	for _, host := range []string{"www.google.com", "www.youtube.com"} {
		env = environment(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == host {
				return response(req, 500, "error", nil), nil
			}
			return response(req, 200, `"en_US"`, nil), nil
		})
		env.share("google", 0, "US")
		id := "google"
		if host == "www.youtube.com" {
			id = "youtube"
		}
		got := Run(context.Background(), env, []Check{check(t, id)}, 1)[0]
		if got.Err == nil {
			t.Fatalf("HTTP failure hidden: %+v", got)
		}
	}
	for _, body := range []string{`Region: "WW"`, "no market"} {
		got := single(t, bing(), func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "login.live.com" {
				return response(req, 200, `"sRequestCountry":"GB"`, nil), nil
			}
			return response(req, 200, body, nil), nil
		})
		if got.Value != "GB" || !strings.Contains(got.Detail, "Microsoft account") {
			t.Fatalf("%+v", got)
		}
	}
}
func TestDependencyOrderConcurrencyAndCancelledChecks(t *testing.T) {
	var active, peak atomic.Int32
	checks := ServiceChecks()
	env := &Env{}
	for i := range checks {
		c := &checks[i]
		id := c.ID
		kind := c.Kind
		c.Run = func(ctx context.Context, e *Env, _ netx.Family) (string, error) {
			n := active.Add(1)
			defer active.Add(-1)
			for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
			}
			if id == "youtube" && e.lookupShared("google", 0) != "US" {
				t.Error("Google dependency not complete")
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Millisecond):
			}
			if kind == KindCountry {
				return "US", nil
			}
			return "yes", nil
		}
	}
	got := Run(context.Background(), env, checks, 2)
	if peak.Load() != 2 || active.Load() != 0 || len(got) != 21 {
		t.Fatal(peak.Load(), active.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, r := range Run(ctx, &Env{}, checks, 2) {
		if r.Err == nil {
			t.Fatal("cancelled result not unknown")
		}
	}
}
