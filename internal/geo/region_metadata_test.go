package geo

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"serviceunlock/internal/netx"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var metadataCases = []struct{ id, url, method, body string }{
	{"chatgpt_web", "https://ab.chatgpt.com/v1/initialize", "POST", `{"derived_fields":{"country":"JP"}}`},
	{"tiktok_access", "https://www.tiktok.com/api/v1/web-cookie-privacy/config?appId=1988", "GET", `{"body":{"appProps":{"region":"jp"}}}`},
	{"youtube_premium_access", "https://www.youtube.com", "GET", `{"countryCode":"JP"}`},
	{"spotify_signup", "https://accounts.spotify.com/status", "GET", `{"geoLocationCountryCode":"JP"}`},
	{"reddit_guest", "https://www.reddit.com/svc/shreddit/reddit-chat", "GET", `<widget country="JP">`},
	{"google_captcha", "https://www.google.com", "GET", `"ja_JP"`},
}

func TestSixOfficialAuxiliaryRequestsAndNoGoogleYouTubeFallback(t *testing.T) {
	for _, tc := range metadataCases {
		t.Run(tc.id, func(t *testing.T) {
			calls := 0
			env := environment(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.String() != tc.url || req.Method != tc.method {
					t.Fatal("unexpected metadata request", req.Method, req.URL)
				}
				if req.Header.Get("Authorization") != "" || req.Header.Get("Cookie") != "" || !strings.Contains(req.UserAgent(), "Chrome/151.0.0.0") {
					t.Fatal("auth/UA changed")
				}
				if tc.id == "chatgpt_web" && req.Header.Get("Statsig-Api-Key") != "client-zUdXdSTygXJdzoE0sWTkP8GKTVsUMF2IRM7ShVO2JAG" {
					t.Fatal("public client identifier changed")
				}
				return response(req, 200, tc.body, nil), nil
			})
			if got := RegionMetadata(context.Background(), env.Stack, 0, tc.id); got != "JP" || calls != 1 {
				t.Fatal(got, calls)
			}
			// Same service success with no country must NOT query another platform.
			env.Stack.HTTP(0).Transport = tripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if tc.id == "youtube_premium_access" && req.URL.String() != tc.url {
					t.Fatal("Google fallback")
				}
				return response(req, 200, "no region", nil), nil
			})
			if got := RegionMetadata(context.Background(), env.Stack, 0, tc.id); got != "" {
				t.Fatal("invented country", got)
			}
		})
	}
}
func TestMetadataMalformedConflictWrongTypeAndChallengeStayUnknown(t *testing.T) {
	for _, tc := range metadataCases {
		for _, mode := range []string{"network", "http403", "http429", "http500", "challenge200", "mitigated", "empty", "cross-origin", "wrong-code", "conflict", "bad-type", "bad-json"} {
			t.Run(tc.id+"/"+mode, func(t *testing.T) {
				env := environment(t, func(req *http.Request) (*http.Response, error) {
					status, body := 200, tc.body
					headers := http.Header{}
					switch mode {
					case "network":
						return nil, errors.New("not reflected-private-response")
					case "http403":
						status = 403
					case "http429":
						status = 429
					case "http500":
						status = 500
					case "challenge200":
						body = "just a moment " + body
					case "mitigated":
						headers.Set("Cf-Mitigated", "challenge")
					case "empty":
						body = ""
					case "wrong-code":
						body = strings.ReplaceAll(strings.ReplaceAll(body, "JP", "WW"), "jp", "WW")
					case "conflict":
						switch tc.id {
						case "chatgpt_web":
							body = `{"derived_fields":{"country":"JP","country":"SG"}}`
						case "tiktok_access":
							body = `{"body":{"appProps":{"region":"JP","region":"SG"}}}`
						case "spotify_signup":
							body = `{"geoLocationCountryCode":"JP","geoLocationCountryCode":"SG"}`
						case "youtube_premium_access":
							body = `"countryCode":"JP","countryCode":"SG"`
						case "reddit_guest":
							body = `country="JP" country="SG"`
						case "google_captcha":
							body = `"ja_JP" "en_SG"`
						}
					case "bad-type":
						body = `{"derived_fields":{"country":42},"body":{"appProps":{"region":["JP"]}},"geoLocationCountryCode":true,"countryCode":42}`
					case "bad-json":
						body = `{"derived_fields":{"country":"JP"}} trailing`
					}
					r := response(req, status, body, headers)
					if mode == "cross-origin" {
						r.Request = req.Clone(req.Context())
						r.Request.URL, _ = url.Parse("https://unrelated.invalid/country")
					}
					return r, nil
				})
				if got := RegionMetadata(context.Background(), env.Stack, 0, tc.id); got != "" {
					t.Fatal("untrusted metadata accepted", mode, got)
				}
			})
		}
	}
	for _, body := range []string{`{"derived_fields":{"country":"JP"},"derived_fields":{"country":"SG"}}`, `{"derived_fields":{"country":"JP"}} {}`, `{"derived_fields":{"country":"ZZ"}}`, `{"derived_fields":{"country":"EU"}}`, `{"derived_fields":{"country":"UK"}}`, `{"derived_fields":{"country":"USA"}}`, `{"derived_fields":{"country":null}}`} {
		if metadataJSONCountry([]byte(body), "derived_fields", "country") != "" {
			t.Fatal("malformed metadata", body)
		}
	}
}
func TestGoogleFallbackSharesOneAuxiliaryBudget(t *testing.T) {
	for _, slow := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "shared-budget"}[slow], func(t *testing.T) {
			stack, _ := netx.New(context.Background(), netx.Options{Timeout: 90 * time.Millisecond})
			defer stack.Close()
			calls := 0
			stack.HTTP(0).Transport = tripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if slow {
					select {
					case <-req.Context().Done():
						return nil, req.Context().Err()
					case <-time.After(60 * time.Millisecond):
					}
				}
				if calls == 1 {
					if req.URL.String() != "https://www.google.com" {
						t.Fatal(req.URL)
					}
					return response(req, 200, "no locale", nil), nil
				}
				if req.URL.String() != "https://play.google.com/" || req.Header.Get("Accept-Language") != "en-US;q=0.9" {
					t.Fatal(req.URL)
				}
				return response(req, 200, `<div class="yVZQTb">Japan</div>`, nil), nil
			})
			start := time.Now()
			got := RegionMetadata(context.Background(), stack, 0, "google_captcha")
			elapsed := time.Since(start)
			if calls != 2 || !slow && got != "JP" || slow && (got != "" || elapsed > 150*time.Millisecond) {
				t.Fatal(got, calls, elapsed)
			}
		})
	}
}
func TestAuxiliaryCancelUnknownAndNoQueryForUnrelatedCheck(t *testing.T) {
	env := environment(t, func(req *http.Request) (*http.Response, error) {
		t.Fatal("query not allowed", req.URL)
		return nil, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, id := range []string{"chatgpt_web", "tiktok_access", "youtube_premium_access", "spotify_signup", "reddit_guest", "google_captcha", "prime", "deepseek", "netflix_access"} {
		if RegionMetadata(ctx, env.Stack, 0, id) != "" {
			t.Fatal(id)
		}
	}
	env.Stack.HTTP(0).Transport = tripFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	ctx, cancel = context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)
	start := time.Now()
	if got := RegionMetadata(ctx, env.Stack, 0, "chatgpt_web"); got != "" || time.Since(start) > 200*time.Millisecond {
		t.Fatal(got)
	}
}
func TestAuxiliaryNeverVotesForPrimaryOrRTTAndStaysInWorker(t *testing.T) {
	for _, status := range []string{"yes", "no", "unknown"} {
		t.Run(status, func(t *testing.T) {
			env := environment(t, func(req *http.Request) (*http.Response, error) {
				time.Sleep(40 * time.Millisecond)
				return response(req, 200, `{"geoLocationCountryCode":"JP"}`, nil), nil
			})
			primaryErr := errors.New("primary sentinel")
			c := Check{ID: "spotify_signup", Kind: KindAvailability, Run: func(ctx context.Context, e *Env, f netx.Family) (string, error) {
				e.note("spotify_signup", "primary detail")
				if status == "unknown" {
					return "", primaryErr
				}
				return status, nil
			}}
			start := time.Now()
			r := Run(context.Background(), env, []Check{c}, 4)[0]
			if r.Region != "JP" || r.Detail != "primary detail" || r.RTT >= 30*time.Millisecond || r.CheckedAt.Before(start) || time.Since(start) < 40*time.Millisecond {
				t.Fatal(r)
			}
			if status == "unknown" {
				if r.Err != primaryErr || r.Value != "" {
					t.Fatal("metadata masked primary", r)
				}
			} else if r.Err != nil || r.Value != status {
				t.Fatal(r)
			}
		})
	}
	var active, peak atomic.Int32
	env := environment(t, func(req *http.Request) (*http.Response, error) {
		n := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
		}
		time.Sleep(10 * time.Millisecond)
		return response(req, 200, `{"geoLocationCountryCode":"JP"}`, nil), nil
	})
	checks := make([]Check, 12)
	for i := range checks {
		checks[i] = Check{ID: "spotify_signup", Kind: KindAvailability, Run: func(context.Context, *Env, netx.Family) (string, error) { return "no", nil }}
	}
	Run(context.Background(), env, checks, 4)
	if active.Load() != 0 || peak.Load() != 4 {
		t.Fatal("metadata escaped global workers", active.Load(), peak.Load())
	}
}

func TestPrimaryYouTubeMetadataRejectsChallengeOriginAndStatus(t *testing.T) {
	for _, tc := range []struct {
		status              int
		body, final, header string
	}{{200, `ad-free just a moment "countryCode":"JP"`, "https://www.youtube.com/premium", ""}, {403, `"countryCode":"JP"`, "https://www.youtube.com/premium", ""}, {200, `"countryCode":"JP"`, "https://evil.invalid/", ""}, {200, `"countryCode":"JP"`, "https://www.youtube.com/premium", "challenge"}} {
		if got := YouTubePrimaryRegion(&netx.Response{Status: tc.status, Body: []byte(tc.body), FinalURL: tc.final, Header: http.Header{"Cf-Mitigated": []string{tc.header}}}); got != "" {
			t.Fatal("untrusted primary metadata", got)
		}
	}
}

func TestEveryAuxiliaryOwnTimeoutIsBoundedUnknown(t *testing.T) {
	for _, tc := range metadataCases {
		t.Run(tc.id, func(t *testing.T) {
			stack, _ := netx.New(context.Background(), netx.Options{Timeout: 25 * time.Millisecond})
			defer stack.Close()
			calls := 0
			stack.HTTP(0).Transport = tripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				<-r.Context().Done()
				return nil, r.Context().Err()
			})
			start := time.Now()
			got := RegionMetadata(context.Background(), stack, 0, tc.id)
			if got != "" || calls != 1 || time.Since(start) > 150*time.Millisecond {
				t.Fatal("aux deadline/unknown", got, calls, time.Since(start))
			}
		})
	}
}
