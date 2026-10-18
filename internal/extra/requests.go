// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

//go:embed scalar-types.json
var scalarTypeBytes []byte
var scalarTypes = func() map[string][2]string {
	var m map[string][2]string
	if err := json.Unmarshal(scalarTypeBytes, &m); err != nil {
		panic(err)
	}
	return m
}()

func validScalarType(id, body string) bool {
	spec, ok := scalarTypes[id]
	if !ok || !validJSON(body) {
		return true
	}
	var value any
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return false
	}
	var visit func(any) bool
	visit = func(v any) bool {
		switch object := v.(type) {
		case map[string]any:
			for k, item := range object {
				if k == spec[0] {
					switch spec[1] {
					case "string":
						if _, ok := item.(string); !ok {
							return false
						}
					case "number":
						if _, ok := item.(json.Number); !ok {
							return false
						}
					case "bool":
						if _, ok := item.(bool); !ok {
							return false
						}
					}
				}
				if !visit(item) {
					return false
				}
			}
		case []any:
			for _, item := range object {
				if !visit(item) {
					return false
				}
			}
		}
		return true
	}
	return visit(value)
}

//go:embed requests-source.json
var requestBytes []byte
var requestCatalogue = func() map[string][]request {
	var m map[string][]request
	if err := json.Unmarshal(requestBytes, &m); err != nil {
		panic(err)
	}
	return m
}()

func allText(r response) string {
	var b strings.Builder
	b.WriteString(r.Text)
	b.WriteByte('\n')
	for k, vs := range r.Header {
		for _, v := range vs {
			b.WriteString(k + ": " + v + "\n")
		}
	}
	return b.String()
}
func uuid() string {
	s := randomID()
	return s[:8] + "-" + s[8:12] + "-4" + s[13:16] + "-a" + s[17:20] + "-" + s[20:]
}
func variables() map[string]string {
	return map[string]string{"randsession": randomID(), "fakePUid": uuid(), "fakeUuid": uuid(), "fakeIfaId": uuid(), "fakeDeviceId": randomID(), "timestamp": strconv.FormatInt(time.Now().Unix(), 10), "SEC_CH_UA": "", "UA_Browser": browserUA, "UA_SecCHUA": ""}
}
func expand(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
		s = strings.ReplaceAll(s, "$"+k, v)
	}
	return s
}
func (c *Client) step(ctx context.Context, id string, index int, vars map[string]string) (response, error) {
	requests, ok := requestCatalogue[id]
	if !ok || index >= len(requests) {
		return response{}, fmt.Errorf("missing fixed request step %s/%d", id, index)
	}
	q := requests[index]
	q.URL = expand(q.URL, vars)
	q.URL = strings.ReplaceAll(strings.ReplaceAll(q.URL, `\[`, `[`), `\]`, `]`)
	q.Body = expand(q.Body, vars)
	q.Headers = map[string]string{}
	for k, v := range requests[index].Headers {
		q.Headers[k] = expand(v, vars)
	}
	// Bahamut's original temporary cookie file is replaced with an in-memory jar.
	if q.Headers["Cookie"] == "bahamut_cookie.txt" {
		delete(q.Headers, "Cookie")
	}
	if strings.Contains(q.URL, "${") || strings.Contains(q.Body, "${") {
		return response{}, fmt.Errorf("unresolved source parameter for %s", id)
	}
	for k, v := range q.Headers {
		if strings.Contains(k, "${") || strings.Contains(v, "${") {
			return response{}, fmt.Errorf("unresolved source header for %s", id)
		}
	}
	if id == "MediaUnlockTest_HuluJP" {
		return c.doWithHuluFallback(ctx, q, true)
	}
	return c.do(ctx, q)
}
func execute(ctx context.Context, c *Client, id string) verdict {
	if parser, ok := singleParsers[id]; ok {
		vars := variables()
		if id == "MediaUnlockTest_NaverTV" {
			ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
			endpoint := "https://apis.naver.com/now_web2/now_web_api/v1/clips/31030608/play-info"
			vars["req_url"] = endpoint + "?msgpad=" + ts + "&md=" + url.QueryEscape(hmacSignature(endpoint+ts, "nbxvs5nwNG9QKEWK0ADjYA4JZoujF4gHcIwvoCxFTPAeamq5eemvt5IWAYXxrbYM"))
		}
		r, err := c.step(ctx, id, 0, vars)
		if err != nil {
			return unknown(err.Error())
		}
		if jsonResponseRequired[id] && !validJSON(r.Text) {
			if id == "MediaUnlockTest_Dazn" && has(r.Text, `(?i)Security policy has been breached`) {
				return no("IP Banned by Dazn")
			}
			return unknown("malformed source JSON")
		}
		if !validScalarType(id, r.Text) {
			return unknown("wrong source JSON field type")
		}
		v := parser(r)
		if r.HTTP1Fallback {
			v.detail += "; HTTP/1.1 fallback after typed HTTP/2 stream error (same request budget)"
		}
		return v
	}
	return runFlow(ctx, c, id)
}
