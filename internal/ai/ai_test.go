// Derived from remnawave/geocheck v0.3.0, commit 50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc.
// MIT License; Copyright (c) 2026 Remnawave.
package ai

import "testing"

func find(t *testing.T, id string) endpoint {
	t.Helper()
	for _, e := range catalog {
		if e.id == id {
			return e
		}
	}
	t.Fatalf("no endpoint %q in the catalogue", id)
	return endpoint{}
}

// TestAuthErrorProvesReachability is the idea the whole package rests on: an
// API that asks for credentials has already agreed to serve the caller's
// network, so 401 is a positive result rather than a failure.
func TestAuthErrorProvesReachability(t *testing.T) {
	e := find(t, "deepseek")
	got := e.classify(401, `{"error":{"message":"Missing bearer authentication in header","type":"invalid_request_error"}}`)
	if got.State != StateReachable {
		t.Errorf("state = %v, want reachable", got.State)
	}
}

// A shared classifier guard remains testable without registering a retired endpoint.
func TestRegionRefusalBeatsTheStatus(t *testing.T) {
	e := endpoint{regionMarkers: []string{"unsupported_country_region_territory", "request not allowed"}}
	for _, body := range []string{`{"error":"unsupported_country_region_territory"}`, `{"error":"Request not allowed"}`} {
		if got := e.classify(401, body); got.State != StateBlocked {
			t.Fatal(got)
		}
	}
}

// TestChallengeIsNotAnAnswer keeps an interstitial from being read as either
// verdict — the same discipline the website checks follow.
func TestChallengeIsNotAnAnswer(t *testing.T) {
	e := find(t, "deepseek")
	body := `<html><head><title>Just a moment...</title></head><body>` +
		`<script>window._cf_chl_opt={};</script></body></html>`
	got := e.classify(403, body)
	if got.State != StateError {
		t.Errorf("state = %v, want error", got.State)
	}
}

func TestRetiredEndpointsAreNotRegistered(t *testing.T) {
	want := []string{"deepseek", "qwen_intl", "moonshot", "zhipu"}
	if len(catalog) != len(want) {
		t.Fatal(len(catalog))
	}
	for i, e := range catalog {
		if e.id != want[i] {
			t.Fatal(i, e.id)
		}
	}
}

// TestUnexpectedStatusIsInconclusive stops a strange answer from being turned
// into a verdict.
func TestUnexpectedStatusIsInconclusive(t *testing.T) {
	e := find(t, "deepseek")
	for _, status := range []int{500, 502, 418} {
		if got := e.classify(status, "server exploded"); got.State != StateError {
			t.Errorf("status %d gave %v, want error", status, got.State)
		}
	}
}

func TestSummarize(t *testing.T) {
	got := Summarize([]Result{
		{State: StateReachable}, {State: StateReachable},
		{State: StateBlocked}, {State: StateError},
	})
	want := Summary{Total: 4, Reachable: 2, Blocked: 1, Failed: 1}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// TestEveryEndpointIsHTTPS keeps an unencrypted probe from leaking which
// services are being asked about.
func TestEveryEndpointIsHTTPS(t *testing.T) {
	for _, e := range catalog {
		if len(e.url) < 8 || e.url[:8] != "https://" {
			t.Errorf("%s is not https: %s", e.id, e.url)
		}
		if len(e.authStatus) == 0 {
			t.Errorf("%s declares no status that proves reachability", e.id)
		}
	}
}
