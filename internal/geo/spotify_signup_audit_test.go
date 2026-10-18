package geo

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestSpotifySignupAuditHTTPAndChallengeGuards(t *testing.T) {
	data, e := os.ReadFile("testdata/spotify-signup-jp.json")
	if e != nil {
		t.Fatal(e)
	}
	var f map[string]json.RawMessage
	json.Unmarshal(data, &f)
	denial := string(f["denied"])
	for _, tc := range []struct {
		name                             string
		status                           int
		body, ctype, challenge, redirect string
	}{
		{"bare403", 403, denial, "application/json", "", ""},
		{"429", 429, denial, "application/json", "", ""},
		{"500", 500, denial, "application/json", "", ""},
		{"non200", 201, denial, "application/json", "", ""},
		{"header_challenge_200", 200, denial, "application/json", "challenge", ""},
		{"header_challenge_empty", 200, "", "text/html", "challenge", ""},
		{"HTML", 200, "<html>Just a moment cf_chl_opt</html>", "text/html", "", ""},
		{"JSON_with_challenge_content", 200, strings.TrimSuffix(denial, "}") + `,"html":"cf_chl_opt"}`, "application/json", "", ""},
		{"wrong_media", 200, denial, "text/html", "", ""},
		{"malformed_media", 200, denial, ";;invalid", "", ""},
		{"redirected_denial", 302, denial, "application/json", "", "https://spclient.wg.spotify.com/unexpected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r := single(t, check(t, "spotify_signup"), func(req *http.Request) (*http.Response, error) {
				calls++
				h := http.Header{}
				h.Set("Content-Type", tc.ctype)
				h.Set("Cf-Mitigated", tc.challenge)
				status := tc.status
				if tc.redirect != "" && calls == 1 {
					h.Set("Location", tc.redirect)
				} else if calls > 1 {
					status = 200
				}
				return response(req, status, tc.body, h), nil
			})
			if r.Value != "" || r.Err == nil {
				t.Fatal("HTTP/WAF/nonofficial response became verdict", r)
			}
			if calls > 2 {
				t.Fatal("unexpected requests", calls)
			}
		})
	}
}
func TestSpotifyCountryObservationDoesNotVoteForSignup(t *testing.T) {
	calls := 0
	env := environment(t, func(req *http.Request) (*http.Response, error) {
		calls++
		switch req.URL.Host {
		case "accounts.spotify.com":
			return response(req, 200, `{"geoLocationCountryCode":"JP"}`, nil), nil
		case "spclient.wg.spotify.com":
			return response(req, 200, `{"status":311,"country":"JP","is_country_launched":true}`, nil), nil
		}
		t.Fatal(req.URL)
		return nil, nil
	})
	results := Run(context.Background(), env, []Check{check(t, "spotify"), check(t, "spotify_signup")}, 1)
	if calls != 2 || results[0].Value != "JP" || results[0].Err != nil || results[1].Value != "" || results[1].Err == nil {
		t.Fatal("country/launch observations promoted signup", results, calls)
	}
	if !strings.Contains(results[1].Err.Error(), "actual registration/login not tested") {
		t.Fatal(results[1])
	}
}
