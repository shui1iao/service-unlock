package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"serviceunlock/internal/extra"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type autoTrip func(*http.Request) (*http.Response, error)

func (f autoTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func autoResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), ContentLength: -1, Request: req}
}
func autoRunFixture(t *testing.T, args []string, traceBody, fallbackBody string) (report, string, int, int) {
	t.Helper()
	var count atomic.Int32
	var out, stderr bytes.Buffer
	run := func(ctx context.Context, timeout time.Duration, workers int) (measurements, error) {
		opts := ctx.Value(extraOptionsKey{}).(extraOptions)
		explicit := false
		for i, a := range args {
			if a == "--region" && i+1 < len(args) && args[i+1] != "auto" {
				explicit = true
			}
		}
		if !explicit && opts.Region != "auto" {
			t.Fatal("CLI default did not select auto", opts)
		}
		stack, _ := mockCompleteStack(t)
		underlying := stack.HTTP(0).Transport
		stack.HTTP(0).Transport = autoTrip(func(r *http.Request) (*http.Response, error) {
			if !explicit && count.Load() == 0 {
				t.Error("native probes ran before automatic scheduling")
			}
			return underlying.RoundTrip(r)
		})
		client := extra.NewClient(4, timeout)
		defer client.Close()
		client.HTTP.Transport = autoTrip(func(r *http.Request) (*http.Response, error) {
			switch r.URL.String() {
			case extra.CountryTraceURL:
				count.Add(1)
				return autoResponse(r, 200, traceBody), nil
			case extra.CountryFallbackURL:
				count.Add(1)
				return autoResponse(r, 200, fallbackBody), nil
			}
			// Extra probes really execute, with deterministic local fake replies.
			// At least one verdict survives even when country detection fails.
			if strings.Contains(r.URL.Hostname(), "tvbanywhere") {
				return autoResponse(r, 200, `{"allow_in_this_country":true}`), nil
			}
			return nil, errors.New("offline service response unavailable")
		})
		return runWithClients(ctx, workers, stack, client, nil)
	}
	code := runCLI(context.Background(), args, &out, &stderr, run)
	if stderr.Len() != 0 {
		t.Fatal(stderr.String())
	}
	var r report
	terminal := ""
	if containsFlag(args, "--machine") {
		var e struct {
			Format   string `json:"format"`
			Report   report `json:"report"`
			Terminal string `json:"terminal"`
		}
		if er := json.Unmarshal(out.Bytes(), &e); er != nil {
			t.Fatal(er, out.String())
		}
		if e.Format != "guko-unified-v1" {
			t.Fatal(e.Format)
		}
		r, terminal = e.Report, e.Terminal
	} else if containsFlag(args, "--json") {
		if e := json.Unmarshal(out.Bytes(), &r); e != nil {
			t.Fatal(e)
		}
		if strings.Contains(out.String(), "\x1b") {
			t.Fatal("ANSI polluted JSON")
		}
	} else {
		return report{}, out.String(), code, int(count.Load())
	}
	if strings.Contains(out.String(), "203.0.113.55") || strings.Contains(out.String(), "PRIVATE_ASN_MARKER") {
		t.Fatal("GeoIP details leaked")
	}
	if r.Network.AddressFamily != "system-selected" || r.Source["port"] != "Go v0.4.0 standalone three-group extraction with conservative error handling" {
		t.Fatal("original metadata changed")
	}
	return r, terminal, code, int(count.Load())
}
func containsFlag(args []string, f string) bool {
	for _, a := range args {
		if a == f {
			return true
		}
	}
	return false
}
func TestDefaultAutoActualPipelineCountrySelectsRegionalChecks(t *testing.T) {
	for _, tc := range []struct {
		country, group string
		total          int
	}{{"JP", "jp", 67}, {"US", "na", 78}, {"ES", "eu", 71}, {"DE", "eu", 71}, {"SG", "sea", 52}, {"HK", "hk", 49}, {"MO", "hk", 49}, {"KR", "kr", 49}, {"TW", "tw", 52}, {"AU", "oc", 60}, {"ZA", "af", 44}, {"BR", "sa", 44}, {"GB", "eu", 71}} {
		t.Run(tc.country, func(t *testing.T) {
			// No --region flag: execute the real resolver + actual core/extra Run paths.
			r, terminal, code, n := autoRunFixture(t, []string{"--machine"}, "h=www.cloudflare.com\nloc="+tc.country+"\nip=203.0.113.55\n", `{"country_code":"ZZ"}`)
			if code != 1 || n != 1 || r.Selection.RequestedRegion != "auto" || r.Selection.Region != tc.group || r.Selection.DetectedCountry != tc.country || r.Selection.Family != 4 || r.Selection.DetectionSource != extra.CountryTraceURL || r.Selection.DetectionError != "" {
				t.Fatal(r.Selection, code, n)
			}
			if len(r.Results)+len(r.Services)+len(r.AIEndpoints)+len(r.ExtraChecks) != tc.total {
				t.Fatal("auto only changed label, not real selected scope", tc.total, len(r.ExtraChecks))
			}
			checks, _ := extra.Selected(tc.group)
			ids := []string{}
			for _, c := range checks {
				ids = append(ids, c.ID)
			}
			if !reflect.DeepEqual(ids, r.Selection.ExpectedIDs) {
				t.Fatal("wrong expected selection")
			}
			if !strings.HasPrefix(terminal, "Automatic region: "+tc.country+" -> ") || strings.Count(terminal, "Automatic region:") != 1 {
				t.Fatal(terminal)
			}
			// Native mocked content country is US. It must NOT override exit country JP/etc.
			if *r.Services[0].Value != "US" {
				t.Fatal("fixture changed")
			}
			if dir := os.Getenv("AUTO_REGION_ARTIFACT_DIR"); dir != "" {
				os.MkdirAll(dir, 0700)
				b, _ := json.MarshalIndent(r, "", "  ")
				os.WriteFile(filepath.Join(dir, "default-"+tc.country+".json"), b, 0600)
				os.WriteFile(filepath.Join(dir, "default-"+tc.country+".ansi.txt"), []byte("OFFLINE MOCK - NOT REAL MEASUREMENT\n"+terminal), 0600)
				envelope, _ := json.MarshalIndent(struct {
					Format   string `json:"format"`
					Report   report `json:"report"`
					Terminal string `json:"terminal"`
				}{"guko-unified-v1", r, terminal}, "", "  ")
				os.WriteFile(filepath.Join(dir, "default-"+tc.country+".machine.json"), envelope, 0600)
			}
		})
	}
}
func TestAutoFallbackUnsupportedAndFailureStillRunsGlobal(t *testing.T) {
	for _, tc := range []struct {
		name, primary, fallback, country, region, reason string
		calls                                            int
	}{{"fallback", "bad trace", `{"country_code":"DE","ip":"203.0.113.55","asn":"PRIVATE_ASN_MARKER"}`, "DE", "eu", "country_mapped", 2}, {"unsupportedCN", "h=www.cloudflare.com\nloc=CN", `{}`, "CN", "global", "unsupported_region_country", 1}, {"unsupportedIN", "h=www.cloudflare.com\nloc=IN", `{}`, "IN", "global", "unsupported_region_country", 1}, {"failed", "h=www.cloudflare.com\nloc=XX", `{"country_code":"WW"}`, "", "global", "detection_failed", 2}, {"APIerror", "h=www.cloudflare.com\nloc=JP\nerror=fail", `{"country_code":"JP","error":"denied"}`, "", "global", "detection_failed", 2}} {
		t.Run(tc.name, func(t *testing.T) {
			r, terminal, _, n := autoRunFixture(t, []string{"--machine"}, tc.primary, tc.fallback)
			if n != tc.calls || r.Selection.Region != tc.region || r.Selection.ResolutionReason != tc.reason || r.Selection.DetectedCountry != tc.country {
				t.Fatal(r.Selection, n)
			}
			if tc.region == "global" && len(r.ExtraChecks)+35 != 41 {
				t.Fatal("fallback did not select global")
			}
			found := false
			for _, row := range r.ExtraChecks {
				if row.ID == "MediaUnlockTest_TVBAnywhere" {
					found = row.Status == "available"
				}
			}
			if !found {
				t.Fatal("GeoIP failure poisoned service verdicts")
			}
			if tc.reason == "detection_failed" && r.Selection.DetectionError == "" {
				t.Fatal("silent failure")
			}
			if !strings.Contains(terminal, tc.reason) && tc.reason != "country_mapped" {
				t.Fatal(terminal)
			}
		})
	}
}
func TestExplicitRegionAndGlobalSkipCountryRequests(t *testing.T) {
	for _, region := range extra.Regions() {
		r, terminal, _, n := autoRunFixture(t, []string{"--machine", "--region", region}, "must-not-request", `{"error":"must-not-request"}`)
		if n != 0 || r.Selection.RequestedRegion != region || r.Selection.Region != region || r.Selection.ResolutionReason != "explicit_region_override" || r.Selection.DetectedCountry != "" || r.Selection.DetectionSource != "" || strings.Contains(terminal, "Automatic region:") {
			t.Fatal(region, r.Selection, n)
		}
	}
}
func TestAutoJSONAndHumanOutputHaveOneSameSelection(t *testing.T) {
	r, _, _, n := autoRunFixture(t, []string{"--json"}, "h=www.cloudflare.com\nloc=JP", `{}`)
	if n != 1 || r.Selection.Region != "jp" {
		t.Fatal(r.Selection, n)
	}
	_, plain, _, n := autoRunFixture(t, nil, "h=www.cloudflare.com\nloc=JP", `{}`)
	if n != 1 || strings.Contains(plain, "\x1b") || strings.Count(plain, "Automatic region:") != 1 || !strings.Contains(plain, "Regional services") {
		t.Fatal(plain, n)
	}
}
func TestAutoCancelNoPartialReportAndNoFamilyNoDetection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	stack, _ := mockCompleteStack(t)
	client := extra.NewClient(4, time.Second)
	defer client.Close()
	var count atomic.Int32
	client.HTTP.Transport = autoTrip(func(r *http.Request) (*http.Response, error) { count.Add(1); cancel(); return nil, ctx.Err() })
	var out, stderr bytes.Buffer
	code := runCLI(ctx, []string{"--machine"}, &out, &stderr, func(ctx context.Context, d time.Duration, w int) (measurements, error) {
		return runWithClients(ctx, w, stack, client, nil)
	})
	if code != 130 || out.Len() != 0 || count.Load() != 1 {
		t.Fatal(code, out.String(), count.Load())
	}
	ctx = context.WithValue(context.Background(), extraOptionsKey{}, extraOptions{"auto", "auto"})
	client.Family = 0
	before := count.Load()
	m, e := runWithClients(ctx, 3, stack, client, fmt.Errorf("no local route"))
	if e != nil || count.Load() != before || m.Selection.Region != "global" || m.Selection.DetectionError != "no_extra_family_route" || len(m.Access) != 6 || len(m.Extra) != 6 {
		t.Fatal(m.Selection, e, count.Load())
	}
}
func TestAutoCountryMappingMatchesEstablishedBoundaries(t *testing.T) {
	for c, g := range map[string]string{"US": "na", "CA": "na", "ES": "eu", "DE": "eu", "GB": "eu", "UK": "eu", "SG": "sea", "TW": "tw", "HK": "hk", "MO": "hk", "JP": "jp", "KR": "kr", "NA": "af"} {
		if countryGroups[c] != g {
			t.Fatal(c, countryGroups[c], g)
		}
	}
	for _, c := range []string{"CN", "IN", "XX", "WW"} {
		if countryGroups[c] != "" {
			t.Fatal("expanded unsupported country", c)
		}
	}
}

func TestAutoDetectionTimeoutDegradesWithoutPoisoningServiceRuns(t *testing.T) {
	stack, _ := mockCompleteStack(t)
	client := extra.NewClient(4, 15*time.Millisecond)
	defer client.Close()
	var countryCalls, serviceCalls atomic.Int32
	client.HTTP.Transport = autoTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() == extra.CountryTraceURL || r.URL.String() == extra.CountryFallbackURL {
			countryCalls.Add(1)
			<-r.Context().Done()
			return nil, r.Context().Err()
		}
		serviceCalls.Add(1)
		if strings.Contains(r.URL.Hostname(), "tvbanywhere") {
			return autoResponse(r, 200, `{"allow_in_this_country":true}`), nil
		}
		return nil, errors.New("offline service response unavailable")
	})
	ctx := context.WithValue(context.Background(), extraOptionsKey{}, extraOptions{"auto", "4"})
	m, e := runWithClients(ctx, 4, stack, client, nil)
	if e != nil || countryCalls.Load() != 1 || serviceCalls.Load() == 0 || m.Selection.Region != "global" || m.Selection.DetectionError != "detection_budget_exhausted" || len(m.Access) != 6 || len(m.Extra) != 6 {
		t.Fatal(m.Selection, e, countryCalls.Load(), serviceCalls.Load())
	}
	for _, row := range m.Extra {
		if row.ID == "MediaUnlockTest_TVBAnywhere" && row.Status != "available" {
			t.Fatal("detection timeout changed service verdict", row)
		}
	}
}
func TestDefaultAutoHelpAndLicenseNeverUseRunner(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--license"}} {
		var out, stderr bytes.Buffer
		code := runCLI(context.Background(), args, &out, &stderr, func(context.Context, time.Duration, int) (measurements, error) {
			t.Fatal("help/license performed detection/probes")
			return measurements{}, nil
		})
		if code != 0 || stderr.Len() != 0 {
			t.Fatal(code, stderr.String())
		}
		if args[0] == "--help" && !strings.Contains(out.String(), "default auto") {
			t.Fatal(out.String())
		}
	}
}

func TestEntireCountryMappingMatchesPinnedGUKOFixture(t *testing.T) {
	data, e := os.ReadFile("testdata/region-country-groups.json")
	if e != nil {
		t.Fatal(e)
	}
	var expected map[string]string
	if e = json.Unmarshal(data, &expected); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(expected, countryGroups) {
		t.Fatal("country scope drift from existing GUKO mapping", countryGroups, expected)
	}
}
