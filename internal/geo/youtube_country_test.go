package geo

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"serviceunlock/internal/netx"
)

func TestYouTubeCountryTrustLiteralConflictAndUnknown(t *testing.T) {
	for _, tc := range []struct {
		name, body, region string
		problem            bool
	}{
		{"country-cn", `"countryCode":"CN"`, "CN", false},
		{"country-jp", `"countryCode":"JP"`, "JP", false},
		{"literal-cn", `ytcfg.set({}); www.google.cn`, "CN", false},
		{"literal-and-cn", `ytInitialData {}; www.google.cn "countryCode":"CN"`, "CN", false},
		{"literal-conflict", `ytcfg.set({}); www.google.cn "countryCode":"JP"`, "", true},
		{"country-conflict", `"countryCode":"CN" "countryCode":"US"`, "", true},
		{"other-field-conflict", `"countryCode":"CN" "INNERTUBE_CONTEXT_GL":"JP"`, "", true},
		{"content-region-conflict", `ytInitialData {}; www.google.cn "contentRegion":"US"`, "", true},
		{"other-fields-conflict", `"contentRegion":"US" "INNERTUBE_CONTEXT_GL":"CN"`, "", true},
		{"context-us", `ytcfg.set({"INNERTUBE_CONTEXT_GL":"US","contentRegion":"US"}); {"contentRegion":"US"}`, "US", false},
		{"context-cn", `ytcfg.set({"INNERTUBE_CONTEXT_GL":"CN","contentRegion":"CN"});`, "CN", false},
		{"context-single", `ytcfg.set({"INNERTUBE_CONTEXT_GL":"CN"});`, "", false},
		{"context-no-document", `{"INNERTUBE_CONTEXT_GL":"CN","contentRegion":"CN"}`, "", false},
		{"context-malformed", `ytcfg.set({"INNERTUBE_CONTEXT_GL":"CN","contentRegion":42});`, "", true},
		{"context-invalid-code", `ytcfg.set({"INNERTUBE_CONTEXT_GL":"ZZ","contentRegion":"ZZ"});`, "", true},
		{"context-conflicting-repetition", `ytcfg.set({"INNERTUBE_CONTEXT_GL":"CN","contentRegion":"CN"}); {"contentRegion":"US"}`, "", true},
		{"context-challenge", `g-recaptcha ytcfg.set({"INNERTUBE_CONTEXT_GL":"CN","contentRegion":"CN"});`, "", true},
		{"literal-not-regexp", `ytcfg.set({}); wwwXgoogleYcn`, "", false},
		{"bare-link", `<a href="https://www.google.cn">link</a>`, "", true},
		{"challenge", `ytcfg.set({}); g-recaptcha www.google.cn "countryCode":"CN"`, "", true},
		{"consent", `ytcfg.set({}); <title>Before you continue</title> www.google.cn "countryCode":"CN"`, "", true},
		{"error", `ytcfg.set({}); <title>Error</title> Something went wrong www.google.cn "countryCode":"CN"`, "", true},
		{"unknown", `"countryCode":"ZZ"`, "", true},
		{"wrong-type", `"countryCode":42`, "", true},
		{"missing", `ad-free`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &netx.Response{Status: 200, Body: []byte(tc.body), FinalURL: "https://www.youtube.com/premium", Header: http.Header{}}
			got := YouTubeCountry(r, "https://www.youtube.com/premium")
			if got.Region != tc.region || (got.Problem != "") != tc.problem || tc.region != "" && got.Source == "" {
				t.Fatal(got)
			}
		})
	}
	for _, final := range []string{"https://consent.youtube.com/", "https://www.youtube.com/sorry/", "http://www.youtube.com/premium", "https://evil.invalid/"} {
		got := YouTubeCountry(&netx.Response{Status: 200, Body: []byte(`ytcfg.set({}); www.google.cn "countryCode":"CN"`), FinalURL: final}, "https://www.youtube.com/premium")
		if got.Region != "" || got.Problem == "" {
			t.Fatal(final, got)
		}
	}
}

func TestGoogleYouTubeMergePreservesSearchExceptTrustedCN(t *testing.T) {
	sentinel := errors.New("Search timeout diagnostic")
	for _, country := range []string{"CN", "JP", ""} {
		for _, raw := range []string{"yes", "no", ""} {
			err := error(nil)
			if raw == "" {
				err = sentinel
			}
			at := time.Unix(100, 0)
			results := []Result{
				{Check: Check{ID: "google_captcha", Kind: KindBlocked}, Value: raw, Err: err, Detail: "Search observation", Region: "US", CheckedAt: at, RTT: time.Millisecond},
				{Check: Check{ID: "spotify_signup"}, Value: "no", Err: sentinel, Detail: "other", Region: "DE"},
			}
			ApplyYouTubeCountry(results, YouTubeCountryEvidence{Region: country, Source: "https://www.youtube.com/premium countryCode"})
			got := results[0]
			if got.Region != country || got.CheckedAt != at || got.RTT != time.Millisecond {
				t.Fatal(got)
			}
			if country == "CN" {
				if got.Value != "yes" || got.Err != nil || !strings.Contains(got.Detail, "YouTube CN evidence") || err != nil && !strings.Contains(got.Detail, sentinel.Error()) {
					t.Fatal(got)
				}
			} else if got.Value != raw || got.Err != err {
				t.Fatal("non-CN changed Search", got)
			}
			if results[1].Region != "DE" || results[1].Err != sentinel || results[1].Detail != "other" {
				t.Fatal("other service changed", results[1])
			}
		}
	}
	r := []Result{{Check: Check{ID: "google_captcha"}, Value: "no"}}
	ApplyYouTubeCountry(r, YouTubeCountryEvidence{Region: "CN", Source: "youtube", Problem: "conflicting fields"})
	if r[0].Region != "" || r[0].Value != "no" || !strings.Contains(r[0].Detail, "conflicting fields") {
		t.Fatal(r)
	}
}
