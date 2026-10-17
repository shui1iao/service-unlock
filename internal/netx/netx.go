// Minimal standard-library extraction of geocheck v0.3.0's HTTP stack.
// MIT License; Copyright (c) 2026 Remnawave. See LICENSE.
// Family is retained only for compatibility with the reused access code;
// this standalone client uses system-selected addresses, not a pinned family.
package netx

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const MaxBodyBytes = 4 << 20

type Family int

const (
	V4 Family = 4
	V6 Family = 6
)

type Options struct{ Timeout time.Duration }
type Stack struct {
	client  *http.Client
	timeout time.Duration
}
type Request struct {
	Method    string
	JSON      string
	HeadOnly  bool
	URL       string
	UserAgent string
	Headers   map[string]string
}
type Response struct {
	Status   int
	Header   http.Header
	Body     []byte
	FinalURL string
}

func (r *Response) Text() string {
	if r == nil {
		return ""
	}
	return string(r.Body)
}
func (r *Response) OK() bool { return r != nil && r.Status >= 200 && r.Status < 300 }

func New(_ context.Context, opts Options) (*Stack, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 6 * time.Second
	}
	tr := &http.Transport{
		Proxy:             nil, // Intentionally ignore ALL proxy environment variables.
		DialContext:       (&net.Dialer{Timeout: opts.Timeout, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2: true, DisableCompression: true,
		MaxIdleConns: 16, MaxIdleConnsPerHost: 4, IdleConnTimeout: 30 * time.Second,
		TLSHandshakeTimeout: opts.Timeout, ExpectContinueTimeout: time.Second,
	}
	client := &http.Client{Transport: tr, Timeout: opts.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			if (req.URL.Scheme != "http" && req.URL.Scheme != "https") || req.URL.User != nil {
				return fmt.Errorf("unsupported redirect destination")
			}
			return nil
		},
	}
	return &Stack{client: client, timeout: opts.Timeout}, nil
}
func (s *Stack) HTTP(_ Family) *http.Client { return s.client }
func (s *Stack) Close()                     { s.client.CloseIdleConnections() }

func readBounded(r io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, MaxBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > MaxBodyBytes {
		return nil, fmt.Errorf("response exceeds 4 MiB; not classified")
	}
	return body, nil
}
func (s *Stack) Do(ctx context.Context, _ Family, req Request) (*Response, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	if req.HeadOnly {
		method = http.MethodHead
	}
	var body io.Reader
	if req.JSON != "" {
		body = strings.NewReader(req.JSON)
	}
	request, err := http.NewRequestWithContext(ctx, method, req.URL, body)
	if err != nil {
		return nil, err
	}
	if (request.URL.Scheme != "http" && request.URL.Scheme != "https") || request.URL.User != nil {
		return nil, fmt.Errorf("unsupported request destination")
	}
	ua := req.UserAgent
	if ua == "" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36"
	}
	request.Header.Set("User-Agent", ua)
	if req.JSON != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Accept-Encoding", "gzip")
	for name, value := range req.Headers {
		request.Header.Set(name, value)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	// Incomplete redirect responses are not success evidence.
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		return nil, fmt.Errorf("redirect failed")
	}
	if req.HeadOnly {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return &Response{Status: response.StatusCode, Header: response.Header, FinalURL: response.Request.URL.String()}, nil
	}
	raw, err := readBounded(response.Body)
	if err != nil {
		return nil, err
	}
	if response.ContentLength >= 0 && int64(len(raw)) != response.ContentLength {
		return nil, fmt.Errorf("incomplete response body")
	}
	switch encoding := strings.ToLower(strings.TrimSpace(response.Header.Get("Content-Encoding"))); encoding {
	case "", "identity":
	case "gzip":
		gz, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		raw, err = readBounded(gz)
		closeErr := gz.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
	default:
		return nil, fmt.Errorf("unsupported Content-Encoding: %s", encoding)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &Response{Status: response.StatusCode, Header: response.Header, Body: raw, FinalURL: response.Request.URL.String()}, nil
}
