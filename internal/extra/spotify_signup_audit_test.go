// SPDX-License-Identifier: AGPL-3.0-only
// Counterexamples to old heuristics, NOT evidence of a successful registration.
package extra

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestSpotifySignupAuditNoUnsupportedYesOrNo(t *testing.T) {
	data, e := os.ReadFile("../geo/testdata/spotify-signup-jp.json")
	if e != nil {
		t.Fatal(e)
	}
	var f map[string]json.RawMessage
	if e = json.Unmarshal(data, &f); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct{ name, body, want string }{
		{"captured_proxy_denial", string(f["denied"]), "no"},
		{"captured_configuration", string(f["configuration"]), ""},
		{"synthetic_311_country_not_registration", `{"status":311,"country":"JP","is_country_launched":true}`, ""},
		{"synthetic_311_field_errors_not_success", `{"status":311,"country":"JP","is_country_launched":true,"errors":{"email":"Required","password":"Required"}}`, ""},
		{"bare_120", `{"status":120}`, ""},
		{"120_non_geographic_error", `{"status":120,"errors":{"generic_error":"Invalid form details"}}`, ""},
		{"120_no_evidenced_contract", `{"status":120,"errors":{"generic_error":"You seem to be using a proxy service. Please turn off these services and try again. For more help, please contact customer service."}}`, ""},
		{"bare_320", `{"status":320}`, ""},
		{"320_other_error", `{"status":320,"errors":{"generic_error":"Invalid form details"}}`, ""},
		{"launched_false_only", `{"is_country_launched":false}`, ""},
		{"config_not_launched", `{"status":1,"country":"CN","is_country_launched":false}`, ""},
		{"missing_country", `{"status":311,"is_country_launched":true}`, ""},
		{"wrong_country_type", `{"status":320,"country":123,"errors":{"generic_error":"You seem to be using a proxy service. Please turn off these services and try again. For more help, please contact customer service."}}`, ""},
		{"duplicate_status_conflict", `{"status":311,"status":320,"errors":{"generic_error":"You seem to be using a proxy service. Please turn off these services and try again. For more help, please contact customer service."}}`, ""},
		{"duplicate_generic_error", `{"status":320,"errors":{"generic_error":"other","generic_error":"You seem to be using a proxy service. Please turn off these services and try again. For more help, please contact customer service."}}`, ""},
		{"case_conflict_status", `{"status":320,"Status":311,"errors":{"generic_error":"You seem to be using a proxy service. Please turn off these services and try again. For more help, please contact customer service."}}`, ""},
		{"malformed", `{"status":320`, ""}, {"HTML", `<html>status320</html>`, ""}, {"null", `null`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, detail, e := ClassifySpotifySignup([]byte(tc.body))
			if value != tc.want || (e != nil) != (tc.want == "") {
				t.Fatalf("got value=%q detail=%q err=%v want=%q", value, detail, e, tc.want)
			}
			if value == "yes" {
				t.Fatal("anonymous request cannot establish registration")
			}
			if tc.want == "no" && !strings.Contains(detail, "actual registration/login not tested") {
				t.Fatal("negative overclaims signup outcome", detail)
			}
		})
	}
}
func TestSpotifySignupAuditStatusesNeverCreatePositiveEvidence(t *testing.T) {
	for _, status := range []int{0, 1, 120, 200, 311, 320, 429, 500} {
		for _, launched := range []bool{false, true} {
			body, _ := json.Marshal(map[string]any{"status": status, "country": "JP", "is_country_launched": launched})
			if value, _, _ := ClassifySpotifySignup(body); value != "" {
				t.Fatalf("status/launch flags alone cannot decide actual signup: %s -> %s", body, value)
			}
		}
	}
}

func TestSpotifySignupAuditTypesAndErrorPrivacy(t *testing.T) {
	for _, body := range []string{
		`{"status":"320"}`, `{"status":true}`, `{"status":null}`, `{"status":320,"country":"WW"}`, `{"status":320,"country":null}`, `{"status":320,"is_country_launched":null}`, `{"status":320,"is_country_launched":"false"}`, `{"status":320,"errors":[]}`, `{"status":320,"errors":null}`, `{"status":320,"errors":{"generic_error":123}}`,
		`{"status":320,"country":"JP","country":"US"}`, `{"status":320,"is_country_launched":true,"is_country_launched":false}`,
		`{"status":311,"country":"JP","is_country_launched":true,"errors":{"email":"PRIVATE_IDENTITY_MARKER","generic_error":"PRIVATE_ACCOUNT_MESSAGE"}}`,
	} {
		value, detail, e := ClassifySpotifySignup([]byte(body))
		if value != "" || e == nil {
			t.Fatal(body, value, detail, e)
		}
		if strings.Contains(detail+e.Error(), "PRIVATE_") {
			t.Fatal("raw account-field errors leaked")
		}
	}
}
