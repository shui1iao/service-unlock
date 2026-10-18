package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/netx"
)

type requestFixture struct {
	ID, URL, Method, Body, Final string
	Status                       int
	ResponseHeaders              http.Header `json:"response_headers"`
}
type completeTransport struct {
	fixtures     map[string]requestFixture
	want, seen   map[string]int
	mu           sync.Mutex
	active, peak atomic.Int32
	failURL      string
	delay        time.Duration
}

func (t *completeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	n := t.active.Add(1)
	defer t.active.Add(-1)
	for old := t.peak.Load(); n > old && !t.peak.CompareAndSwap(old, n); old = t.peak.Load() {
	}
	t.mu.Lock()
	t.seen[req.URL.String()]++
	f, ok := t.fixtures[req.URL.String()]
	t.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("offline guard: unexpected URL %s", req.URL)
	}
	if req.Method != f.Method {
		return nil, fmt.Errorf("unexpected method %s for %s", req.Method, req.URL)
	}
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-time.After(t.delay):
	}
	if req.URL.String() == t.failURL {
		f.Status = 500
		f.Body = "server error"
	}
	headers := f.ResponseHeaders
	if headers == nil {
		headers = make(http.Header)
	}
	returnedRequest := req
	if f.Final != "" {
		returnedRequest = req.Clone(req.Context())
		returnedRequest.URL, _ = url.Parse(f.Final)
	}
	return &http.Response{StatusCode: f.Status, Body: io.NopCloser(strings.NewReader(f.Body)), Header: headers, ContentLength: -1, Request: returnedRequest}, nil
}
func mockCompleteStack(t *testing.T) (*netx.Stack, *completeTransport) {
	t.Helper()
	tr := &completeTransport{fixtures: map[string]requestFixture{}, want: map[string]int{}, seen: map[string]int{}, delay: 2 * time.Millisecond}
	add := func(f requestFixture) {
		if f.Status == 0 {
			f.Status = 200
		}
		if f.Method == "" {
			f.Method = "GET"
		}
		tr.fixtures[f.URL] = f
		tr.want[f.URL]++
	}
	for _, p := range []string{"internal/geo/testdata/services.json", "internal/ai/testdata/endpoints.json"} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct{ Checks []requestFixture }
		if err = json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		for _, f := range doc.Checks {
			if strings.Contains(p, "/ai/") {
				f.Status = 401
				f.Body = `{"error":"credentials not supplied"}`
			}
			add(f)
		}
	}
	for _, f := range []requestFixture{
		{URL: "https://api.openai.com/compliance/cookie_requirements", Body: `{}`},
		{URL: "https://www.youtube.com/premium", Body: "ad-free offer"},
		{URL: "https://www.netflix.com/title/70143836", Body: "licensed title"},
		{URL: "https://claude.ai/cdn-cgi/trace", Body: "loc=US\n"}, {URL: "https://claude.ai/", Body: "Claude landing page"},
		{URL: "https://www.tiktok.com/", Body: "TikTok landing page"},
		{URL: "https://gemini.google.com/app", Body: `,3,7,200,"USA"`},
	} {
		add(f)
	}
	stack, err := netx.New(context.Background(), netx.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	stack.HTTP(0).Transport = tr
	t.Cleanup(stack.Close)
	return stack, tr
}
func TestComplete35RealCallPathsAndAdditiveJSON(t *testing.T) {
	stack, tr := mockCompleteStack(t)
	m, err := runGroups(context.Background(), stack, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Access) != 6 || len(m.Services) != 21 || len(m.AI) != 8 {
		t.Fatalf("counts=%d/%d/%d", len(m.Access), len(m.Services), len(m.AI))
	}
	if !reflect.DeepEqual(tr.seen, tr.want) {
		t.Fatalf("not all exact request paths executed\nseen=%v\nwant=%v", tr.seen, tr.want)
	}
	if tr.active.Load() != 0 || tr.peak.Load() > 3 || tr.peak.Load() < 2 {
		t.Fatal("global request concurrency", tr.active.Load(), tr.peak.Load())
	}
	for _, r := range m.Access {
		if r.State != access.StateAvailable {
			t.Fatal(r)
		}
	}
	for _, r := range m.Services {
		if r.Err != nil || r.Value == "" {
			t.Fatal(r)
		}
	}
	for _, r := range m.AI {
		if r.State != ai.StateReachable || r.Status != 401 {
			t.Fatal(r)
		}
	}
	report := makeCompleteReport(m, 6, 3)
	baseline := makeReport(m.Access, 6, 3)
	if !reflect.DeepEqual(report.Results, baseline.Results) {
		t.Fatal("legacy results semantics changed")
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err = json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc) != 9 || doc["services"] == nil || doc["ai_endpoints"] == nil || bytes.Contains(data, []byte("\x1b")) {
		t.Fatal("schema mismatch")
	}
	if report.Network.Mode != "direct" || report.Network.AddressFamily != "system-selected" {
		t.Fatal("fabricated family/proxy")
	}
	for _, r := range report.Services {
		if r.Status != "observed" || r.Value == nil || r.Error != nil {
			t.Fatal(r)
		}
		if r.Type != "country" && r.Type != "availability" && r.Type != "blocked" {
			t.Fatal(r.Type)
		}
	}
	for _, r := range report.AIEndpoints {
		if r.Status != "reachable" || r.HTTPStatus != 401 || r.Error != nil {
			t.Fatal(r)
		}
	}
}
func TestNewGroupErrorsAffectExitNotLegacyArray(t *testing.T) {
	for _, failure := range []string{"", "https://gspe1-ssl.ls.apple.com/pep/gcc", "https://api.deepseek.com/models"} {
		stack, tr := mockCompleteStack(t)
		tr.failURL = failure
		out, errout := new(bytes.Buffer), new(bytes.Buffer)
		code := runCLI(context.Background(), []string{"--json"}, out, errout, func(ctx context.Context, _ time.Duration, w int) (measurements, error) {
			return runGroups(ctx, stack, w)
		})
		want := 0
		if failure != "" {
			want = 1
		}
		if code != want || errout.Len() != 0 {
			t.Fatal(code, errout)
		}
		var r report
		if err := json.Unmarshal(out.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if len(r.Results) != 6 || len(r.Services) != 21 || len(r.AIEndpoints) != 8 {
			t.Fatal("scope")
		}
		for _, r := range r.Results {
			if r.Status != "available" {
				t.Fatal("new group failure changed legacy result", r)
			}
		}
		if strings.Contains(failure, "apple") {
			if r.Services[14].Status != "error" || r.Services[14].Value != nil || r.Services[14].Error == nil {
				t.Fatal(r.Services[14])
			}
		}
		if strings.Contains(failure, "deepseek") {
			if r.AIEndpoints[3].Status != "error" || r.AIEndpoints[3].Error == nil {
				t.Fatal(r.AIEndpoints[3])
			}
		}
	}
}
func TestCompleteCancellationStopsRemainingGroups(t *testing.T) {
	stack, tr := mockCompleteStack(t)
	tr.delay = time.Second
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(15*time.Millisecond, cancel)
	defer timer.Stop()
	defer cancel()
	out := new(bytes.Buffer)
	start := time.Now()
	code := runCLI(ctx, []string{"--json"}, out, io.Discard, func(ctx context.Context, _ time.Duration, w int) (measurements, error) {
		return runGroups(ctx, stack, w)
	})
	if code != 130 || out.Len() != 0 || time.Since(start) > time.Second || tr.active.Load() != 0 {
		t.Fatal(code, out, tr.active.Load())
	}
	for _, f := range []string{"https://www.google.com", "https://api.deepseek.com/models"} {
		if tr.seen[f] != 0 {
			t.Fatal("later group ran after cancellation")
		}
	}
}
func TestCompleteRunReportsRunnerFailure(t *testing.T) {
	if code := runCLI(context.Background(), nil, io.Discard, io.Discard, func(context.Context, time.Duration, int) (measurements, error) {
		return measurements{}, errors.New("fixture failure")
	}); code != 1 {
		t.Fatal(code)
	}
}
