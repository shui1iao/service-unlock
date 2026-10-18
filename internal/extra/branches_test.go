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
	if n != 126 {
		t.Fatal(n)
	}
}
