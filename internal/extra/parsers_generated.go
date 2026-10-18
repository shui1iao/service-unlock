// Native translations; AGPL-3.0-only. RRC snapshot b6d4a6f9a87fc6eae6d3e62d0092ececcec8e844.
package extra

import "strings"

var singleParsers = map[string]func(response) verdict{
	"MediaUnlockTest_BilibiliHKMCTW": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "code")
		switch value {
		case "0":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		case "-10403":
			return no("source refusal: " + value)
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_BilibiliTW": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "code")
		switch value {
		case "0":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		case "-10403":
			return no("source refusal: " + value)
		}
		return unknown("source result field not recognized")
	},
	"GameTest_PCRJP": func(r response) verdict {
		switch r.Status {
		case 404:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"GameTest_UMAJP": func(r response) verdict {
		switch r.Status {
		case 404:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_Lemino": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_AnimeFesta": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_mora": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_Dazn": func(r response) verdict {
		if has(r.Text, `(?i)Security policy has been breached`) {
			return no("IP Banned by Dazn")
		}
		value := scalar(r.Text, "isAllowed")
		if value == "false" {
			return no("source isAllowed=false")
		}
		if value == "true" && r.Status >= 200 && r.Status < 300 {
			return yes(scalar(r.Text, "GeolocatedCountry"))
		}
		return unknown("DAZN location result not recognized")
	},
	"MediaUnlockTest_NowE": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "OTTAPI_ResponseCode")
		switch value {
		case "GEO_CHECK_FAIL":
			return no("source refusal: " + value)
		case "SUCCESS":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_ViuTV": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "responseCode")
		switch value {
		case "GEO_CHECK_FAIL":
			return no("source refusal: " + value)
		case "SUCCESS":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_unext": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "resultStatus")
		switch value {
		case "475":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		case "200":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		case "467":
			return no("source refusal: " + value)
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_HamiVideo": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "code")
		switch value {
		case "06001-106":
			return no("source refusal: " + value)
		case "06001-107":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_4GTV": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "Data")
		switch value {
		case "N":
			return no("source refusal: " + value)
		case "Y":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_Channel4": func(r response) verdict {
		switch r.Status {
		case 403:
			return no("HTTP source refusal")
		case 200:
			return yes("")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_DSTV": func(r response) verdict {
		switch r.Status {
		case 403:
			return no("HTTP source refusal")
		case 451:
			return no("HTTP source refusal")
		case 404:
			return yes("")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_encoreTVB": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		if !validJSON(r.Text) {
			return unknown("malformed JSON")
		}
		value := scalar(r.Text, "error_subcode")
		switch value {
		case "CLIENT_GEO":
			return no("source refusal: " + value)
		case "":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_Molotov": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "is_france")
		switch value {
		case "false":
			return no("source refusal: " + value)
		case "true":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_FOD": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := rx(r.Text, `<FLAG\sTYPE="([^"]+)`)
		switch value {
		case "true":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		case "false":
			return no("source refusal: " + value)
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_Fox": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_SpotvNow": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		if !validJSON(r.Text) {
			return unknown("malformed JSON")
		}
		value := scalar(r.Text, "error_subcode")
		switch value {
		case "CLIENT_GEO":
			return no("source refusal: " + value)
		case "":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_SKY_DE": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		if !validJSON(r.Text) {
			return unknown("malformed JSON")
		}
		value := scalar(r.Text, "error_subcode")
		switch value {
		case "CLIENT_GEO":
			return no("source refusal: " + value)
		case "":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_ZDF": func(r response) verdict {
		switch r.Status {
		case 404:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_videoland": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "isOnboardingGeoBlocked")
		switch value {
		case "false":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		case "true":
			return no("source refusal: " + value)
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_MoviStarPlus": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_RaiPlay": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		if strings.Contains(strings.ToLower(r.Text), "video_no_available") {
			return no("source video refusal")
		}
		value := rx(r.Text, `<geoprotection>([^<]+)`)
		switch value {
		case "Y":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		case "N":
			return no("source refusal: " + value)
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_TVBAnywhere": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "allow_in_this_country")
		switch value {
		case "true":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		case "false":
			return no("source refusal: " + value)
		}
		return unknown("source result field not recognized")
	},
	"GameTest_ProjectSekai": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"GameTest_KonosubaFD": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_Directv": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_Binge": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_OptusSports": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_KayoSports": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_SkyGONZ": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_ThreeNow": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_JCOM_ON_DEMAND": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_Popcornflix": func(r response) verdict {
		switch r.Status {
		case 403:
			return no("HTTP source refusal")
		case 200:
			return yes("")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_Philo": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "status")
		switch value {
		case "FAIL":
			return no("source refusal: " + value)
		case "SUCCESS":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_CWTV": func(r response) verdict {
		switch r.Status {
		case 403:
			return no("HTTP source refusal")
		case 451:
			return no("HTTP source refusal")
		case 200:
			return yes("")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_Wavve": func(r response) verdict {
		switch r.Status {
		case 403:
			return no("HTTP source refusal")
		case 200:
			return yes("")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_NaverTV": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "playable")
		switch value {
		case "NOT_COUNTRY_AVAILABLE":
			return no("source refusal: " + value)
		case "PLAYABLE":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_Watcha": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 451:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_KOCOWA": func(r response) verdict {
		switch r.Status {
		case 200:
			return yes("")
		case 403:
			return no("HTTP source refusal")
		}
		return unknown("unexpected HTTP status")
	},
	"MediaUnlockTest_NBCTV": func(r response) verdict {
		if strings.TrimSpace(r.Text) == "" {
			return unknown("empty response")
		}
		value := scalar(r.Text, "restricted")
		switch value {
		case "false":
			if r.Status >= 200 && r.Status < 300 {
				return yes("")
			}
			return unknown("unexpected HTTP")
		case "true":
			return no("source refusal: " + value)
		}
		return unknown("source result field not recognized")
	},
	"MediaUnlockTest_BBCiPLAYER": func(r response) verdict {
		if has(r.Text, "(?i)geolocation") {
			return no("source refusal wording")
		}
		if r.Status >= 200 && r.Status < 300 && has(r.Text, "(?i)vs-hls-push-uk") {
			return yes("")
		}
		return unknown("neither source positive nor refusal marker")
	},
	"MediaUnlockTest_ITVHUB": func(r response) verdict {
		if has(r.Text, "(?i)Outside Of Allowed Geographic") {
			return no("source refusal wording")
		}
		if r.Status >= 200 && r.Status < 300 && has(r.Text, "(?i)Playlist") {
			return yes("")
		}
		return unknown("neither source positive nor refusal marker")
	},
	"MediaUnlockTest_DMM": func(r response) verdict {
		if has(r.Text, "This page is not available in your area") {
			return no("source refusal wording")
		}
		if r.Status >= 200 && r.Status < 300 && has(r.Text, "\u6697\u53f7\u8cc7\u7523") {
			return yes("")
		}
		return unknown("neither source positive nor refusal marker")
	},
	"MediaUnlockTest_DMMTV": func(r response) verdict {
		if has(r.Text, "FOREIGN") {
			return no("source refusal wording")
		}
		if r.Status >= 200 && r.Status < 300 && has(r.Text, "UNAUTHORIZED") {
			return yes("")
		}
		return unknown("neither source positive nor refusal marker")
	},
	"MediaUnlockTest_NLZIET": func(r response) verdict {
		if has(r.Text, "(?i)CountryNotAllowed") {
			return no("source refusal wording")
		}
		if r.Status >= 200 && r.Status < 300 && has(r.Text, "(?i)streamSessionId") {
			return yes("")
		}
		return unknown("neither source positive nor refusal marker")
	},
	"MediaUnlockTest_AcornTV": func(r response) verdict {
		if has(r.Text, "(?i)not yet available in your|unavailable in your") {
			return no("source refusal wording")
		}
		if r.Status >= 200 && r.Status < 300 && has(r.Text, "(?i)signup.acorn.tv") {
			return yes("")
		}
		return unknown("neither source positive nor refusal marker")
	},
	"MediaUnlockTest_Crave": func(r response) verdict {
		if has(r.Text, "(?i)Geo Constraint Restrictions") {
			return no("source refusal wording")
		}
		if r.Status >= 200 && r.Status < 300 && has(r.Text, "(?i)video.9c9media.com") {
			return yes("")
		}
		return unknown("neither source positive nor refusal marker")
	},
	"MediaUnlockTest_Channel9": func(r response) verdict {
		if has(r.Text, "(?i)Geoblock") {
			return no("source refusal wording")
		}
		if r.Status >= 200 && r.Status < 300 && has(r.Text, "(?i)Log in to") {
			return yes("")
		}
		return unknown("neither source positive nor refusal marker")
	},
	"MediaUnlockTest_VideoMarket": func(r response) verdict {
		if has(r.Text, "OverseasAccess") {
			return no("source refusal wording")
		}
		if r.Status >= 200 && r.Status < 300 && has(r.Text, "292072") {
			return yes("")
		}
		return unknown("neither source positive nor refusal marker")
	},
	"WebTest_MetaAI": func(r response) verdict {
		if has(r.Text, "(?i)AbraGeoBlockedErrorRoot") {
			return no("source refusal wording")
		}
		if r.Status >= 200 && r.Status < 300 && has(r.Text, "(?i)KadabraRootContainer") {
			return yes("")
		}
		return unknown("neither source positive nor refusal marker")
	},
	"MediaUnlockTest_HuluJP": func(r response) verdict {
		if has(r.Final, "restrict") {
			return no("source refusal destination")
		}
		if r.Status == 403 {
			return no("HTTP403 source refusal")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected destination/HTTP")
	},
	"MediaUnlockTest_SlingTV": func(r response) verdict {
		if has(r.Final, "(?i)geo-block") {
			return no("source refusal destination")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected destination/HTTP")
	},
	"MediaUnlockTest_PlutoTV": func(r response) verdict {
		if has(r.Final, "plutotv-is-not-available") {
			return no("source refusal destination")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected destination/HTTP")
	},
	"MediaUnlockTest_PeacockTV": func(r response) verdict {
		if has(r.Final, "(?i)unavailable") {
			return no("source refusal destination")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected destination/HTTP")
	},
	"MediaUnlockTest_BritBox": func(r response) verdict {
		if has(r.Final, "locationnotsupported|locationnotvalidated|britbox.co.uk") {
			return no("source refusal destination")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected destination/HTTP")
	},
	"MediaUnlockTest_CanalPlus": func(r response) verdict {
		if has(r.Final, "(?i)other-country-blocking") {
			return no("source refusal destination")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected destination/HTTP")
	},
	"MediaUnlockTest_Sky_CH": func(r response) verdict {
		if has(r.Final, "(?i)out-of-country") {
			return no("source refusal destination")
		}
		if r.Status == 403 {
			return no("HTTP403 source refusal")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected destination/HTTP")
	},
	"MediaUnlockTest_Amediateka": func(r response) verdict {
		if has(r.Final, "(?i)unavailable") {
			return no("source refusal destination")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected destination/HTTP")
	},
	"MediaUnlockTest_Docplay": func(r response) verdict {
		if has(r.Final, "(?i)geoblocked") {
			return no("source refusal destination")
		}
		if r.Status == 403 {
			return no("HTTP403 source refusal")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected destination/HTTP")
	},
	"MediaUnlockTest_MyVideo": func(r response) verdict {
		if has(r.Final, "(?i)serviceAreaBlock") {
			return no("source refusal destination")
		}
		if r.Status == 403 {
			return no("HTTP403 source refusal")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected destination/HTTP")
	},
	"MediaUnlockTest_CoupangPlay": func(r response) verdict {
		if has(r.Final, "(?i)not-available") {
			return no("source refusal destination")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected destination/HTTP")
	},
	"MediaUnlockTest_NFLPlus": func(r response) verdict {
		if has(r.Final, "(?i)nfl-game-pass|gpi.nfl.com") {
			return no("source refusal destination")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected destination/HTTP")
	},
	"MediaUnlockTest_MyTVSuper": func(r response) verdict {
		v := scalar(r.Text, "country_code")
		if v == "" {
			return unknown("missing country field")
		}
		if r.Status < 200 || r.Status >= 300 {
			return unknown("unexpected HTTP")
		}
		if v == "HK" {
			return yes(v)
		}
		return no("source country refusal")
	},
	"MediaUnlockTest_KKTV": func(r response) verdict {
		v := scalar(r.Text, "country")
		if v == "" {
			return unknown("missing country field")
		}
		if r.Status < 200 || r.Status >= 300 {
			return unknown("unexpected HTTP")
		}
		if v == "TW" {
			return yes(v)
		}
		return no("source country refusal")
	},
	"MediaUnlockTest_CBCGem": func(r response) verdict {
		v := scalar(r.Text, "country")
		if v == "" {
			return unknown("missing country field")
		}
		if r.Status < 200 || r.Status >= 300 {
			return unknown("unexpected HTTP")
		}
		if v == "CA" {
			return yes(v)
		}
		return no("source country refusal")
	},
	"MediaUnlockTest_SBSonDemand": func(r response) verdict {
		v := scalar(r.Text, "country_code")
		if v == "" {
			return unknown("missing country field")
		}
		if r.Status < 200 || r.Status >= 300 {
			return unknown("unexpected HTTP")
		}
		if v == "AU" {
			return yes(v)
		}
		return no("source country refusal")
	},
	"MediaUnlockTest_NBATV": func(r response) verdict {
		if !has(r.Text, `(?i)<html|<!doctype`) {
			return unknown("malformed service HTML")
		}

		if r.Status != 200 || strings.TrimSpace(r.Text) == "" {
			return unknown("no normal response evidence")
		}
		if has(r.Text, "(?i)Service is not available in your region") {
			return no("source refusal wording")
		}
		return yes("")
	},
	"MediaUnlockTest_MegogoTV": func(r response) verdict {
		if !has(r.Text, `(?i)<html|<!doctype`) {
			return unknown("malformed service HTML")
		}

		if r.Status != 200 || strings.TrimSpace(r.Text) == "" {
			return unknown("no normal response evidence")
		}
		if has(r.Text, "(?i)VPN") {
			return no("source refusal wording")
		}
		return yes("")
	},
	"MediaUnlockTest_FXNOW": func(r response) verdict {
		return sourcePair(r, `(?i)is not accessible`, `(?i)FX Movies`)
	},
	"WebTest_Wikipedia_Editable": func(r response) verdict {
		if !has(r.Text, `(?i)Banned`) && !has(r.Text, `(?i)<textarea[^>]+(?:id|name)="wpTextbox1"`) {
			return unknown("edit form evidence absent")
		}

		if r.Status != 200 || strings.TrimSpace(r.Text) == "" {
			return unknown("no normal response evidence")
		}
		if has(r.Text, "(?i)Banned") {
			return no("source refusal wording")
		}
		return yes("")
	},
	"WebTest_EroGameSpace": func(r response) verdict {
		if !has(r.Text, `(?i)<html|<!doctype`) {
			return unknown("malformed service HTML")
		}

		if r.Status != 200 || strings.TrimSpace(r.Text) == "" {
			return unknown("no normal response evidence")
		}
		if has(r.Text, "(?i)18\u6b73") {
			return yes("")
		}
		return no("source access marker absent")
	},
	"MediaUnlockTest_Afreeca": func(r response) verdict {
		if !has(r.Text, `(?i)<html|<!doctype`) {
			return unknown("malformed service HTML")
		}

		if r.Status != 200 || strings.TrimSpace(r.Text) == "" {
			return unknown("no normal response evidence")
		}
		if has(r.Text, "document.location.href='https://vod.afreecatv.com'") {
			return yes("")
		}
		return no("source access marker absent")
	},
	"MediaUnlockTest_Catchplay": func(r response) verdict {
		v := scalar(r.Text, "code")
		if v == "" {
			return unknown("missing source result field")
		}
		if v == "0" && r.Status >= 200 && r.Status < 300 {
			return yes("")
		}
		if v == "100016" {
			return no("source refusal value")
		}
		return unknown("unrecognized source field")
	},
	"MediaUnlockTest_Channel5": func(r response) verdict {
		v := scalar(r.Text, "code")
		if v == "" {
			return unknown("missing source result field")
		}
		if v == "4003" && r.Status >= 200 && r.Status < 300 {
			return yes("")
		}
		if v == "3000" {
			return no("source refusal value")
		}
		return unknown("unrecognized source field")
	},
	"MediaUnlockTest_Channel10": func(r response) verdict {
		v := scalar(r.Text, "allow")
		if v == "" {
			return unknown("missing source result field")
		}
		if v == "true" && r.Status >= 200 && r.Status < 300 {
			return yes("")
		}
		if v == "false" {
			return no("source refusal value")
		}
		return unknown("unrecognized source field")
	},
	"MediaUnlockTest_SetantaSports": func(r response) verdict {
		v := scalar(r.Text, "outsideAllowedTerritories")
		if v == "" {
			return unknown("missing source result field")
		}
		if v == "false" && r.Status >= 200 && r.Status < 300 {
			return yes("")
		}
		if v == "true" {
			return no("source refusal value")
		}
		return unknown("unrecognized source field")
	},
	"MediaUnlockTest_BGlobalSEA": func(r response) verdict {
		v := scalar(r.Text, "code")
		if v == "0" && r.Status == 200 {
			return yes("")
		}
		if v == "10003003" || v == "10004001" {
			return no("source region refusal")
		}
		return unknown("unknown source error code")
	},
	"MediaUnlockTest_BGlobalTH": func(r response) verdict {
		v := scalar(r.Text, "code")
		if v == "0" && r.Status == 200 {
			return yes("")
		}
		if v == "10003003" || v == "10004001" {
			return no("source region refusal")
		}
		return unknown("unknown source error code")
	},
	"MediaUnlockTest_BGlobalID": func(r response) verdict {
		v := scalar(r.Text, "code")
		if v == "0" && r.Status == 200 {
			return yes("")
		}
		if v == "10003003" || v == "10004001" {
			return no("source region refusal")
		}
		return unknown("unknown source error code")
	},
	"MediaUnlockTest_BGlobalVN": func(r response) verdict {
		v := scalar(r.Text, "code")
		if v == "0" && r.Status == 200 {
			return yes("")
		}
		if v == "10003003" || v == "10004001" {
			return no("source region refusal")
		}
		return unknown("unknown source error code")
	},
	"MediaUnlockTest_7plus": func(r response) verdict {
		if r.Status == 200 {
			return yes("")
		}
		if r.Status == 403 {
			return no("source HTTP refusal")
		}
		return unknown("unexpected HTTP")
	},
	"MediaUnlockTest_TubiTV": func(r response) verdict {
		if has(r.Text, "(?i)not currently available in your area") {
			return no("source refusal wording")
		}
		if r.Status == 200 && has(r.Text, "(?i)manifest") {
			return yes("")
		}
		return unknown("no source conclusion marker")
	},
	"MediaUnlockTest_Shudder": func(r response) verdict {
		if has(r.Text, "(?i)not available|not yet available|403 ERROR") {
			return no("source refusal wording")
		}
		if r.Status == 200 && has(r.Text, "(?i)movies") {
			return yes("")
		}
		return unknown("no source conclusion marker")
	},
	"MediaUnlockTest_Tving": func(r response) verdict {
		if has(r.Text, "(?i)available in South Korea") {
			return no("source refusal wording")
		}
		if r.Status == 200 && has(r.Text, "vod_type") {
			return yes("")
		}
		return unknown("no source conclusion marker")
	},
	"MediaUnlockTest_Radiko": func(r response) verdict {
		if has(r.Text, "class=\"OUT\"") {
			return no("source refusal wording")
		}
		if r.Status == 200 && has(r.Text, "(?i)JAPAN") {
			return yes("")
		}
		return unknown("no source conclusion marker")
	},
	"MediaUnlockTest_AbemaTV": func(r response) verdict {
		v := scalar(r.Text, "isoCountryCode")
		if country(v) == "" {
			return unknown("missing country")
		}
		if r.Status != 200 {
			return unknown("unexpected HTTP")
		}
		if v == "JP" {
			return yes(v)
		}
		return limited(v, "Oversea Only")
	},
	"MediaUnlockTest_HBOMax": func(r response) verdict {
		v := rx(allText(r), `countryCode=([A-Z]{2})`)
		if v == "" {
			return unknown("country cookie absent")
		}
		if v == "US" || has(r.Text, `(?i)"url":"/`+strings.ToLower(v)+`/[a-z]{2}"`) {
			if r.Status == 200 {
				return yes(v)
			}
		}
		return no("source territory list excludes caller")
	},
	"MediaUnlockTest_Showmax": func(r response) verdict {
		v := rx(allText(r), `activeTerritory=([A-Za-z]{2})`)
		if v == "" {
			return unknown("served territory missing")
		}
		if r.Status == 200 {
			return yes(v)
		}
		return unknown("unexpected HTTP")
	},
	"MediaUnlockTest_ViuCom": func(r response) verdict {
		v := pathSegment(r.Final, 2)
		if v == "NO-SERVICE" {
			return no("no-service destination")
		}
		if country(v) == "" {
			return unknown("country path absent")
		}
		if r.Status == 200 {
			return yes(v)
		}
		return unknown("unexpected HTTP")
	},
	"MediaUnlockTest_ParamountPlus": func(r response) verdict {
		v := pathSegment(r.Final, 1)
		if v == "INTL" {
			return no("international refusal page")
		}
		if r.Status == 200 {
			if v == "" {
				v = "US"
			}
			return yes(v)
		}
		return unknown("unexpected HTTP")
	},
	"MediaUnlockTest_PrimeVideo": func(r response) verdict {
		if has(r.Text, `(?i)isServiceRestricted`) {
			return no("Service Not Available")
		}
		v := scalar(r.Text, "currentTerritory")
		if country(v) != "" && r.Status == 200 {
			return yes(v)
		}
		return unknown("neither service refusal nor territory")
	},
	"MediaUnlockTest_HotStar": func(r response) verdict {
		if r.Status == 403 || r.Status == 475 {
			return no("source HTTP region refusal")
		}
		v := rx(allText(r), `geo=([A-Z]+)`)
		if v == "" {
			return unknown("geo cookie missing")
		}
		if v == "US" {
			return no("Discontinued in the US")
		}
		u := strings.TrimPrefix(strings.TrimPrefix(r.Final, "https://www.hotstar.com/"), "https://hotstar.com/")
		if strings.EqualFold(v, strings.Trim(u, "/")) && r.Status == 200 {
			return yes(v)
		}
		return unknown("geo/site territory mismatch")
	},
	"MediaUnlockTest_LiTV": func(r response) verdict {
		if strings.Contains(r.Text, "AssetURLs") && r.Status == 200 {
			return yes("")
		}
		switch scalar(r.Text, "code") {
		case "42000075":
			if r.Status == 200 {
				return yes("")
			}
		case "42000087":
			return no("source region refusal")
		}
		return unknown("unknown LiTV response")
	},
	"MediaUnlockTest_FuboTV": func(r response) verdict {
		if has(r.Text, `(?i)NO_SERVICE_IN_COUNTRY`) || scalar(r.Text, "network_allowed") == "false" {
			return no("source network/country refusal")
		}
		if scalar(r.Text, "network_allowed") == "true" && r.Status == 200 {
			return yes(scalar(r.Text, "country_code2"))
		}
		return unknown("network_allowed not supplied")
	},
	"MediaUnlockTest_RakutenTV": func(r response) verdict {
		if has(r.Text, `(?i)VPN|forbidden|is not available`) {
			return no("source refusal wording")
		}
		v := scalar(r.Text, "iso3166_code")
		if country(v) != "" && has(r.Text, `(?i)streaming_drm_types`) && r.Status == 200 {
			return yes(v)
		}
		return unknown("country/streaming evidence missing")
	},
	"MediaUnlockTest_SkyGo": func(r response) verdict {
		if has(r.Text, `Sign in</h3>|skygoSignin`) && r.Status == 200 {
			return yes("")
		}
		if has(r.Text, `Access Denied.*You don.t have permission to access`) {
			return no("source permission refusal")
		}
		return unknown("neither source sign-in nor refusal")
	},
	"MediaUnlockTest_DirecTVGO": func(r response) verdict {
		if strings.Contains(r.Final, "proximamente") || r.Status == 403 {
			return no("source refusal")
		}
		if r.Status == 200 {
			return yes(pathSegment(r.Final, 1))
		}
		return unknown("unexpected HTTP")
	},
	"MediaUnlockTest_ABCiView": func(r response) verdict {
		if has(r.Text, `(?i)unavailable outside Australia`) {
			return no("source location refusal")
		}
		if scalar(r.Text, "playable") == "true" && r.Status == 200 {
			return yes("")
		}
		return unknown("playability not established")
	},
	"MediaUnlockTest_Telasa": func(r response) verdict {
		if has(r.Text, `(?i)IPLocationNotAllowed`) {
			return no("source IP location refusal")
		}
		if scalar(r.Text, "type") == "OK" && r.Status == 200 {
			return yes("")
		}
		return unknown("no OK or refusal result")
	},
	"MediaUnlockTest_musicjp": func(r response) verdict {
		if r.Status == 403 {
			return no("source HTTP403 refusal")
		}
		if has(r.Text, `(?i)checkIp`) && r.Status == 200 {
			return yes("")
		}
		return unknown("IP-check evidence absent")
	},
	"MediaUnlockTest_KBSDomestic": func(r response) verdict {
		v := rx(r.Text, `Domestic\\"\s*:\s*(false|true)`)
		if !strings.Contains(r.Text, "ipck") {
			return unknown("ipck configuration missing")
		}
		if v == "true" && r.Status == 200 {
			return yes("")
		}
		if v == "false" {
			return no("source domestic refusal")
		}
		return unknown("Domestic flag missing")
	},
	"MediaUnlockTest_Crackle": func(r response) verdict {
		v := r.Header.Get("X-Crackle-Region")
		if v == "" {
			v = rx(r.Text, `(?i)x-crackle-region:\s*([A-Z]+)`)
		}
		if v == "" {
			return unknown("region header missing")
		}
		if v == "US" && r.Status == 200 {
			return yes("")
		}
		return no("source US-only region criterion")
	},
	"MediaUnlockTest_SkyShowTime": func(r response) verdict {
		if r.Status == 403 || has(r.Text, `(?i)where-can-i-stream`) {
			return no("source region refusal")
		}
		v := rx(allText(r), `activeTerritory=([A-Z]+)`)
		if country(v) != "" && r.Status == 200 {
			return yes(v)
		}
		return unknown("served territory unavailable")
	},
	"MediaUnlockTest_MeWatch": func(r response) verdict {
		if scalar(r.Text, "code") != "" {
			return no("source API refusal code")
		}
		if has(r.Text, `(?i)Stream`) && r.Status == 200 {
			return yes("")
		}
		return unknown("no stream or refusal evidence")
	},
	"MediaUnlockTest_AMCPlus": func(r response) verdict {
		if r.Status == 403 || has(r.Final, `(?i)geographic-restriction`) {
			return no("source restriction page")
		}
		if r.Status == 200 {
			return yes("")
		}
		return unknown("unexpected HTTP")
	},
}
