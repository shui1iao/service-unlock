package geo

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"serviceunlock/internal/netx"
)

// YouTubeCountryEvidence is internal evidence from an already scheduled YouTube
// response, not a Google Search country observation. A Problem rejects that
// response; a separately validated homepage response can supply the country.
type YouTubeCountryEvidence struct {
	Region, Source, Problem string
}

var youtubeOtherCountry = regexp.MustCompile(`"(INNERTUBE_CONTEXT_GL|contentRegion)"\s*:\s*"([^"]*)"`)
var youtubeOtherCountryKey = regexp.MustCompile(`"(?:INNERTUBE_CONTEXT_GL|contentRegion)"\s*:`)

func YouTubeCountry(resp *netx.Response, origin string) YouTubeCountryEvidence {
	unknown := func(problem string) YouTubeCountryEvidence {
		return YouTubeCountryEvidence{Problem: problem}
	}
	if resp == nil || validMetadataResponse(resp, origin) == nil {
		return unknown("untrusted YouTube response (HTTP, origin or challenge)")
	}
	final, _ := url.Parse(resp.FinalURL)
	body := resp.Text()
	if youtubeInterstitial(body) {
		return unknown("YouTube challenge, consent or error page")
	}
	if final.Path != "/" && final.Path != "" && final.Path != "/premium" && final.Path != "/premium/" {
		return unknown("unexpected YouTube destination")
	}
	region := YouTubeRegion(body)
	hasCountry := strings.Contains(body, `"countryCode"`)
	if hasCountry && region == "" {
		return unknown("malformed or conflicting YouTube countryCode")
	}
	// Literal match, never an unescaped regexp. Require a recognizable YouTube
	// document, so a bare error/consent page link is not routing evidence.
	cn := strings.Contains(body, "www.google.cn")
	otherMatches := youtubeOtherCountry.FindAllStringSubmatch(body, -1)
	if len(otherMatches) != len(youtubeOtherCountryKey.FindAllStringIndex(body, -1)) {
		return unknown("malformed YouTube region field")
	}
	var contextRegion string
	var hasClientCountry, hasContentRegion bool
	for _, match := range otherMatches {
		other := metadataCountry(match[2])
		if other == "" || contextRegion != "" && other != contextRegion {
			return unknown("malformed YouTube region field")
		}
		contextRegion = other
		switch match[1] {
		case "INNERTUBE_CONTEXT_GL":
			hasClientCountry = true
		case "contentRegion":
			hasContentRegion = true
		}
	}
	if contextRegion != "" && (region != "" && contextRegion != region || cn && contextRegion != "CN") {
		return unknown("malformed or conflicting YouTube region fields")
	}
	if cn {
		if !youtubeDocument(body) {
			return unknown("www.google.cn without a recognizable YouTube document")
		}
		if region != "" && region != "CN" {
			return unknown("www.google.cn conflicts with YouTube countryCode")
		}
		return YouTubeCountryEvidence{Region: "CN", Source: origin + " literal www.google.cn"}
	}
	if region != "" {
		return YouTubeCountryEvidence{Region: region, Source: origin + " countryCode"}
	}
	// Some normal Premium/home pages omit countryCode. In that case require
	// both country fields to exist and agree across the same document.
	if hasClientCountry && hasContentRegion && youtubeDocument(body) {
		return YouTubeCountryEvidence{Region: contextRegion, Source: origin + " matching INNERTUBE_CONTEXT_GL and contentRegion"}
	}
	return YouTubeCountryEvidence{}
}

func youtubeInterstitial(body string) bool {
	lower := strings.ToLower(body)
	for _, marker := range []string{"g-recaptcha", "<title>before you continue", "our systems have detected unusual traffic", "unusual traffic from", "sign in to confirm you're not a bot", "sign in to confirm you’re not a bot", "<title>error", "<title>oops", "<title>500", "<title>sorry", "<title>something went wrong"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func youtubeDocument(body string) bool {
	return strings.Contains(body, "ytInitialData") || strings.Contains(body, "ytcfg.set") ||
		strings.Contains(body, `"INNERTUBE_CONTEXT"`) || strings.Contains(body, `"INNERTUBE_API_KEY"`)
}

var youtubeMusicCountry = regexp.MustCompile(`"(INNERTUBE_CONTEXT_GL|contentRegion|countryCode)"\s*:\s*"([^"]*)"`)
var youtubeMusicCountryKey = regexp.MustCompile(`"(?:INNERTUBE_CONTEXT_GL|contentRegion|countryCode)"\s*:`)

// Read only the Music product's own response. Without a direct countryCode,
// require corroboration between its client country and content region fields.
func youtubeMusicRegion(resp *netx.Response, home string, allowedStatus ...int) string {
	if validMetadataResponse(resp, home, allowedStatus...) == nil {
		return ""
	}
	final, _ := url.Parse(resp.FinalURL)
	body := resp.Text()
	if final.Path != "/" && final.Path != "" || youtubeInterstitial(body) || !youtubeDocument(body) {
		return ""
	}
	matches := youtubeMusicCountry.FindAllStringSubmatch(body, -1)
	if len(matches) != len(youtubeMusicCountryKey.FindAllStringIndex(body, -1)) {
		return ""
	}
	var region string
	var direct, client, content bool
	for _, match := range matches {
		code := metadataCountry(match[2])
		if code == "" || region != "" && code != region {
			return ""
		}
		region = code
		switch match[1] {
		case "countryCode":
			direct = true
		case "INNERTUBE_CONTEXT_GL":
			client = true
		case "contentRegion":
			content = true
		}
	}
	if direct || client && content {
		return region
	}
	return ""
}

// YouTubeMetadata uses the existing optional homepage request, never a second
// Google-specific probe. The same response supplies both consumers.
func YouTubeMetadata(ctx context.Context, stack *netx.Stack, f netx.Family) YouTubeCountryEvidence {
	if ctx.Err() != nil || stack == nil {
		return YouTubeCountryEvidence{Problem: "YouTube metadata unavailable"}
	}
	aux, cancel := context.WithTimeout(ctx, stack.Timeout())
	defer cancel()
	const origin = "https://www.youtube.com"
	resp := metadataResponse(aux, stack, f, netx.Request{URL: origin})
	return YouTubeCountry(resp, origin)
}

// ApplyYouTubeCountry merges into the actual observation before either JSON or
// terminal rendering. KindBlocked keeps its existing raw yes => display No.
func ApplyYouTubeCountry(results []Result, evidence YouTubeCountryEvidence) {
	for i := range results {
		r := &results[i]
		if r.Check.ID != "google_captcha" {
			continue
		}
		// No Google locale/Play value is allowed to survive this merge.
		r.Region = ""
		if evidence.Problem != "" || metadataCountry(evidence.Region) == "" || evidence.Source == "" {
			detail := "YouTube region unknown; no country inference"
			if evidence.Problem != "" {
				detail += ": " + evidence.Problem
			}
			r.Detail += "; " + detail
			continue
		}
		r.Region = evidence.Region
		r.Detail += "; region from YouTube: " + evidence.Source + " (not Google Search geolocation)"
		if evidence.Region == "CN" {
			r.Detail = "Google No from YouTube CN evidence; region from YouTube: " + evidence.Source + " (not Google Search geolocation)"
			if r.Err != nil {
				r.Detail += "; Search request failed: " + r.Err.Error()
			} else if r.Value == "yes" {
				r.Detail += "; Search captcha/block marker also observed"
			} else {
				r.Detail += "; Search captcha/block marker not observed"
			}
			r.Value, r.Err = "yes", nil
		}
	}
}
