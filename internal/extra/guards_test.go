// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestEveryNativeEndpointEmptyChallengeAndMalformed(t *testing.T) {
	data, _ := os.ReadFile("testdata/single.json")
	var fixtures map[string]struct{ Positive, Negative mockReply }
	json.Unmarshal(data, &fixtures)
	for id := range fixtures {
		t.Run(id, func(t *testing.T) {
			for _, s := range []mockReply{{Status: 200, Body: ""}, {Status: 200, Body: "<html>Just a moment cf_chl_opt</html>"}, {Status: 403, Body: "cf-browser-verification"}} {
				c, _ := scripted(t, []mockReply{s}, 0)
				if got := execute(context.Background(), c, id); got.status != "error" {
					t.Fatalf("empty/challenge -> %+v", got)
				}
			}
		})
	}
}
func TestEveryFlowIntermediateFailure(t *testing.T) {
	data, _ := os.ReadFile("testdata/flows.json")
	var fixtures map[string]fixture
	json.Unmarshal(data, &fixtures)
	for id, f := range fixtures {
		for index := range f.Positive {
			for _, status := range []int{429, 500} {
				t.Run(fmt.Sprintf("%s/%d/%d", id, index, status), func(t *testing.T) {
					steps := append([]mockReply(nil), f.Positive...)
					steps[index].Status = status
					c, n := scripted(t, steps, 0)
					got := execute(context.Background(), c, id)
					if got.status != "error" {
						t.Fatalf("got %+v", got)
					}
					if *n > index+1 {
						t.Errorf("continued requests %d after failure at %d", *n, index)
					}
				})
			}
		}
	}
}
func TestClientTLSAndNoProxyRedirect(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "http://127.0.0.1/private", 302)
			return
		}
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()
	c := NewClient(4, time.Second)
	defer c.Close()
	if _, e := c.do(context.Background(), request{URL: server.URL}); e == nil {
		t.Fatal("accepted untrusted TLS")
	}
	tr := c.HTTP.Transport.(*http.Transport)
	if tr.Proxy != nil || tr.TLSClientConfig != nil && tr.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("weak network policy")
	}
	trusted := server.Client()
	trusted.CheckRedirect = c.HTTP.CheckRedirect
	safe := &Client{HTTP: trusted, Family: 4, Timeout: time.Second}
	if _, e := safe.do(context.Background(), request{URL: server.URL + "/redirect"}); e == nil {
		t.Fatal("HTTPS downgrade accepted")
	}
	if _, e := c.do(context.Background(), request{URL: "http://127.0.0.1"}); e == nil {
		t.Fatal("plain HTTP accepted")
	}
}
func TestConcurrencyCancellationAndFamily(t *testing.T) {
	var active, peak atomic.Int32
	c := &Client{Family: 4, Timeout: time.Second, HTTP: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		n := active.Add(1)
		defer active.Add(-1)
		for p := peak.Load(); n > p && !peak.CompareAndSwap(p, n); p = peak.Load() {
		}
		select {
		case <-r.Context().Done():
			return nil, r.Context().Err()
		case <-time.After(15 * time.Millisecond):
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"code":0}`)), Request: r}, nil
	})}}
	checks := []Check{}
	for i := 0; i < 12; i++ {
		checks = append(checks, Check{ID: "MediaUnlockTest_BilibiliTW", Name: "BiliBili", Groups: []string{"tw"}})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	r := Run(ctx, c, "tw", checks, 3)
	if len(r) != 12 || active.Load() != 0 || peak.Load() > 3 || peak.Load() < 2 {
		t.Fatal(len(r), active.Load(), peak.Load())
	}
	c.Family = 6
	checks = []Check{{ID: "MediaUnlockTest_BilibiliTW", IPv6Unsupported: true}}
	r = Run(context.Background(), c, "tw", checks, 1)
	if r[0].Status != "error" || !strings.Contains(r[0].Detail, "IPv6") {
		t.Fatal(r)
	}
	for _, mode := range []string{"4", "6"} {
		f, e := SelectFamily(mode)
		if e != nil || f != int(mode[0]-'0') {
			t.Fatal(f, e)
		}
	}
	if _, e := SelectFamily("46"); e == nil {
		t.Fatal("bad family accepted")
	}
}
func TestResponseSizeAndTimeout(t *testing.T) {
	c := &Client{Family: 4, Timeout: 5 * time.Millisecond, HTTP: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })}}
	if _, e := c.do(context.Background(), request{URL: "https://fixture.invalid/"}); e == nil {
		t.Fatal("timeout absent")
	}
	if _, e := bounded(strings.NewReader(strings.Repeat("x", maxBody+1))); e == nil {
		t.Fatal("body bound absent")
	}
}
