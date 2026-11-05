// Six optional, same-platform country observations, adapted from the retired
// geo checks in service-unlock v0.5.6 (53e7e8f1), originally geocheck MIT,
// Copyright (c) 2026 Remnawave. No check registration or availability verdict.
package geo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"serviceunlock/internal/countries"
	"serviceunlock/internal/netx"
)

var youtubeCountry = regexp.MustCompile(`"countryCode"\s*:\s*"([^"]*)"`)
var googleUnderscore = regexp.MustCompile(`"[a-z]{2}_([A-Z]{2})"`)
var googleDash = regexp.MustCompile(`"[a-z]{2}-([A-Z]{2})"`)
var googlePlayCountry = regexp.MustCompile(`<div class="yVZQTb">([^<(]+)`)
var redditCountry = regexp.MustCompile(`country="([^"]*)"`)

// metadataCountry deliberately excludes provider pseudo-codes, not merely WW.
func metadataCountry(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 2 || !countries.Known(value) || value == "WW" || value == "EU" || value == "UK" {
		return ""
	}
	return value
}

// uniqueCountry fails closed on conflicting or malformed observations.
func uniqueCountry(matches [][]string) string {
	value := ""
	for _, m := range matches {
		code := metadataCountry(m[1])
		if code == "" || value != "" && code != value {
			return ""
		}
		value = code
	}
	return value
}

// YouTubeRegion can reuse a genuine countryCode already in the Premium body.
// It never consults Google, the OS, auto selection or an extra-family client.
func YouTubeRegion(body string) string {
	matches := youtubeCountry.FindAllStringSubmatch(body, -1)
	if len(matches) != len(regexp.MustCompile(`"countryCode"\s*:`).FindAllStringIndex(body, -1)) {
		return ""
	}
	return uniqueCountry(matches)
}

// Strict local metadata decoding: reject duplicate keys, trailing JSON and
// non-string leaves. This is not a new provider/schema inference subsystem.
func metadataJSONCountry(body []byte, path ...string) string {
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	var read func(int) (any, error)
	read = func(depth int) (any, error) {
		if depth > 64 {
			return nil, fmt.Errorf("metadata depth")
		}
		token, err := d.Token()
		if err != nil {
			return nil, err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return token, nil
		}
		if delim == '{' {
			object := map[string]any{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return nil, err
				}
				k, ok := key.(string)
				if !ok {
					return nil, fmt.Errorf("metadata key")
				}
				if _, exists := object[k]; exists {
					return nil, fmt.Errorf("metadata duplicate")
				}
				v, err := read(depth + 1)
				if err != nil {
					return nil, err
				}
				object[k] = v
			}
			end, err := d.Token()
			if err != nil || end != json.Delim('}') {
				return nil, fmt.Errorf("metadata object")
			}
			return object, nil
		}
		if delim == '[' {
			for d.More() {
				if _, err := read(depth + 1); err != nil {
					return nil, err
				}
			}
			end, err := d.Token()
			if err != nil || end != json.Delim(']') {
				return nil, fmt.Errorf("metadata array")
			}
			return []any{}, nil
		}
		return nil, fmt.Errorf("metadata delimiter")
	}
	node, err := read(0)
	if err != nil {
		return ""
	}
	if _, err = d.Token(); err != io.EOF {
		return ""
	}
	for _, key := range path {
		m, ok := node.(map[string]any)
		if !ok {
			return ""
		}
		node = m[key]
	}
	value, ok := node.(string)
	if !ok {
		return ""
	}
	return metadataCountry(value)
}

func metadataResponse(ctx context.Context, stack *netx.Stack, f netx.Family, req netx.Request) *netx.Response {
	resp, err := stack.Do(ctx, f, req)
	if err != nil {
		return nil
	}
	return validMetadataResponse(resp, req.URL)
}

func validMetadataResponse(resp *netx.Response, originURL string) *netx.Response {
	if !resp.OK() {
		return nil
	}
	final, err := url.Parse(resp.FinalURL)
	origin, _ := url.Parse(originURL)
	if err != nil || final.Scheme != "https" || final.User != nil || final.Host != origin.Host {
		return nil
	}
	lower := strings.ToLower(resp.Text())
	if strings.TrimSpace(lower) == "" || strings.EqualFold(strings.TrimSpace(resp.Header.Get("Cf-Mitigated")), "challenge") {
		return nil
	}
	for _, marker := range []string{"cf_chl_opt", "_cf_chl", "just a moment", "cf-browser-verification", "enable javascript and cookies to continue", "blocked by network security"} {
		if strings.Contains(lower, marker) {
			return nil
		}
	}
	return resp
}

// Reuse only successful, unchallenged official Premium response metadata;
// this helper does not change the Premium primary verdict.
func YouTubePrimaryRegion(resp *netx.Response) string {
	if validMetadataResponse(resp, "https://www.youtube.com/premium") == nil {
		return ""
	}
	return YouTubeRegion(resp.Text())
}

// RegionMetadata runs only after the primary result and its RTT are frozen,
// in that result's existing worker. One auxiliary budget (including Google's
// two sequential steps) <= --timeout, using the SAME direct/system stack.
// No errors/raw response are reflected into primary error/detail/exit status.
func RegionMetadata(ctx context.Context, stack *netx.Stack, f netx.Family, id string) string {
	var req netx.Request
	switch id {
	case "chatgpt_web":
		req = netx.Request{Method: "POST", URL: "https://ab.chatgpt.com/v1/initialize", Headers: map[string]string{"Statsig-Api-Key": "client-zUdXdSTygXJdzoE0sWTkP8GKTVsUMF2IRM7ShVO2JAG"}}
	case "tiktok_access":
		req.URL = "https://www.tiktok.com/api/v1/web-cookie-privacy/config?appId=1988"
	case "youtube_premium_access":
		req.URL = "https://www.youtube.com"
	case "spotify_signup":
		req.URL = "https://accounts.spotify.com/status"
	case "reddit_guest":
		req.URL = "https://www.reddit.com/svc/shreddit/reddit-chat"
	case "google_captcha":
		req.URL = "https://www.google.com"
	default:
		return ""
	}
	if ctx.Err() != nil || stack == nil {
		return ""
	}
	aux, cancel := context.WithTimeout(ctx, stack.Timeout())
	defer cancel()
	resp := metadataResponse(aux, stack, f, req)
	if resp == nil {
		return ""
	}
	switch id {
	case "chatgpt_web":
		return metadataJSONCountry(resp.Body, "derived_fields", "country")
	case "tiktok_access":
		return metadataJSONCountry(resp.Body, "body", "appProps", "region")
	case "youtube_premium_access":
		return YouTubeRegion(resp.Text())
	case "spotify_signup":
		return metadataJSONCountry(resp.Body, "geoLocationCountryCode")
	case "reddit_guest":
		return uniqueCountry(redditCountry.FindAllStringSubmatch(resp.Text(), -1))
	case "google_captcha":
		underscore := googleUnderscore.FindAllStringSubmatch(resp.Text(), -1)
		dash := googleDash.FindAllStringSubmatch(resp.Text(), -1)
		if len(underscore) > 0 {
			code := uniqueCountry(underscore)
			if code == "" {
				return ""
			}
			if len(dash) > 0 && metadataCountry(dash[len(dash)-1][1]) != code {
				return ""
			}
			return code
		}
		// Historical parser's LAST dash locale is the document's own, not alternate hreflangs.
		if len(dash) > 0 {
			return metadataCountry(dash[len(dash)-1][1])
		}
		play := metadataResponse(aux, stack, f, netx.Request{URL: "https://play.google.com/", Headers: map[string]string{"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Accept-Language": "en-US;q=0.9"}})
		if play == nil {
			return ""
		}
		matches := googlePlayCountry.FindAllStringSubmatch(play.Text(), -1)
		for i := range matches {
			matches[i][1] = countries.Code(matches[i][1])
		}
		return uniqueCountry(matches)
	}
	return ""
}
