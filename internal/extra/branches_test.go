// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestJSONParserTruncationAndWrongType(t *testing.T) {
	data, _ := os.ReadFile("testdata/single.json")
	var fs map[string]struct{ Positive, Negative mockReply }
	json.Unmarshal(data, &fs)
	for id := range jsonResponseRequired {
		f, ok := fs[id]
		if !ok {
			continue
		}
		t.Run(id, func(t *testing.T) {
			for _, body := range []string{strings.TrimSuffix(f.Positive.Body, "}"), "<html>upstream failure</html>", "{bad-json"} {
				s := f.Positive
				s.Body = body
				c, _ := scripted(t, []mockReply{s}, 0)
				if got := execute(context.Background(), c, id); got.status != "error" {
					t.Errorf("malformed -> %+v", got)
				}
			}
			if spec, ok := scalarTypes[id]; ok {
				var obj map[string]any
				if json.Unmarshal([]byte(f.Positive.Body), &obj) == nil {
					obj[spec[0]] = map[string]any{"wrong": "type"}
					b, _ := json.Marshal(obj)
					s := f.Positive
					s.Body = string(b)
					c, _ := scripted(t, []mockReply{s}, 0)
					if got := execute(context.Background(), c, id); got.status != "error" {
						t.Errorf("wrong field type -> %+v", got)
					}
				}
			}
		})
	}
}
func TestDAZNRegionAndBan(t *testing.T) {
	for _, tc := range []struct{ body, status, region string }{{`{"isAllowed":true,"GeolocatedCountry":"us"}`, "available", "US"}, {"Security policy has been breached", "blocked", ""}} {
		c, _ := scripted(t, []mockReply{{Status: 200, Body: tc.body}}, 0)
		got := execute(context.Background(), c, "MediaUnlockTest_Dazn")
		if got.status != tc.status || got.region != tc.region {
			t.Fatal(got)
		}
	}
}
func TestDisneyLimitedAndTVerNonReferenceBranch(t *testing.T) {
	data, _ := os.ReadFile("testdata/flows.json")
	var fs map[string]fixture
	json.Unmarshal(data, &fs)
	disney := append([]mockReply(nil), fs["MediaUnlockTest_DisneyPlus"].Positive...)
	disney[2].Body = `{"countryCode":"US","inSupportedLocation":false}`
	c, _ := scripted(t, disney, 0)
	if got := execute(context.Background(), c, "MediaUnlockTest_DisneyPlus"); got.status != "restricted" || got.region != "US" {
		t.Fatal(got)
	}
	tv := append([]mockReply(nil), fs["MediaUnlockTest_TVer"].Positive...)
	tv[2].Body = `{"accountID":"1234567890123","playerID":"abcdefghi","videoID":"MOCK_VIDEO"}`
	tv[4].Contains = []string{"MOCK_VIDEO", "MOCK_CONFIG", "BCpkMOCK_POLICY"}
	c, _ = scripted(t, tv, 0)
	if got := execute(context.Background(), c, "MediaUnlockTest_TVer"); got.status != "available" {
		t.Fatal(got)
	}
}
func TestNativeCatalogueHasNoExecutionOrForbiddenData(t *testing.T) {
	for _, name := range []string{"requests-source.json", "client-data.json"} {
		data, _ := os.ReadFile(name)
		for _, bad := range []string{"UpdateAccount", "restriction.check@gmail.com", "user_id=649635267711712178", "c.evidon.com", "animestore.docomo.ne.jp", "api.trueid", "AISPlay"} {
			if strings.Contains(string(data), bad) {
				t.Fatalf("forbidden identity/safety/scope data %q in %s", bad, name)
			}
		}
	}
	// Every non-exception ID must have a concrete native request and a request/parser fixture.
	single, _ := os.ReadFile("testdata/single.json")
	flows, _ := os.ReadFile("testdata/flows.json")
	var s, f map[string]json.RawMessage
	json.Unmarshal(single, &s)
	json.Unmarshal(flows, &f)
	n := 0
	for _, c := range Catalogue() {
		if c.Safety != "" {
			if len(requestCatalogue[c.ID]) != 0 {
				t.Errorf("safety ID has executable request: %s", c.ID)
			}
			continue
		}
		n++
		if s[c.ID] == nil && f[c.ID] == nil {
			t.Errorf("missing real endpoint/parser fixture: %s", c.ID)
		}
	}
	if n != 125 {
		t.Fatal(n)
	}
}

func TestSingleServiceResponseRegions(t *testing.T) {
	for _, tc := range []struct {
		name, id   string
		status     int
		body       string
		headers    map[string][]string
		wantStatus string
		wantRegion string
	}{
		{"MyTVSuper other country", "MyTVSuper", 200, `{"country_code":"SG"}`, nil, "blocked", "SG"},
		{"KKTV other country", "KKTV", 200, `{"country":"JP"}`, nil, "blocked", "JP"},
		{"CBC Gem other country", "CBCGem", 200, `{"country":"US"}`, nil, "blocked", "US"},
		{"SBS other country", "SBSonDemand", 200, `{"country_code":"NZ"}`, nil, "blocked", "NZ"},
		{"MyTVSuper pseudo country", "MyTVSuper", 200, `{"country_code":"ZZ"}`, nil, "blocked", ""},
		{"KKTV conflicting countries", "KKTV", 200, `{"country":"JP","data":{"country":"SG"}}`, nil, "blocked", ""},
		{"SBS country on Unknown", "SBSonDemand", 403, `{"country_code":"NZ"}`, nil, "error", ""},
		{"DAZN geolocated refusal", "Dazn", 403, `{"isAllowed":false,"GeolocatedCountry":"gb"}`, nil, "blocked", "GB"},
		{"DAZN ambiguous country keys", "Dazn", 403, `{"isAllowed":false,"GeolocatedCountry":"GB","Geolocated\u0043ountry":"US"}`, nil, "blocked", ""},
		{"DAZN policy ban", "Dazn", 403, `{"error":"Security policy has been breached","GeolocatedCountry":"GB"}`, nil, "blocked", ""},
		{"Fubo location refusal", "FuboTV", 403, `{"error":"NO_SERVICE_IN_COUNTRY","country_code2":"mx"}`, nil, "blocked", "MX"},
		{"Fubo merely refused", "FuboTV", 403, `{"network_allowed":false}`, nil, "blocked", ""},
		{"Fubo malformed location value", "FuboTV", 403, `{"network_allowed":false,"country_code2":["MX","CA"]}`, nil, "blocked", ""},
		{"Rakuten caller country", "RakutenTV", 403, `{"error":"VPN detected","iso3166_code":"GB"}`, nil, "blocked", "GB"},
		{"Rakuten error with target country", "RakutenTV", 403, `{"error":{"message":"is not available","target":{"iso3166_code":"GB"}}}`, nil, "blocked", ""},
		{"Rakuten forbidden HTML", "RakutenTV", 403, `<html>forbidden "iso3166_code":"GB"</html>`, nil, "blocked", ""},
		{"Prime Video page config", "PrimeVideo", 200, `<html><script>var cfg={"isServiceRestricted":true,"currentTerritory":"CH"};</script></html>`, nil, "blocked", "CH"},
		{"Prime Video mere error wording", "PrimeVideo", 200, `<html>isServiceRestricted "currentTerritory":"CH"</html>`, nil, "blocked", ""},
		{"Prime Video country outside config", "PrimeVideo", 200, `<html><script>var cfg={"isServiceRestricted":true};</script><div>"currentTerritory":"CH"</div></html>`, nil, "blocked", ""},
		{"HBO Max territory excludes caller", "HBOMax", 200, `{"url":"/us/en"}`, map[string][]string{"Set-Cookie": {"countryCode=GB; Path=/"}}, "blocked", "GB"},
		{"HBO Max conflicting cookies", "HBOMax", 200, `{"url":"/us/en"}`, map[string][]string{"Set-Cookie": {"countryCode=GB; Path=/", "countryCode=FR; Path=/"}}, "blocked", ""},
		{"HBO Max apparent WAF", "HBOMax", 200, `<html><title>Access Denied</title></html>`, map[string][]string{"Set-Cookie": {"countryCode=GB; Path=/"}}, "blocked", ""},
		{"Hotstar US discontinued", "HotStar", 200, `landing`, map[string][]string{"Set-Cookie": {"geo=US; Path=/"}}, "blocked", "US"},
		{"Hotstar generic denied", "HotStar", 403, `Access Denied`, map[string][]string{"Set-Cookie": {"geo=US; Path=/"}}, "blocked", ""},
		{"Crackle US", "Crackle", 200, `{}`, map[string][]string{"X-Crackle-Region": {"US"}}, "available", "US"},
		{"Crackle another country", "Crackle", 200, `{}`, map[string][]string{"X-Crackle-Region": {"GB"}}, "blocked", "GB"},
		{"Crackle noncountry", "Crackle", 200, `{}`, map[string][]string{"X-Crackle-Region": {"EMEA"}}, "blocked", ""},
		{"SkyShowtime where to stream", "SkyShowTime", 200, `where-can-i-stream`, map[string][]string{"Set-Cookie": {"activeTerritory=BE; Path=/"}}, "blocked", "BE"},
		{"SkyShowtime ordinary 403", "SkyShowTime", 403, `Forbidden`, map[string][]string{"Set-Cookie": {"activeTerritory=BE; Path=/"}}, "blocked", ""},
		{"Abema explicit caller country on anonymous IP", "AbemaTV", 403, `{"message":"anonymous_ip","isoCountryCode":"DE"}`, nil, "blocked", "DE"},
		{"Abema captured refusal without a country", "AbemaTV", 403, jpFile(t, "abema-anonymous.json"), nil, "blocked", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, n := scripted(t, []mockReply{{Status: tc.status, Body: tc.body, Headers: tc.headers}}, 0)
			got := execute(context.Background(), c, "MediaUnlockTest_"+tc.id)
			if got.status != tc.wantStatus || got.region != tc.wantRegion || *n != 1 {
				t.Fatalf("got %+v (%d requests), want %s|%s", got, *n, tc.wantStatus, tc.wantRegion)
			}
		})
	}
}

func TestFlowServiceResponseRegions(t *testing.T) {
	data, err := os.ReadFile("testdata/flows.json")
	if err != nil {
		t.Fatal(err)
	}
	var fs map[string]fixture
	if err := json.Unmarshal(data, &fs); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id           string
		fromPositive bool
		edit         func([]mockReply)
		status       string
		region       string
	}{
		{"MediaUnlockTest_BahamutAnime", false, nil, "blocked", "TW"},
		{"MediaUnlockTest_BahamutAnime", false, func(r []mockReply) { r[2].Body = `<html data-geo="TW"><body data-geo="JP">` }, "blocked", ""},
		{"MediaUnlockTest_DisneyPlus", true, func(r []mockReply) { r[3].Final = "https://www.disneyplus.com/unavailable" }, "blocked", "US"},
		{"MediaUnlockTest_DisneyPlus", false, nil, "blocked", ""},
		{"MediaUnlockTest_ESPNPlus", true, nil, "available", "US"},
		{"MediaUnlockTest_ESPNPlus", false, nil, "blocked", "GB"},
		{"MediaUnlockTest_AETV", true, nil, "available", "US"},
		{"MediaUnlockTest_AETV", false, nil, "blocked", "GB"},
		{"MediaUnlockTest_SonyLiv", false, nil, "blocked", "US"},
		{"GameTest_MathsSpot", false, nil, "blocked", "US"},
		{"MediaUnlockTest_DiscoveryPlus", false, nil, "blocked", ""},
		{"MediaUnlockTest_DiscoveryPlus", false, func(r []mockReply) { r[2].Body = `{"error":"not yet available","mainTerritoryCode":"BE"}` }, "blocked", "BE"},
		{"MediaUnlockTest_DiscoveryPlus", false, func(r []mockReply) {
			r[2].Body = `{"error":"not yet available","markets":[{"mainTerritoryCode":"BE"}]}`
		}, "blocked", ""},
		{"MediaUnlockTest_TLCGO", false, nil, "blocked", ""},
		{"MediaUnlockTest_TLCGO", false, func(r []mockReply) { r[1].Body = `{"error":"is not yet available","mainTerritoryCode":"CA"}` }, "blocked", "CA"},
		{"MediaUnlockTest_LineTVTW", true, nil, "available", ""},
		{"MediaUnlockTest_LineTVTW", false, nil, "blocked", ""},
		{"MediaUnlockTest_RakutenTVJP", false, nil, "blocked", ""},
	} {
		t.Run(tc.id+"/"+tc.status+"/"+tc.region, func(t *testing.T) {
			base := fs[tc.id].Negative
			if tc.fromPositive {
				base = fs[tc.id].Positive
			}
			steps := append([]mockReply(nil), base...)
			if tc.edit != nil {
				tc.edit(steps)
			}
			c, n := scripted(t, steps, 0)
			got := execute(context.Background(), c, tc.id)
			if got.status != tc.status || got.region != tc.region || *n != len(steps) {
				t.Fatalf("got %+v (%d requests), want %s|%s (%d requests)", got, *n, tc.status, tc.region, len(steps))
			}
		})
	}
}
