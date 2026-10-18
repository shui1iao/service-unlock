// Anonymous legacy signup-endpoint observation derived from RRC MediaUnlockTest_Spotify
// check.sh:3569-3602, commit b6d4a6f9a87fc6eae6d3e62d0092ececcec8e844.
// Modified 2026-10-18: incomplete anonymous form; evidence-bounded refusals only.
// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"serviceunlock/internal/netx"
	"strings"
)

// SpotifySignupRequest is an incomplete anonymous request to a legacy signup
// endpoint, not a validate=1 operation or a completed registration. No email,
// password, account token, Cookie or login authorization can be attached by this
// constructor. Missing account credentials
// deliberately prevent this request from completing an account creation.
func SpotifySignupRequest() netx.Request {
	form := url.Values{
		"birth_day": {"11"}, "birth_month": {"11"}, "birth_year": {"2000"},
		"collect_personal_info": {"undefined"}, "creation_flow": {""},
		"creation_point": {"https://www.spotify.com/hk-en/"},
		"displayname":    {"Region eligibility check"}, "gender": {"male"}, "iagree": {"1"},
		"key": {"a1e486e2729f46d6bb368d6b2bcda326"}, "platform": {"www"}, "referrer": {""},
		"send-email": {"0"}, "thirdpartyemail": {"0"},
	}
	return netx.Request{Method: "POST", URL: "https://spclient.wg.spotify.com/signup/public/v1/account", JSON: form.Encode(),
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded", "Accept-Language": "en"}}
}

// ClassifySpotifySignup cannot establish account-registration eligibility.
// A launch/config flag or undocumented status311 is not positive evidence.
// Only the captured status320 + exact proxy refusal establishes a narrowly
// scoped No for THIS anonymous legacy request, not for every signup route.
func ClassifySpotifySignup(body []byte) (value, detail string, err error) {
	// A local strict object decoder avoids last-key-wins ambiguity in status,
	// country/launch fields and nested errors. No other API uses this parser.
	object := func(raw []byte) (map[string]json.RawMessage, error) {
		dec := json.NewDecoder(bytes.NewReader(raw))
		token, e := dec.Token()
		if e != nil || token != json.Delim('{') {
			return nil, fmt.Errorf("invalid Spotify signup response object")
		}
		fields := map[string]json.RawMessage{}
		seen := map[string]bool{}
		for dec.More() {
			token, e = dec.Token()
			if e != nil {
				return nil, fmt.Errorf("malformed Spotify signup JSON")
			}
			key, ok := token.(string)
			if !ok {
				return nil, fmt.Errorf("invalid Spotify signup JSON key")
			}
			folded := strings.ToLower(key)
			if seen[folded] {
				return nil, fmt.Errorf("ambiguous duplicate Spotify signup field")
			}
			seen[folded] = true
			var value json.RawMessage
			if e = dec.Decode(&value); e != nil {
				return nil, fmt.Errorf("malformed Spotify signup field")
			}
			fields[key] = value
		}
		if _, e = dec.Token(); e != nil {
			return nil, fmt.Errorf("malformed Spotify signup object")
		}
		if _, e = dec.Token(); e != io.EOF {
			return nil, fmt.Errorf("trailing Spotify signup data")
		}
		return fields, nil
	}
	fields, e := object(body)
	if e != nil {
		return "", "", e
	}
	var status *int
	if e = json.Unmarshal(fields["status"], &status); e != nil || status == nil {
		return "", "", fmt.Errorf("Spotify signup status missing or invalid; registration not tested")
	}
	// Optional metadata must be well-typed, but never supplies a yes/no verdict.
	countryEvidence := ""
	if raw, ok := fields["country"]; ok {
		var c string
		if json.Unmarshal(raw, &c) != nil || country(c) == "" {
			return "", "", fmt.Errorf("invalid Spotify signup country metadata")
		}
		countryEvidence = "; country=" + country(c)
	}
	if raw, ok := fields["is_country_launched"]; ok {
		var launched *bool
		if json.Unmarshal(raw, &launched) != nil || launched == nil {
			return "", "", fmt.Errorf("invalid Spotify signup launched metadata")
		}
	}
	errors := map[string]json.RawMessage{}
	if raw, ok := fields["errors"]; ok {
		errors, e = object(raw)
		if e != nil {
			return "", "", fmt.Errorf("invalid Spotify signup error object")
		}
	}
	var refusal string
	if raw, ok := errors["generic_error"]; ok {
		if json.Unmarshal(raw, &refusal) != nil {
			return "", "", fmt.Errorf("invalid Spotify signup generic error")
		}
	}
	const observedProxyRefusal = "You seem to be using a proxy service. Please turn off these services and try again. For more help, please contact customer service."
	if *status == 320 && strings.TrimSpace(refusal) == observedProxyRefusal {
		return "no", "legacy anonymous signup endpoint explicitly rejected this request as a proxy (status320); actual registration/login not tested", nil
	}
	// Do not reflect raw error messages: they may contain account/identity data.
	return "", "", fmt.Errorf("Spotify anonymous response status%d%s does not establish registration eligibility; actual registration/login not tested", *status, countryEvidence)
}
