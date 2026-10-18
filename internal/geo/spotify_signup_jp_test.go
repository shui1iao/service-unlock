// JP captured-response regressions. No real network/account is used.
package geo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestJPSpotifySignupRegistrationNotConfiguration(t *testing.T) {
	data, err := os.ReadFile("testdata/spotify-signup-jp.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]json.RawMessage
	if err = json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, body, want string
		unknown          bool
	}{
		{"captured_POST_proxy_denied", string(fixtures["denied"]), "no", false},
		{"captured_GET_configuration_not_registration", string(fixtures["configuration"]), "", true},
		{"synthetic_311_complete", string(fixtures["synthetic_success"]), "yes", false},
		{"status120", `{"status":120}`, "no", false},
		{"not_launched", `{"is_country_launched":false}`, "no", false},
		{"311_missing_country", `{"status":311,"is_country_launched":true}`, "", true},
		{"311_missing_launched", `{"status":311,"country":"JP"}`, "", true},
		{"311_country_numeric", `{"status":311,"country":123,"is_country_launched":true}`, "", true},
		{"311_launched_string", `{"status":311,"country":"JP","is_country_launched":"true"}`, "", true},
		{"311_status_string", `{"status":"311","country":"JP","is_country_launched":true}`, "", true},
		{"311_invalid_country", `{"status":311,"country":"ZZ","is_country_launched":true}`, "", true},
		{"malformed", `{"status":311`, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r := single(t, check(t, "spotify_signup"), func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "POST" || req.URL.String() != "https://spclient.wg.spotify.com/signup/public/v1/account" {
					t.Errorf("not anonymous registration validation POST: %s %s", req.Method, req.URL)
				}
				var body []byte
				if req.Body != nil {
					body, _ = io.ReadAll(req.Body)
				}
				form, e := url.ParseQuery(string(body))
				if e != nil {
					t.Fatal(e)
				}
				for _, k := range []string{"email", "password", "username", "authorization", "identifier_token"} {
					if _, ok := form[k]; ok {
						t.Errorf("identity/account field supplied: %s", k)
					}
				}
				if req.Header.Get("Authorization") != "" || req.Header.Get("Cookie") != "" {
					t.Error("account credential attached")
				}
				if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || form.Get("key") == "" || form.Get("send-email") != "0" {
					t.Error("validation form/side-effect suppression missing")
				}
				return response(req, 200, tc.body, nil), nil
			})
			if calls != 1 || r.Value != tc.want || (r.Err != nil) != tc.unknown {
				t.Fatalf("calls=%d result=%+v want=%s unknown=%v", calls, r, tc.want, tc.unknown)
			}
			if tc.name == "synthetic_311_complete" && !strings.Contains(r.Detail, "JP") {
				t.Error("actual country evidence lost")
			}
		})
	}
}
