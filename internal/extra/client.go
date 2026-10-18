// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxBody = 4 << 20
const browserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"

type request struct {
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	BodyBase64 string            `json:"body_base64"`
}
type response struct {
	Status      int
	Text, Final string
	Header      http.Header
}
type Client struct {
	HTTP    *http.Client
	Family  int
	Timeout time.Duration
}

func NewClient(family int, timeout time.Duration) *Client {
	tr := &http.Transport{Proxy: nil, ForceAttemptHTTP2: true, DisableCompression: true, MaxIdleConns: 32, MaxIdleConnsPerHost: 4, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: timeout,
		DialContext: func(ctx context.Context, _ string, address string) (net.Conn, error) {
			return (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp"+strconv.Itoa(family), address)
		}}
	return &Client{HTTP: &http.Client{Transport: tr, Timeout: timeout, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("redirect limit")
		}
		if r.URL.Scheme != "https" || r.URL.User != nil {
			return fmt.Errorf("unsafe redirect")
		}
		return nil
	}}, Family: family, Timeout: timeout}
}
func (c *Client) Close() { c.HTTP.CloseIdleConnections() }
func (c *Client) withCookies() *Client {
	copy := *c
	h := *c.HTTP
	h.Jar, _ = cookiejar.New(nil)
	copy.HTTP = &h
	return &copy
}
func bounded(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxBody+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBody {
		return nil, fmt.Errorf("response exceeds 4 MiB")
	}
	return data, nil
}
func (c *Client) do(ctx context.Context, q request) (response, error) {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	u, err := url.Parse(q.URL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return response{}, fmt.Errorf("unsafe or invalid source URL")
	}
	body := []byte(q.Body)
	if q.BodyBase64 != "" {
		body, err = base64.StdEncoding.DecodeString(q.BodyBase64)
		if err != nil {
			return response{}, err
		}
	}
	method := q.Method
	if method == "" {
		method = "GET"
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
	if err != nil {
		return response{}, err
	}
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept-Encoding", "gzip")
	for k, v := range q.Headers {
		if strings.EqualFold(k, "host") {
			req.Host = v
		} else {
			req.Header.Set(k, v)
		}
	}
	raw, err := c.HTTP.Do(req)
	if err != nil {
		return response{}, err
	}
	defer raw.Body.Close()
	data := []byte{}
	if method != "HEAD" {
		data, err = bounded(raw.Body)
		if err != nil {
			return response{}, err
		}
		switch strings.ToLower(strings.TrimSpace(raw.Header.Get("Content-Encoding"))) {
		case "", "identity":
		case "gzip":
			gz, e := gzip.NewReader(bytes.NewReader(data))
			if e != nil {
				return response{}, e
			}
			data, err = bounded(gz)
			gz.Close()
			if err != nil {
				return response{}, err
			}
		default:
			return response{}, fmt.Errorf("unsupported content encoding")
		}
	}
	if ctx.Err() != nil {
		return response{}, ctx.Err()
	}
	r := response{raw.StatusCode, string(data), raw.Request.URL.String(), raw.Header}
	if r.Status == 429 || r.Status >= 500 || r.Status < 200 {
		return r, fmt.Errorf("unexpected HTTP %d", r.Status)
	}
	if method != "HEAD" && r.Status >= 200 && r.Status < 300 && strings.TrimSpace(r.Text) == "" {
		return r, fmt.Errorf("empty response body")
	}
	if r.Status >= 200 {
		for _, m := range []string{"cf_chl_opt", "_cf_chl", "just a moment", "cf-browser-verification", "enable javascript and cookies to continue"} {
			if strings.Contains(strings.ToLower(r.Text), m) {
				return r, fmt.Errorf("challenged before reaching service")
			}
		}
	}
	return r, nil
}
func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func rx(text, pattern string) string {
	m := regexp.MustCompile(pattern).FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}
func has(text, pattern string) bool { return regexp.MustCompile(pattern).MatchString(text) }
func field(text, key string) string {
	return rx(text, `"`+regexp.QuoteMeta(key)+`"\s*:\s*(?:"([^"]*)"|(?:true|false|-?[0-9]+))`)
}
func scalar(text, key string) string {
	m := regexp.MustCompile(`"` + regexp.QuoteMeta(key) + `"\s*:\s*("([^"]*)"|true|false|-?[0-9]+)`).FindStringSubmatch(text)
	if len(m) < 3 {
		return ""
	}
	if strings.HasPrefix(m[1], `"`) {
		return m[2]
	}
	return m[1]
}
func pathSegment(final string, index int) string {
	u, err := url.Parse(final)
	if err != nil {
		return ""
	}
	parts := strings.Split(u.Path, "/")
	if index < len(parts) {
		return strings.ToUpper(parts[index])
	}
	return ""
}
func hmacSignature(text, key string) string {
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(text))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
func validJSON(text string) bool { return json.Valid([]byte(text)) }
