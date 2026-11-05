package access

import (
	"context"
	"errors"
	"io"
	"net/http"
	"serviceunlock/internal/netx"
	"strings"
	"testing"
	"time"
)

type metadataTrip func(*http.Request) (*http.Response, error)

func (f metadataTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestAccessAuxiliaryPreservesEveryPrimaryField(t *testing.T) {
	for _, state := range []State{StateAvailable, StateBlocked, StateRestricted, StateError} {
		for _, fail := range []bool{false, true} {
			env := testEnv(t)
			sentinel := errors.New("primary sentinel")
			env.Stack.HTTP(0).Transport = metadataTrip(func(r *http.Request) (*http.Response, error) {
				time.Sleep(30 * time.Millisecond)
				if fail {
					return nil, errors.New("private aux failure")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"derived_fields":{"country":"JP"}}`)), ContentLength: -1, Request: r}, nil
			})
			c := Check{ID: "chatgpt_web", Name: "ChatGPT (web)", Run: func(context.Context, Env) Result {
				return Result{State: state, Detail: "primary detail", Err: sentinel, RTT: 17 * time.Millisecond}
			}}
			r := Run(context.Background(), env.Stack, 0, []Check{c}, 4)[0]
			if r.State != state || r.Detail != "primary detail" || r.Err != sentinel || r.RTT != 17*time.Millisecond || r.CheckedAt.IsZero() || fail && r.Region != "" || !fail && r.Region != "JP" {
				t.Fatal(r, fail)
			}
		}
	}
}
func TestExistingPrimaryRegionNotOverwrittenOrRequeried(t *testing.T) {
	env := testEnv(t)
	env.Stack.HTTP(0).Transport = metadataTrip(func(r *http.Request) (*http.Response, error) {
		t.Fatal("existing country requeried", r.URL)
		return nil, nil
	})
	for _, id := range []string{"chatgpt_web", "youtube_premium_access", "tiktok_access", "netflix_access", "claude_access", "gemini_access"} {
		c := Check{ID: id, Run: func(context.Context, Env) Result {
			return Result{State: StateAvailable, Region: "SG", RTT: time.Millisecond}
		}}
		r := Run(context.Background(), env.Stack, 0, []Check{c}, 4)[0]
		if r.Region != "SG" {
			t.Fatal(r)
		}
	}
}
func TestYouTubeReusesPrimaryCountryButNeverChangesPremiumVerdict(t *testing.T) {
	for _, tc := range []struct {
		body  string
		state State
	}{{`ad-free "countryCode":"JP"`, StateAvailable}, {`YouTube Premium is not available in your country "countryCode":"JP"`, StateBlocked}, {`no conclusive wording "countryCode":"JP"`, StateError}} {
		env := testEnv(t)
		calls := 0
		env.Stack.HTTP(0).Transport = metadataTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.String() != "https://www.youtube.com/premium" {
				t.Fatal("unnecessary auxiliary request", r.URL)
			}
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(tc.body)), ContentLength: -1, Request: r}, nil
		})
		r := Run(context.Background(), env.Stack, 0, []Check{youTubePremium()}, 4)[0]
		if calls != 1 || r.Region != "JP" || r.State != tc.state {
			t.Fatal(r, calls)
		}
	}
}

func TestMeasuredPrimaryRTTDoesNotIncludeMetadataWait(t *testing.T) {
	env := testEnv(t)
	env.Stack.HTTP(0).Transport = metadataTrip(func(r *http.Request) (*http.Response, error) {
		time.Sleep(45 * time.Millisecond)
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"derived_fields":{"country":"JP"}}`)), ContentLength: -1, Request: r}, nil
	})
	start := time.Now()
	c := Check{ID: "chatgpt_web", Run: func(context.Context, Env) Result { return Result{State: StateAvailable} }}
	r := Run(context.Background(), env.Stack, 0, []Check{c}, 4)[0]
	if r.RTT >= 30*time.Millisecond || r.Region != "JP" || time.Since(start) < 45*time.Millisecond || r.State != StateAvailable || r.Err != nil {
		t.Fatal(r)
	}
}

func TestAuxiliaryOwnTimeoutCannotMakePrimaryError(t *testing.T) {
	stack, _ := netx.New(context.Background(), netx.Options{Timeout: 25 * time.Millisecond})
	defer stack.Close()
	stack.HTTP(0).Transport = metadataTrip(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })
	c := Check{ID: "chatgpt_web", Run: func(context.Context, Env) Result {
		return Result{State: StateAvailable, Detail: "primary success", RTT: time.Millisecond}
	}}
	r := Run(context.Background(), stack, 0, []Check{c}, 4)[0]
	if r.State != StateAvailable || r.Err != nil || r.Detail != "primary success" || r.RTT != time.Millisecond || r.Region != "" {
		t.Fatal("aux timeout changed primary", r)
	}
}
