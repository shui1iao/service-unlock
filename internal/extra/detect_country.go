// Scheduling-only exit-country lookup; no probe or information-report row.
// Uses the existing family-pinned direct HTTPS client and request safeguards.
// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const CountryTraceURL = "https://www.cloudflare.com/cdn-cgi/trace"
const CountryFallbackURL = "https://api.ip.sb/geoip"
const maxCountryBody = 16 << 10

type CountryDetection struct{ Country, Source, Error string }

// DetectCountry performs at most two serial requests under ONE shared Timeout
// budget. Only validated country/source/error codes survive; IP, ASN, coordinates,
// raw body and underlying connection error strings are deliberately discarded.
func (c *Client) DetectCountry(ctx context.Context) CountryDetection {
	if c.Family != 4 && c.Family != 6 {
		return CountryDetection{Error: "no_extra_family_route"}
	}
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	failures := []string{}
	for _, source := range []struct {
		url, accept string
		parse       func(string) (string, error)
	}{
		{CountryTraceURL, "text/plain", parseCountryTrace},
		{CountryFallbackURL, "application/json", parseCountryJSON},
	} {
		if ctx.Err() != nil {
			return CountryDetection{Error: "detection_budget_exhausted"}
		}
		// Keep the same transport/family, but reject every country redirect
		// (including self-redirect loops): at most one attempt per source.
		// No shared client/transport is mutated.
		scoped := *c
		httpClient := *c.HTTP
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return fmt.Errorf("country endpoint redirect rejected")
		}
		scoped.HTTP = &httpClient
		r, err := scoped.do(ctx, request{Method: "GET", URL: source.url, Headers: map[string]string{"Accept": source.accept}})
		reason := ""
		switch {
		case ctx.Err() != nil:
			return CountryDetection{Error: "detection_budget_exhausted"}
		case err != nil:
			reason = "request_failed"
		case r.Status != 200:
			reason = fmt.Sprintf("http_%d", r.Status)
		case r.Final != source.url:
			reason = "unexpected_final_endpoint"
		case strings.EqualFold(strings.TrimSpace(r.Header.Get("Cf-Mitigated")), "challenge"):
			reason = "challenge"
		case len(r.Text) > maxCountryBody:
			reason = "response_too_large"
		default:
			code, e := source.parse(r.Text)
			if e != nil {
				reason = "invalid_country_response"
			} else {
				return CountryDetection{Country: code, Source: source.url}
			}
		}
		// Source names here are fixed, never values from an untrusted response.
		label := "cloudflare_trace"
		if source.url == CountryFallbackURL {
			label = "ip_sb_country"
		}
		failures = append(failures, label+":"+reason)
	}
	return CountryDetection{Error: strings.Join(failures, "; ")}
}
func strictCountryCode(s string) string {
	if len(s) != 2 || s[0] < 'A' || s[0] > 'Z' || s[1] < 'A' || s[1] > 'Z' {
		return ""
	}
	if s == "UK" {
		return "GB"
	} // established GUKO alias, not a country-name inference
	return country(s) // pinned ISO country/private-use checks.
}
func parseCountryTrace(body string) (string, error) {
	fields := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(body), "\n") {
		line = strings.TrimSuffix(line, "\r")
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" {
			return "", fmt.Errorf("invalid trace line")
		}
		if _, exists := fields[key]; exists {
			return "", fmt.Errorf("duplicate trace key")
		}
		fields[key] = value
	}
	if fields["h"] != "www.cloudflare.com" {
		return "", fmt.Errorf("unexpected trace host")
	}
	for _, k := range []string{"error", "errors", "country", "country_code", "countryCode"} {
		if _, ok := fields[k]; ok {
			return "", fmt.Errorf("ambiguous trace")
		}
	}
	code := strictCountryCode(fields["loc"])
	if code == "" {
		return "", fmt.Errorf("invalid loc")
	}
	return code, nil
}
func parseCountryJSON(body string) (string, error) {
	// Token decoding rejects duplicate object keys instead of accepting last-wins.
	dec := json.NewDecoder(strings.NewReader(body))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return "", fmt.Errorf("expected country object")
	}
	fields := map[string]json.RawMessage{}
	for dec.More() {
		t, e := dec.Token()
		if e != nil {
			return "", e
		}
		key, ok := t.(string)
		if !ok {
			return "", fmt.Errorf("invalid key")
		}
		if _, exists := fields[key]; exists {
			return "", fmt.Errorf("duplicate key")
		}
		var raw json.RawMessage
		if e = dec.Decode(&raw); e != nil {
			return "", e
		}
		fields[key] = raw
	}
	if _, err = dec.Token(); err != nil {
		return "", err
	}
	if _, err = dec.Token(); err != io.EOF {
		return "", fmt.Errorf("trailing country data")
	}
	for _, k := range []string{"error", "errors", "countryCode", "country_iso", "loc"} {
		if _, ok := fields[k]; ok {
			return "", fmt.Errorf("error or ambiguous country response")
		}
	}
	if raw, ok := fields["success"]; ok && string(raw) != "true" {
		return "", fmt.Errorf("API error")
	}
	if raw, ok := fields["status"]; ok {
		var status string
		if json.Unmarshal(raw, &status) != nil || status != "success" {
			return "", fmt.Errorf("API error status")
		}
	}
	var value string
	if json.Unmarshal(fields["country_code"], &value) != nil {
		return "", fmt.Errorf("invalid country_code")
	}
	code := strictCountryCode(value)
	if code == "" {
		return "", fmt.Errorf("unknown country_code")
	}
	// A name is not used to infer geography. Conflicting explicit alpha2 evidence
	// is rejected, while ordinary full country names are ignored.
	if raw, ok := fields["country"]; ok {
		var other string
		if json.Unmarshal(raw, &other) != nil {
			return "", fmt.Errorf("invalid country field")
		}
		if normalized := strictCountryCode(other); normalized != "" && normalized != code {
			return "", fmt.Errorf("conflicting country fields")
		}
	}
	return code, nil
}
