// Anonymous signup-eligibility validation derived from RRC MediaUnlockTest_Spotify
// check.sh:3569-3602, commit b6d4a6f9a87fc6eae6d3e62d0092ececcec8e844.
// Modified 2026-10-18: native form, no account identity fields; strict typed verdict.
// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"encoding/json"
	"fmt"
	"net/url"
	"serviceunlock/internal/netx"
)

// SpotifySignupRequest is an incomplete anonymous signup validation, not a
// completed registration. No email, password, account token, Cookie or login
// authorization can be attached by this constructor. Missing account credentials
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

// ClassifySpotifySignup never promotes configuration status1 or the mere
// absence of a refusal into registration eligibility. Region stays in detail;
// the existing availability JSON schema remains unchanged.
func ClassifySpotifySignup(body []byte) (value, detail string, err error) {
	var response struct {
		Status   *int    `json:"status"`
		Country  *string `json:"country"`
		Launched *bool   `json:"is_country_launched"`
	}
	if e := json.Unmarshal(body, &response); e != nil {
		return "", "", fmt.Errorf("malformed Spotify signup JSON or field type: %w", e)
	}
	if response.Status != nil && (*response.Status == 320 || *response.Status == 120) {
		return "no", fmt.Sprintf("anonymous signup validation refused (status %d)", *response.Status), nil
	}
	if response.Launched != nil && !*response.Launched {
		return "no", "anonymous signup validation: country not launched", nil
	}
	if response.Status == nil || *response.Status != 311 {
		return "", "", fmt.Errorf("Spotify registration eligibility not established: expected status311 (configuration status1 is not sufficient)")
	}
	if response.Country == nil || country(*response.Country) == "" || response.Launched == nil || !*response.Launched {
		return "", "", fmt.Errorf("Spotify status311 lacks valid country/launched evidence")
	}
	return "yes", "anonymous signup validation status311; country=" + country(*response.Country) + "; launched=true; not a completed registration/login test", nil
}
