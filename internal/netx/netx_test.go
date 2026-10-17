// Offline HTTP/TLS tests. MIT; Copyright (c) 2026 Remnawave.
package netx

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestStack(t *testing.T, timeout time.Duration) *Stack {
	t.Helper()
	s, err := New(context.Background(), Options{Timeout: timeout})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}
func TestTransportRedirectHeadersAndGzip(t *testing.T) {
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "fixture-UA" || r.Header.Get("Accept-Language") != "en-US" || r.Header.Get("Accept-Encoding") != "gzip" {
			t.Error("headers not preserved")
		}
		if r.URL.Path == "/start" {
			http.Redirect(w, r, base+"/final", 302)
			return
		}
		w.Header().Set("Content-Encoding", "GZIP")
		gz := gzip.NewWriter(w)
		fmt.Fprint(gz, "ad-free")
		gz.Close()
	}))
	defer srv.Close()
	base = srv.URL
	s := newTestStack(t, time.Second)
	r, err := s.Do(context.Background(), 0, Request{URL: base + "/start", UserAgent: "fixture-UA", Headers: map[string]string{"Accept-Language": "en-US"}})
	if err != nil || r.FinalURL != base+"/final" || r.Text() != "ad-free" {
		t.Fatalf("%+v %v", r, err)
	}
}
func TestTransportErrorsAndSizeLimits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/403":
			w.WriteHeader(403)
			fmt.Fprint(w, "unsupported_country")
		case "/500":
			w.WriteHeader(500)
			fmt.Fprint(w, "ad-free")
		case "/truncated":
			w.Header().Set("Content-Length", "100")
			fmt.Fprint(w, "short")
		case "/bad-gzip":
			w.Header().Set("Content-Encoding", "gzip")
			fmt.Fprint(w, "not gzip")
		case "/encoding":
			w.Header().Set("Content-Encoding", "br")
			fmt.Fprint(w, "ok")
		case "/large":
			w.Write(bytes.Repeat([]byte("x"), MaxBodyBytes+1))
		case "/gzip-large":
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			gz.Write(bytes.Repeat([]byte("x"), MaxBodyBytes+1))
			gz.Close()
		case "/loop":
			http.Redirect(w, r, "/loop", 302)
		case "/no-location":
			w.WriteHeader(302)
		case "/scheme":
			http.Redirect(w, r, "file:///etc/passwd", 302)
		}
	}))
	defer srv.Close()
	s := newTestStack(t, time.Second)
	for _, path := range []string{"/truncated", "/bad-gzip", "/encoding", "/large", "/gzip-large", "/loop", "/no-location", "/scheme"} {
		t.Run(path, func(t *testing.T) {
			if r, err := s.Do(context.Background(), 0, Request{URL: srv.URL + path}); err == nil {
				t.Fatalf("accepted %+v", r)
			}
		})
	}
	for _, status := range []int{403, 500} {
		r, err := s.Do(context.Background(), 0, Request{URL: fmt.Sprintf("%s/%d", srv.URL, status)})
		if err != nil || r.Status != status || r.Text() == "" {
			t.Fatalf("%+v %v", r, err)
		}
	}
}
func TestTransportTimeoutAndCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/trickle" {
			w.WriteHeader(200)
			for {
				fmt.Fprint(w, "x")
				w.(http.Flusher).Flush()
				select {
				case <-r.Context().Done():
					return
				case <-time.After(5 * time.Millisecond):
				}
			}
		}
		if r.URL.Path == "/redirect" {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(15 * time.Millisecond):
				http.Redirect(w, r, "/redirect", 302)
				return
			}
		}
		<-r.Context().Done()
	}))
	defer srv.Close()
	s := newTestStack(t, 50*time.Millisecond)
	for _, path := range []string{"/hang", "/trickle", "/redirect"} {
		start := time.Now()
		_, err := s.Do(context.Background(), 0, Request{URL: srv.URL + path})
		if err == nil || time.Since(start) > time.Second {
			t.Fatalf("timeout: %v", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Do(ctx, 0, Request{URL: srv.URL + "/hang"}); err == nil {
		t.Fatal("cancelled request accepted")
	}
}
func TestTransportTLSVerificationAndDirectMode(t *testing.T) {
	tls := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") }))
	defer tls.Close()
	s := newTestStack(t, time.Second)
	if _, err := s.Do(context.Background(), 0, Request{URL: tls.URL}); err == nil {
		t.Fatal("untrusted TLS accepted")
	}
	tr := s.client.Transport.(*http.Transport)
	if tr.Proxy != nil || tr.TLSClientConfig != nil && tr.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("unsafe transport")
	}
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("NO_PROXY", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") }))
	defer srv.Close()
	if r, err := s.Do(context.Background(), 0, Request{URL: srv.URL}); err != nil || !strings.Contains(r.Text(), "ok") {
		t.Fatalf("%+v %v", r, err)
	}
}
