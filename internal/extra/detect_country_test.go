// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func countryReply(req *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}
}
func TestDetectCountryActualRequestPathsAndFallback(t *testing.T) {
	for _, tc := range []struct {
		name, primary, fallback, country, source string
		primaryStatus                            int
		calls                                    int
	}{
		{"primaryJP", "h=www.cloudflare.com\nloc=JP\nip=203.0.113.99\n", `{"country_code":"US"}`, "JP", CountryTraceURL, 200, 1},
		{"fallbackDE", "invalid trace", `{"country_code":"DE","country":"Germany","ip":"203.0.113.99","asn":12345}`, "DE", CountryFallbackURL, 200, 2},
		{"primary403", "h=www.cloudflare.com\nloc=JP", `{"country_code":"HK"}`, "HK", CountryFallbackURL, 403, 2},
		{"unsupportedValidCN", "h=www.cloudflare.com\nloc=CN", `{"country_code":"US"}`, "CN", CountryTraceURL, 200, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient(4, time.Second)
			defer c.Close()
			count := 0
			c.HTTP.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
				count++
				if req.Method != "GET" || (req.Body != nil && req.Body != http.NoBody) || req.Header.Get("Authorization") != "" || req.Header.Get("Cookie") != "" {
					t.Fatal("country request has payload/credentials")
				}
				if count == 1 {
					if req.URL.String() != CountryTraceURL {
						t.Fatal(req.URL)
					}
					return countryReply(req, tc.primaryStatus, tc.primary), nil
				}
				if count != 2 || req.URL.String() != CountryFallbackURL {
					t.Fatal("more than fixed fallback", req.URL)
				}
				return countryReply(req, 200, tc.fallback), nil
			})
			got := c.DetectCountry(context.Background())
			if got.Country != tc.country || got.Source != tc.source || got.Error != "" || count != tc.calls {
				t.Fatal(got, count)
			}
			b, _ := json.Marshal(got)
			if strings.Contains(string(b), "203.0.113.99") || strings.Contains(string(b), "12345") {
				t.Fatal("GeoIP details leaked")
			}
		})
	}
}
func TestDetectCountryStrictFailuresThroughHTTP(t *testing.T) {
	badTrace := []string{"", "<html>loc=JP</html>", "h=www.cloudflare.com\nloc=XX", "h=www.cloudflare.com\nloc=WW", "h=www.cloudflare.com\nloc=ZZ", "h=www.cloudflare.com\nloc=jp", "h=www.cloudflare.com\nloc=JP\nloc=US", "h=www.cloudflare.com\nloc=JP\ncountry_code=DE", "h=other.invalid\nloc=JP", "h=www.cloudflare.com\nloc=JP\nerror=SECRET_BODY", "h=www.cloudflare.com\nloc=JP\n" + strings.Repeat("x", maxCountryBody)}
	badJSON := []string{`{}`, `{"country_code":"XX"}`, `{"country_code":"WW"}`, `{"country_code":123}`, `{"country_code":"JP","country_code":"US"}`, `{"country_code":"JP","country":"DE"}`, `{"country_code":"JP","countryCode":"JP"}`, `{"country_code":"JP","error":"SECRET_BODY"}`, `{"country_code":"JP","success":false}`, `{"country_code":"JP","status":"fail"}`, `{"country_code":"JP"} trailing`, `["JP"]`, `<html>country_code=JP</html>`, `{"country_code":"JP"`}
	for i := 0; i < len(badTrace)+len(badJSON); i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			c := NewClient(4, time.Second)
			defer c.Close()
			calls := 0
			c.HTTP.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				body := "invalid trace"
				if i < len(badTrace) {
					if calls == 1 {
						body = badTrace[i]
					} else {
						body = `{"country_code":"XX"}`
					}
				} else if calls == 2 {
					body = badJSON[i-len(badTrace)]
				}
				return countryReply(r, 200, body), nil
			})
			got := c.DetectCountry(context.Background())
			if got.Country != "" || got.Source != "" || got.Error == "" || calls != 2 {
				t.Fatal(got, calls)
			}
			if strings.Contains(got.Error, "SECRET_BODY") {
				t.Fatal("untrusted response leaked")
			}
		})
	}
	for _, status := range []int{403, 429, 500} {
		c := NewClient(4, time.Second)
		c.HTTP.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
			return countryReply(r, status, `{"country_code":"JP"}`), nil
		})
		got := c.DetectCountry(context.Background())
		c.Close()
		if got.Country != "" || got.Error == "" {
			t.Fatal(status, got)
		}
	}
}
func TestDetectCountrySharedBudgetCancellationNoRoute(t *testing.T) {
	c := NewClient(4, 80*time.Millisecond)
	defer c.Close()
	var calls atomic.Int32
	c.HTTP.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		n := calls.Add(1)
		if n == 1 {
			select {
			case <-time.After(50 * time.Millisecond):
				return countryReply(r, 500, "failure"), nil
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
		}
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	start := time.Now()
	r := c.DetectCountry(context.Background())
	if r.Error != "detection_budget_exhausted" || calls.Load() != 2 || time.Since(start) > 140*time.Millisecond {
		t.Fatal(r, calls.Load(), time.Since(start))
	}
	calls.Store(0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r = c.DetectCountry(ctx)
	if calls.Load() != 0 || r.Country != "" {
		t.Fatal(r, calls.Load())
	}
	c.Family = 0
	r = c.DetectCountry(context.Background())
	if calls.Load() != 0 || r.Error != "no_extra_family_route" {
		t.Fatal(r)
	}
}
func TestDetectCountryStrictTLSDirectFamilyAndRedirect(t *testing.T) {
	var outside atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cdn-cgi/trace":
			if r.URL.RawQuery == "" {
				fmt.Fprint(w, "h=www.cloudflare.com\nloc=JP\n")
			} else {
				http.Redirect(w, r, "http://must-not-connect.invalid/", 302)
			}
		case "/geoip":
			fmt.Fprint(w, `{"country_code":"US"}`)
		default:
			outside.Add(1)
		}
	}))
	defer server.Close()
	c := NewClient(4, time.Second)
	defer c.Close()
	tr := c.HTTP.Transport.(*http.Transport)
	if tr.Proxy != nil || tr.TLSClientConfig != nil && tr.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("unsafe client")
	}
	u, _ := url.Parse(server.URL)
	originalDial := tr.DialContext
	tr.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return originalDial(ctx, network, u.Host)
	}
	if got := c.DetectCountry(context.Background()); got.Country != "" {
		t.Fatal("accepted untrusted server TLS")
	}
	tr.TLSClientConfig = server.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	tr.TLSClientConfig.ServerName = u.Hostname()
	got := c.DetectCountry(context.Background())
	if got.Country != "JP" {
		t.Fatal(got)
	}
	// A family6 client cannot silently use the same IPv4-only loopback route.
	c6 := NewClient(6, time.Second)
	defer c6.Close()
	tr6 := c6.HTTP.Transport.(*http.Transport)
	dial6 := tr6.DialContext
	tr6.DialContext = func(ctx context.Context, n, a string) (net.Conn, error) { return dial6(ctx, n, u.Host) }
	tr6.TLSClientConfig = tr.TLSClientConfig.Clone()
	if v := c6.DetectCountry(context.Background()); v.Country != "" {
		t.Fatal("silently changed family", v)
	}
	c.HTTP.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() == CountryTraceURL {
			out := countryReply(r, 302, "")
			out.Header.Set("Location", "https://must-not-connect.invalid/secret")
			return out, nil
		}
		if r.URL.String() != CountryFallbackURL {
			outside.Add(1)
			return nil, fmt.Errorf("unexpected endpoint")
		}
		return countryReply(r, 200, `{"country_code":"US"}`), nil
	})
	if v := c.DetectCountry(context.Background()); v.Country != "US" || outside.Load() != 0 {
		t.Fatal("country redirect followed", v, outside.Load())
	}
	c.HTTP.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		out := countryReply(r, 200, "h=www.cloudflare.com\nloc=JP")
		out.Header.Set("Cf-Mitigated", "challenge")
		return out, nil
	})
	if v := c.DetectCountry(context.Background()); v.Country != "" {
		t.Fatal("header challenge treated as country")
	}
}

func TestCountryOnlyCanonicalISOAndLegacyUKAlias(t *testing.T) {
	for input, want := range map[string]string{"JP": "JP", "US": "US", "DE": "DE", "GB": "GB", "UK": "GB", "IN": "IN", "CN": "CN", "XX": "", "WW": "", "ZZ": "", "EU": "", "JPN": "", "jp": "", "": ""} {
		if got := strictCountryCode(input); got != want {
			t.Errorf("%q -> %q want %q", input, got, want)
		}
	}
}

func TestCountrySelfRedirectCannotMultiplyAttempts(t *testing.T) {
	c := NewClient(4, time.Second)
	defer c.Close()
	var requests atomic.Int32
	c.HTTP.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		if r.URL.String() == CountryTraceURL {
			out := countryReply(r, 302, "")
			out.Header.Set("Location", CountryTraceURL)
			return out, nil
		}
		return countryReply(r, 200, `{"country_code":"JP"}`), nil
	})
	got := c.DetectCountry(context.Background())
	if got.Country != "JP" || got.Source != CountryFallbackURL || requests.Load() != 2 {
		t.Fatal("country retry/redirect loop", got, requests.Load())
	}
}
