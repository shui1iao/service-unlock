// SPDX-License-Identifier: AGPL-3.0-only
// Captured JP response-shape regressions; no external platform is contacted.
package extra

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func jpFile(t *testing.T, name string) string {
	t.Helper()
	b, e := os.ReadFile("testdata/jp/" + name)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
func wowowJP(t *testing.T) []mockReply {
	lineup := jpFile(t, "wowow-lineup.json")
	var links []struct {
		Link string `json:"link"`
	}
	if e := json.Unmarshal([]byte(lineup), &links); e != nil {
		t.Fatal(e)
	}
	return []mockReply{
		{Status: 200, Body: lineup, Contains: []string{"/drama/original/json/lineup.json"}},
		{Status: 200, Body: "<!DOCTYPE html><html lang=\"ja-jp\"><body>MR programme</body></html>", Contains: []string{links[0].Link}},
		{Status: 200, Body: "<!DOCTYPE html><html lang=\"ja-jp\"><body>My Sweet Home programme</body></html>", Contains: []string{links[1].Link}},
		{Status: 200, Body: jpFile(t, "wowow-programme.html"), Contains: []string{links[2].Link}},
		{Status: 200, Body: jpFile(t, "wowow-content.html"), Contains: []string{"https://wod.wowow.co.jp/content/207531"}},
		{Status: 403, Body: jpFile(t, "wowow-denied.json"), Contains: []string{"/playback/auth", "176237", "meta_id"}},
	}
}
func TestJPWOWOWRealHTMLThenCaptured2055(t *testing.T) {
	steps := wowowJP(t)
	c, calls := scripted(t, steps, 0)
	got := execute(context.Background(), c, "MediaUnlockTest_wowow")
	if got.status != "blocked" || *calls != len(steps) {
		t.Fatalf("actual HTML chain must reach 403/2055: %+v calls=%d", got, *calls)
	}
}
func TestJPWOWOWStageFormatsNotDisabled(t *testing.T) {
	base := wowowJP(t)
	for _, tc := range []struct {
		index int
		body  string
	}{{0, "{broken"}, {1, `{"link":"https://wod.wowow.co.jp/content/207531"}`}, {4, `{"watch":"https://wod.wowow.co.jp/watch/176237"}`}, {5, `{"error":`}} {
		steps := append([]mockReply(nil), base...)
		steps[tc.index].Body = tc.body
		c, calls := scripted(t, steps, 0)
		got := execute(context.Background(), c, "MediaUnlockTest_wowow")
		if got.status != "error" || *calls != tc.index+1 {
			t.Fatalf("stage%d malformed must stop: %+v calls=%d", tc.index, got, *calls)
		}
	}
	for _, index := range []int{0, 1, 4, 5} {
		for _, mode := range []string{"challenge", "500", "429"} {
			steps := append([]mockReply(nil), base...)
			if mode == "challenge" {
				steps[index].Body = "<html>Just a moment cf_chl_opt</html>"
			} else if mode == "500" {
				steps[index].Status = 500
			} else {
				steps[index].Status = 429
			}
			c, _ := scripted(t, steps, 0)
			if got := execute(context.Background(), c, "MediaUnlockTest_wowow"); got.status != "error" {
				t.Fatal(index, mode, got)
			}
		}
	}
	steps := append([]mockReply(nil), base...)
	steps[4].Body = "<!DOCTYPE html><html><body>missing canonical/watch link</body></html>"
	c, _ := scripted(t, steps, 0)
	if got := execute(context.Background(), c, "MediaUnlockTest_wowow"); got.status != "error" {
		t.Fatal(got)
	}
}
func TestJPAbemaAnonymousIPIsExplicitRefusal(t *testing.T) {
	for _, tc := range []struct {
		status     int
		body, want string
	}{
		{403, jpFile(t, "abema-anonymous.json"), "blocked"},
		{200, `{"message":"anonymous_ip"}`, "blocked"},
		{200, `{}`, "error"}, {403, `{}`, "error"}, {403, `{"message":"unexpected"}`, "error"},
		{200, `{"isoCountryCode":"JP"}`, "available"}, {200, `{"isoCountryCode":"US"}`, "restricted"},
		{200, `{"isoCountryCode":123}`, "error"}, {403, `{"message":"anonymous_ip"`, "error"},
		{500, `{"message":"anonymous_ip"}`, "error"}, {429, `{"message":"anonymous_ip"}`, "error"},
		{403, `<html>cf_chl_opt anonymous_ip</html>`, "error"},
	} {
		c, _ := scripted(t, []mockReply{{Status: tc.status, Body: tc.body}}, 0)
		got := execute(context.Background(), c, "MediaUnlockTest_AbemaTV")
		if got.status != tc.want {
			t.Errorf("%d %s -> %+v want %s", tc.status, tc.body, got, tc.want)
		}
	}
}
func TestJPAbemaNoArbitraryTextRefusal(t *testing.T) {
	c, _ := scripted(t, []mockReply{{Status: 403, Body: "anonymous_ip"}}, 0)
	if got := execute(context.Background(), c, "MediaUnlockTest_AbemaTV"); got.status != "error" || strings.Contains(got.detail, "criterion matched") {
		t.Fatal(got)
	}
}

// This emits classification replays, not new JP measurements.
func TestJPWriteOfflineClassificationReplay(t *testing.T) {
	dir := os.Getenv("JP_FIX_ARTIFACT_DIR")
	if dir == "" {
		return
	}
	rows := []map[string]string{}
	c, _ := scripted(t, wowowJP(t), 0)
	wowow := execute(context.Background(), c, "MediaUnlockTest_wowow")
	rows = append(rows, map[string]string{"case": "WOWOW captured HTML + 403/2055", "before": "Unknown: malformed JSON step1", "after": wowow.status, "detail": wowow.detail})
	c, _ = scripted(t, []mockReply{{Status: 403, Body: jpFile(t, "abema-anonymous.json")}}, 0)
	abema := execute(context.Background(), c, "MediaUnlockTest_AbemaTV")
	rows = append(rows, map[string]string{"case": "Abema captured anonymous_ip", "before": "Unknown: missing country", "after": abema.status, "detail": abema.detail})
	data, e := os.ReadFile("../geo/testdata/spotify-signup-jp.json")
	if e != nil {
		t.Fatal(e)
	}
	var f map[string]json.RawMessage
	json.Unmarshal(data, &f)
	for _, key := range []string{"configuration", "denied", "synthetic_success"} {
		value, detail, e := ClassifySpotifySignup(f[key])
		if e != nil {
			detail = e.Error()
		}
		if value == "" {
			value = "Unknown"
		}
		rows = append(rows, map[string]string{"case": "Spotify " + key, "before": map[string]string{"configuration": "Yes (GET configuration)", "denied": "No when denial body supplied; old live GET used different configuration response", "synthetic_success": "Yes without strict country/launched checks"}[key], "after": value, "detail": detail})
	}
	os.MkdirAll(dir, 0700)
	out, e := json.MarshalIndent(map[string]any{"offline_replay_only": true, "real_platform_retested": false, "rows": rows}, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, "offline-classification-before-after.json"), out, 0600); e != nil {
		t.Fatal(e)
	}
}

func TestJPWOWOWPlaybackRequiresTypedPositiveOrExplicitError(t *testing.T) {
	for _, body := range []string{`{}`, `{"playback_session_id":null}`, `{"playback_session_id":""}`, `{"playback_session_id":123}`, `{"description":"VPN playback_session_id"}`, `{"error":{"code":999,"message":"other error"}}`} {
		steps := wowowJP(t)
		steps[len(steps)-1].Status = 200
		steps[len(steps)-1].Body = body
		c, n := scripted(t, steps, 0)
		v := execute(context.Background(), c, "MediaUnlockTest_wowow")
		if v.status != "error" || *n != len(steps) {
			t.Fatal(body, v, *n)
		}
	}
	steps := wowowJP(t)
	steps[len(steps)-1].Status = 200
	steps[len(steps)-1].Body = `{"playback_session_id":"SYNTHETIC_SESSION_NOT_LIVE"}`
	c, n := scripted(t, steps, 0)
	if v := execute(context.Background(), c, "MediaUnlockTest_wowow"); v.status != "available" || *n != len(steps) {
		t.Fatal(v, *n)
	}
}
