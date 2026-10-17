package netx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPOSTJSONHEADAndDefaultUA(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("User-Agent"), "Chrome/151.0.0.0") {
			t.Error("upstream default UA missing")
		}
		switch r.URL.Path {
		case "/post":
			if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Client-Id") != "public-fixture" {
				t.Error("POST headers", r.Method, r.Header)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["operationName"] != "Fixture" {
				t.Error("POST JSON", body, err)
			}
			http.Redirect(w, r, "/echo", 307)
		case "/echo":
			if r.Method != "POST" {
				t.Error("307 lost POST")
			}
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.Write(body)
		case "/head":
			if r.Method != "HEAD" {
				t.Error("not HEAD")
			}
			w.Header().Set("Set-Cookie", "country=US; Path=/")
			w.Header().Set("Content-Length", "99999999")
			w.Header().Set("Content-Encoding", "gzip")
		case "/empty-post":
			if r.Method != "POST" || r.ContentLength != 0 {
				t.Error("empty POST changed")
			}
			fmt.Fprint(w, `{"ok":true}`)
		}
	}))
	defer srv.Close()
	stack := newTestStack(t, time.Second)
	out, err := stack.Do(context.Background(), 0, Request{URL: srv.URL + "/post", Method: "POST", JSON: `{"operationName":"Fixture"}`, Headers: map[string]string{"Client-Id": "public-fixture"}})
	if err != nil || out.Text() != `{"operationName":"Fixture"}` || out.FinalURL != srv.URL+"/echo" {
		t.Fatal(out, err)
	}
	out, err = stack.Do(context.Background(), 0, Request{URL: srv.URL + "/head", HeadOnly: true})
	if err != nil || len(out.Body) != 0 || out.Header.Get("Set-Cookie") == "" {
		t.Fatal(out, err)
	}
	out, err = stack.Do(context.Background(), 0, Request{URL: srv.URL + "/empty-post", Method: "POST"})
	if err != nil || !out.OK() {
		t.Fatal(out, err)
	}
}
func TestPOSTTimeoutAndCancelClosesServerRequest(t *testing.T) {
	done := make(chan struct{}, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
		done <- struct{}{}
	}))
	defer srv.Close()
	stack := newTestStack(t, 30*time.Millisecond)
	if _, err := stack.Do(context.Background(), 0, Request{URL: srv.URL, Method: "POST", JSON: `{"fixture":true}`}); err == nil {
		t.Fatal("timeout accepted")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout left request open")
	}
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(10*time.Millisecond, cancel)
	defer timer.Stop()
	defer cancel()
	if _, err := stack.Do(ctx, 0, Request{URL: srv.URL, Method: "POST", JSON: `{}`}); err == nil {
		t.Fatal("cancelled request accepted")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancellation left request open")
	}
}
