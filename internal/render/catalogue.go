// Terminal-only purpose, capability labels and platform adjacency.
// MIT License; Copyright (c) 2026 Remnawave. See LICENSE.
package render

import "strings"

type purpose int

const (
	video purpose = iota
	music
	aiServices
	gaming
	community
	general
)

var purposeTitles = [...]string{
	"Video, live & sports", "Music & audio", "AI services",
	"Gaming", "Community", "General websites",
}

type presentation struct {
	purpose  purpose
	platform string
	label    string // Empty preserves the measurement's name; never written back.
}

// Source-qualified IDs keep capabilities such as access.gemini_access and ai.gemini separate.
// Only explicit platform keys are shared; no hostname/name inference or merging.
var corePresentation = map[string]map[string]presentation{
	"geo": {
		"google":         {general, "google", "Google (region)"},
		"google_captcha": {general, "google", ""},
		"youtube":        {video, "youtube", "YouTube (region)"},
		"youtube_music":  {music, "youtube", ""},
		"twitch":         {video, "twitch", "Twitch (region)"},
		"chatgpt":        {aiServices, "openai", "ChatGPT (region)"},
		"netflix":        {video, "netflix", "Netflix (Fast region)"},
		"spotify":        {music, "spotify", "Spotify (region)"},
		"spotify_signup": {music, "spotify", "Spotify (legacy signup endpoint)"},
		"deezer":         {music, "deezer", "Deezer (region)"},
		"reddit":         {community, "reddit", "Reddit (region)"},
		"reddit_guest":   {community, "reddit", "Reddit (guest access)"},
		"prime":          {video, "prime", "Amazon Prime Video (region)"},
		"apple":          {general, "apple", "Apple (region)"},
		"steam":          {gaming, "steam", "Steam (store region)"},
		"playstation":    {gaming, "playstation", "PlayStation (region)"},
		"tiktok":         {video, "tiktok", "TikTok (region)"},
		"ookla":          {general, "ookla", "Ookla Speedtest (region)"},
		"jetbrains":      {general, "jetbrains", "JetBrains (region)"},
		"bing":           {general, "bing", "Microsoft (Bing) (region)"},
	},
	"access": {
		"chatgpt_web":            {aiServices, "openai", "ChatGPT (Web)"},
		"youtube_premium_access": {video, "youtube", ""},
		"netflix_access":         {video, "netflix", "Netflix (catalogue)"},
		"claude_access":          {aiServices, "anthropic", "Claude (Web)"},
		"tiktok_access":          {video, "tiktok", "TikTok (Web)"},
		"gemini_access":          {aiServices, "gemini", "Gemini (Web)"},
	},
	"ai": {
		"openai":    {aiServices, "openai", "OpenAI (API)"},
		"anthropic": {aiServices, "anthropic", "Anthropic (API)"},
		"gemini":    {aiServices, "gemini", "Google Gemini (API)"},
		"deepseek":  {aiServices, "deepseek", "DeepSeek (API)"},
		"qwen_intl": {aiServices, "qwen", "Qwen (Singapore) (API)"},
		"qwen_cn":   {aiServices, "qwen", "Qwen (Beijing) (API)"},
		"moonshot":  {aiServices, "moonshot", "Kimi / Moonshot (API)"},
		"zhipu":     {aiServices, "zhipu", "Zhipu GLM (API)"},
	},
}

func presentationFor(source, id string) presentation {
	if p, ok := corePresentation[source][id]; ok {
		return p
	}
	// Unknown caller-supplied rows stay visible without impersonating a platform.
	p := presentation{purpose: general, platform: source + ":" + id}
	if source != "extra" {
		return p
	}
	switch id {
	case "WebTest_MetaAI":
		p.purpose, p.label = aiServices, "Meta AI (Web)"
	case "WebTest_Wikipedia_Editable":
		p.purpose = community
	case "WebTest_EroGameSpace":
		p.purpose = gaming
	case "MediaUnlockTest_mora", "MediaUnlockTest_Radiko", "MediaUnlockTest_DAM", "MediaUnlockTest_musicjp":
		p.purpose = music
	case "MediaUnlockTest_DMM":
		p.label = "DMM (financial site)"
	default:
		switch {
		case strings.HasPrefix(id, "GameTest_"):
			p.purpose = gaming
		case strings.HasPrefix(id, "MediaUnlockTest_"):
			p.purpose = video
		}
	}
	if id == "MediaUnlockTest_PrimeVideo" {
		p.platform = "prime"
	}
	return p
}

func terminalName(source, id, name string) string {
	if label := presentationFor(source, id).label; label != "" {
		return label
	}
	return name
}
