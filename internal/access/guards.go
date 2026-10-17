// MIT License; Copyright (c) 2026 Remnawave. See LICENSE.
package access

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

func responseProblem(status int, body string) string {
	if isChallenge(status, strings.ToLower(body)) {
		return "Cloudflare challenged the request, so availability was never tested"
	}
	if status == 429 || status >= 500 || status < 200 {
		return "unexpected HTTP " + itoa(status)
	}
	return ""
}
func textProblem(status int, body string) string {
	if p := responseProblem(status, body); p != "" {
		return p
	}
	if strings.TrimSpace(body) == "" {
		return "empty response; no availability evidence"
	}
	return ""
}
func validDestination(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || strings.Contains(raw, "\\") || strings.IndexFunc(raw, unicode.IsSpace) >= 0 {
		return nil, fmt.Errorf("invalid or missing final URL")
	}
	if port := u.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return nil, fmt.Errorf("invalid or missing final URL")
		}
	}
	return u, nil
}
