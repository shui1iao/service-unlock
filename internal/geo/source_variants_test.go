package geo

import (
	"net/http"
	"testing"
)

func TestOriginalGoogleLocaleAndBingChinaBranches(t *testing.T) {
	for _, tc := range []struct{ id, body, want string }{
		{"google", `"en-GB" "fr-FR"`, "FR"},
		{"google", `"en_US" "fr-FR"`, "US"},
		{"bing", `redirect to cn.bing.com`, "CN"},
	} {
		result := single(t, check(t, tc.id), func(req *http.Request) (*http.Response, error) { return response(req, 200, tc.body, nil), nil })
		if result.Err != nil || result.Value != tc.want {
			t.Fatalf("%s: %+v", tc.id, result)
		}
	}
}
