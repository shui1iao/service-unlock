// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func huluLoopback(t *testing.T, timeout time.Duration, h2, h1 http.HandlerFunc) (*Client, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	n2, n1 := new(atomic.Int32), new(atomic.Int32)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "id.hulu.jp" {
			t.Errorf("Hulu host changed: %s", r.Host)
		}
		if r.Method != "GET" {
			t.Error("fallback not GET")
		}
		if r.ProtoMajor == 2 {
			n2.Add(1)
			h2(w, r)
		} else {
			n1.Add(1)
			h1(w, r)
		}
	}))
	server.EnableHTTP2 = true
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	t.Cleanup(server.Close)
	c := NewClient(4, timeout)
	tr := c.HTTP.Transport.(*http.Transport)
	trusted := server.Client().Transport.(*http.Transport)
	tr.TLSClientConfig = trusted.TLSClientConfig.Clone()
	u, _ := url.Parse(server.URL)
	tr.TLSClientConfig.ServerName = u.Hostname()
	tr.Protocols = new(http.Protocols)
	tr.Protocols.SetHTTP1(true)
	tr.Protocols.SetHTTP2(true)
	tr.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		// net/http calls the configured dial hook with tcp; this hook, like
		// NewClient's production hook, pins its actual dial to tcp4.
		if network != "tcp" || address != "id.hulu.jp:443" {
			t.Errorf("family/source changed: %s %s", network, address)
		}
		return (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp4", u.Host)
	}
	t.Cleanup(c.Close)
	return c, n2, n1
}
func huluRST(w http.ResponseWriter, r *http.Request) { panic(http.ErrAbortHandler) }
func TestJPHuluActualTypedHTTP2ErrorFallsBackOnce(t *testing.T) {
	c, n2, n1 := huluLoopback(t, time.Second, huluRST, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept-Language") == "" {
			t.Error("headers lost")
		}
		if r.URL.Path == "/" {
			http.Redirect(w, r, "https://id.hulu.jp/_error/restrict.html", 302)
			return
		}
		fmt.Fprint(w, "<html>restricted</html>")
	})
	v := execute(context.Background(), c, "MediaUnlockTest_HuluJP")
	if v.status != "blocked" || !strings.Contains(v.detail, "HTTP/1.1 fallback") {
		t.Fatalf("real standard HTTP2 RST: %+v h2=%d h1=%d", v, n2.Load(), n1.Load())
	}
	if n2.Load() != 1 || n1.Load() != 2 {
		t.Fatal("one h2 + one h1 attempt including its redirect", n2.Load(), n1.Load())
	}
	if dir := os.Getenv("JP_FIX_ARTIFACT_DIR"); dir != "" {
		os.MkdirAll(dir, 0700)
		data, _ := json.MarshalIndent(map[string]any{"loopback_only": true, "live_hulu_retested": false, "before": "Unknown: actual standard HTTP2 StreamError INTERNAL_ERROR", "after": v.status, "detail": v.detail, "http2_requests": n2.Load(), "http1_requests_including_redirect": n1.Load(), "fallback_attempts": 1}, "", "  ")
		if e := os.WriteFile(filepath.Join(dir, "hulu-loopback-before-after.json"), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	tr := c.HTTP.Transport.(*http.Transport)
	if !tr.ForceAttemptHTTP2 || !tr.Protocols.HTTP2() || tr.Proxy != nil || tr.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("global source transport mutated")
	}
}
func TestJPHuluFallbackSharesOriginalBudget(t *testing.T) {
	c, n2, n1 := huluLoopback(t, 160*time.Millisecond, func(w http.ResponseWriter, r *http.Request) { time.Sleep(100 * time.Millisecond); huluRST(w, r) }, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(100 * time.Millisecond):
			fmt.Fprint(w, "landing")
		case <-r.Context().Done():
		}
	})
	start := time.Now()
	v := execute(context.Background(), c, "MediaUnlockTest_HuluJP")
	elapsed := time.Since(start)
	if v.status != "error" || !strings.Contains(v.detail, "HTTP/1.1 fallback") || elapsed > 260*time.Millisecond || n2.Load() != 1 || n1.Load() != 1 {
		t.Fatalf("budget reset or retry leak: %+v elapsed=%v h2=%d h1=%d", v, elapsed, n2.Load(), n1.Load())
	}
}
func TestJPHuluNoStatusOrGenericNetworkRetry(t *testing.T) {
	for _, status := range []int{200, 403, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			c, n2, n1 := huluLoopback(t, time.Second, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status); fmt.Fprint(w, "normal response") }, func(w http.ResponseWriter, r *http.Request) { t.Error("unwanted h1 retry") })
			execute(context.Background(), c, "MediaUnlockTest_HuluJP")
			if n2.Load() != 1 || n1.Load() != 0 {
				t.Fatal(n2.Load(), n1.Load())
			}
		})
	}
	for _, err := range []error{errors.New("stream error: stream ID 1; INTERNAL_ERROR; received from peer"), errors.New("network reset"), context.Canceled, context.DeadlineExceeded, huluStreamError{1, 7, nil}, huluStreamError{1, 8, nil}, huluStreamError{1, 2, context.Canceled}} {
		if isHuluProtocolError(err) {
			t.Errorf("not a retryable typed protocol error: %v", err)
		}
	}
}
func TestJPHuluNoSecondFallbackAndCancellation(t *testing.T) {
	c, n2, n1 := huluLoopback(t, time.Second, huluRST, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500); fmt.Fprint(w, "server error") })
	v := execute(context.Background(), c, "MediaUnlockTest_HuluJP")
	if v.status != "error" || n2.Load() != 1 || n1.Load() != 1 || !strings.Contains(v.detail, "fallback") {
		t.Fatal(v, n2.Load(), n1.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	v = execute(ctx, c, "MediaUnlockTest_HuluJP")
	if v.status != "error" || n2.Load() != 1 || n1.Load() != 1 {
		t.Fatal("cancel retried", v)
	}
}
func TestJPHuluFallbackCannotApplyToOtherProbeOrPOST(t *testing.T) {
	c, n2, n1 := huluLoopback(t, time.Second, huluRST, func(w http.ResponseWriter, r *http.Request) { t.Error("other probe changed HTTP policy") })
	q := requestCatalogue["MediaUnlockTest_HuluJP"][0]
	_, err := c.do(context.Background(), q)
	if err == nil || n2.Load() != 1 || n1.Load() != 0 {
		t.Fatal("ordinary do got fallback")
	}
	// Same URL but no probe opt-in has already proven no fallback. Typed
	// recognition is checked using the actual standard-library error, not text.
	if !isHuluProtocolError(err) {
		t.Fatalf("standard error type not recognized: %T %v", err, err)
	}
	q.Method = "POST"
	c.HTTP.Transport = transportFunc(func(r *http.Request) (*http.Response, error) { return nil, huluStreamError{1, 2, nil} })
	if _, err = c.doWithHuluFallback(context.Background(), q, true); err == nil || strings.Contains(err.Error(), "cannot apply") {
		t.Fatal("POST entered fallback", err)
	}
}

func TestJPHuluFallbackStillRejectsDowngradeOversizeAndHonorsCancel(t *testing.T) {
	for _, mode := range []string{"downgrade", "oversize"} {
		t.Run(mode, func(t *testing.T) {
			c, n2, n1 := huluLoopback(t, time.Second, huluRST, func(w http.ResponseWriter, r *http.Request) {
				if mode == "downgrade" {
					http.Redirect(w, r, "http://must-not-connect.invalid/", 302)
				} else {
					fmt.Fprint(w, strings.Repeat("x", maxBody+1))
				}
			})
			v := execute(context.Background(), c, "MediaUnlockTest_HuluJP")
			if v.status != "error" || n2.Load() != 1 || n1.Load() != 1 || !strings.Contains(v.detail, "fallback") {
				t.Fatal(mode, v, n2.Load(), n1.Load())
			}
		})
	}
	entered := make(chan struct{})
	finished := make(chan struct{})
	c, n2, n1 := huluLoopback(t, time.Second, huluRST, func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done(); close(finished) })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan verdict, 1)
	go func() { done <- execute(ctx, c, "MediaUnlockTest_HuluJP") }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("fallback did not start")
	}
	cancel()
	select {
	case v := <-done:
		if v.status != "error" || !strings.Contains(v.detail, "fallback") {
			t.Fatal(v)
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not stop fallback")
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("fallback body/connection leaked")
	}
	if n2.Load() != 1 || n1.Load() != 1 {
		t.Fatal("cancel caused retry", n2.Load(), n1.Load())
	}
}
