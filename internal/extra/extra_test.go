// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

type mockReply struct {
	Status   int                 `json:"status"`
	Body     string              `json:"body"`
	Final    string              `json:"final"`
	Headers  map[string][]string `json:"headers"`
	Contains []string            `json:"request_contains"`
	Method   string              `json:"request_method"`
	BodyHash string              `json:"request_body_sha256"`
}
type fixture struct{ Positive, Negative []mockReply }
type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func scripted(t *testing.T, steps []mockReply, fault int) (*Client, *int) {
	t.Helper()
	calls := 0
	h := &http.Client{Transport: transportFunc(func(q *http.Request) (*http.Response, error) {
		index := calls
		calls++
		if index >= len(steps) {
			return nil, fmt.Errorf("unexpected request %s", q.URL)
		}
		s := steps[index]
		body, _ := io.ReadAll(q.Body)
		q.Body.Close()
		if s.Method != "" && q.Method != s.Method {
			t.Errorf("wrong request method: %s want %s", q.Method, s.Method)
		}
		if s.BodyHash != "" && fmt.Sprintf("%x", sha256.Sum256(body)) != s.BodyHash {
			t.Errorf("static request body changed at step %d", index)
		}
		text := q.URL.String() + "\n" + string(body) + "\n" + fmt.Sprint(q.Header)
		for _, need := range s.Contains {
			if !strings.Contains(text, need) {
				t.Errorf("step %d missing request token %q in %s", index, need, text)
			}
		}
		if q.URL.Scheme != "https" {
			t.Fatal("non-HTTPS request")
		}
		for _, bad := range []string{"${", "bahamut_cookie.txt", "restriction.check@gmail.com"} {
			if strings.Contains(text, bad) {
				t.Errorf("unresolved/identity parameter %q", bad)
			}
		}
		if fault == -1 {
			return nil, fmt.Errorf("fixture network error")
		}
		status := s.Status
		if fault > 0 {
			status = fault
		}
		final := s.Final
		if final == "" {
			final = q.URL.String()
		}
		u, _ := url.Parse(final)
		copy := q.Clone(q.Context())
		copy.URL = u
		headers := http.Header(s.Headers)
		if headers == nil {
			headers = make(http.Header)
		}
		return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(s.Body)), Request: copy}, nil
	})}
	return &Client{HTTP: h, Family: 4, Timeout: time.Second}, &calls
}
func TestAllSingleNativeRequests(t *testing.T) {
	data, err := os.ReadFile("testdata/single.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]struct{ Positive, Negative mockReply }
	if err = json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 107 {
		t.Fatalf("fixtures %d", len(fixtures))
	}
	for id, f := range fixtures {
		t.Run(id, func(t *testing.T) {
			for _, v := range []struct {
				name     string
				reply    mockReply
				expected string
			}{{"positive", f.Positive, "available"}, {"negative", f.Negative, "blocked"}} {
				t.Run(v.name, func(t *testing.T) {
					c, n := scripted(t, []mockReply{v.reply}, 0)
					got := execute(context.Background(), c, id)
					if id == "MediaUnlockTest_AbemaTV" && v.name == "negative" {
						v.expected = "restricted"
					}
					if id == "MediaUnlockTest_Showmax" && v.name == "negative" {
						v.expected = "error"
					}
					if got.status != v.expected {
						t.Fatalf("got %+v want %s", got, v.expected)
					}
					if *n != 1 {
						t.Fatalf("requests %d", *n)
					}
				})
			}
			for _, fault := range []int{-1, 429, 500} {
				c, _ := scripted(t, []mockReply{f.Positive}, fault)
				if got := execute(context.Background(), c, id); got.status != "error" {
					t.Errorf("fault %d got %+v", fault, got)
				}
			}
		})
	}
}
func TestAllMultiStepNativeRequests(t *testing.T) {
	data, err := os.ReadFile("testdata/flows.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]fixture
	if err = json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 18 {
		t.Fatalf("flow fixtures %d", len(fixtures))
	}
	for id, f := range fixtures {
		t.Run(id, func(t *testing.T) {
			for _, v := range []struct {
				name     string
				steps    []mockReply
				expected string
			}{{"positive", f.Positive, "available"}, {"negative", f.Negative, "blocked"}} {
				t.Run(v.name, func(t *testing.T) {
					c, n := scripted(t, v.steps, 0)
					got := execute(context.Background(), c, id)
					if got.status != v.expected {
						t.Fatalf("got %+v want %s (requests %d)", got, v.expected, *n)
					}
					if *n != len(v.steps) {
						t.Errorf("request count got %d want %d", *n, len(v.steps))
					}
				})
			}
			for _, fault := range []int{-1, 429, 500} {
				c, _ := scripted(t, f.Positive, fault)
				if got := execute(context.Background(), c, id); got.status != "error" {
					t.Errorf("fault %d got %+v", fault, got)
				}
			}
		})
	}
}
func TestCatalogueAndSafety(t *testing.T) {
	if len(Catalogue()) != 134 {
		t.Fatal("catalogue count")
	}
	count := map[string]int{"global": 6, "tw": 17, "hk": 14, "jp": 32, "na": 43, "sa": 9, "eu": 36, "oc": 25, "kr": 14, "sea": 17, "af": 9}
	for region, n := range count {
		checks, e := Selected(region)
		if e != nil || len(checks) != n {
			t.Fatalf("%s %d %v", region, len(checks), e)
		}
	}
	if _, e := Selected("in"); e == nil {
		t.Fatal("India silently added")
	}
	exceptions := 0
	for _, c := range Catalogue() {
		if c.Safety == "" {
			if len(requestCatalogue[c.ID]) == 0 {
				t.Errorf("no native request for %s", c.ID)
			}
			continue
		}
		exceptions++
		client, _ := scripted(t, nil, 0)
		r := Run(context.Background(), client, "global", []Check{c}, 1)
		if len(r) != 1 || r[0].Status != "error" || !strings.Contains(r[0].Detail, "safety") {
			t.Errorf("exception %+v", r)
		}
	}
	if exceptions != 9 {
		t.Fatal("exception count")
	}
}
