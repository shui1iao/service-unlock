package geo

import (
	"net/http"
	"testing"
)

func TestRetainedBingChinaBranch(t *testing.T) {
	for _, tc := range []struct{ id, body, want string }{
		{"bing", `redirect to cn.bing.com`, "CN"},
	} {
		result := single(t, check(t, tc.id), func(req *http.Request) (*http.Response, error) { return response(req, 200, tc.body, nil), nil })
		if result.Err != nil || result.Value != tc.want {
			t.Fatalf("%s: %+v", tc.id, result)
		}
	}
}
