// RRC anonymous multi-request flows translated to Go. No script/JS execution.
// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:embed client-data.json
var clientDataBytes []byte
var clientData = func() map[string]string {
	var m map[string]string
	if e := json.Unmarshal(clientDataBytes, &m); e != nil {
		panic(e)
	}
	return m
}()

//go:embed flow-json-steps.json
var flowJSONBytes []byte
var flowJSONSteps = func() map[string][]bool {
	var m map[string][]bool
	if e := json.Unmarshal(flowJSONBytes, &m); e != nil {
		panic(e)
	}
	return m
}()

type flow struct {
	ctx context.Context
	c   *Client
	id  string
	v   map[string]string
	err error
}

func (f *flow) get(index int) response {
	if f.err != nil {
		return response{}
	}
	r, e := f.c.step(f.ctx, f.id, index, f.v)
	if flags := flowJSONSteps[f.id]; e == nil && index < len(flags) && flags[index] && !validJSON(r.Text) {
		e = fmt.Errorf("malformed JSON at source flow step %d", index)
	}
	// WOWOW's two middle requests are HTML documents, not unchecked text/JSON.
	if e == nil && f.id == "MediaUnlockTest_wowow" && (index == 1 || index == 2) {
		lower := strings.ToLower(strings.TrimSpace(r.Text))
		htmlStart := strings.HasPrefix(lower, "<!doctype html") || strings.HasPrefix(lower, "<html")
		if r.Status != 200 || !htmlStart || !has(lower, `<html(?:\s|>)`) || !strings.Contains(lower, "</html>") {
			e = fmt.Errorf("malformed HTML at WOWOW flow step %d", index)
		}
	}
	if e != nil {
		f.err = e
	}
	return r
}
func (f *flow) need(key, value string) bool {
	if value == "" {
		f.err = fmt.Errorf("missing source flow field %s", key)
		return false
	}
	f.v[key] = value
	return true
}
func (f *flow) failure() verdict {
	if f.err != nil {
		return unknown(f.err.Error())
	}
	return unknown("unrecognized source response")
}
func sourcePair(r response, negative, positive string) verdict {
	if has(r.Text, negative) {
		return no("source refusal marker")
	}
	if r.Status >= 200 && r.Status < 300 && has(r.Text, positive) {
		return yes("")
	}
	return unknown("neither source refusal nor positive marker")
}
func sourceErrorSubcode(r response) verdict {
	if strings.TrimSpace(r.Text) == "" || !validJSON(r.Text) {
		return unknown("malformed source playback JSON")
	}
	if scalar(r.Text, "error_subcode") == "CLIENT_GEO" {
		return no("source CLIENT_GEO")
	}
	if scalar(r.Text, "error_subcode") == "" && r.Status >= 200 && r.Status < 300 {
		return yes("")
	}
	return unknown("unrecognized playback result")
}
func runFlow(ctx context.Context, c *Client, id string) verdict {
	f := &flow{ctx: ctx, c: c, id: id, v: variables()}
	switch id {
	case "MediaUnlockTest_BahamutAnime":
		f.c = c.withCookies()
		r := f.get(0)
		if !f.need("tempdeviceid", scalar(r.Text, "deviceid")) {
			return f.failure()
		}
		f.v["sn"] = "37783"
		token := f.get(1)
		home := f.get(2)
		if f.err != nil {
			return f.failure()
		}
		if !validJSON(token.Text) || strings.TrimSpace(home.Text) == "" {
			return unknown("malformed token/home response")
		}
		if !strings.Contains(token.Text, "animeSn") {
			return no("title withheld")
		}
		region := rx(home.Text, `data-geo="([^"]+)`)
		if region == "" {
			return unknown("served region missing")
		}
		return yes(region)
	case "MediaUnlockTest_DisneyPlus":
		device := f.get(0)
		if f.err != nil {
			return f.failure()
		}
		if has(device.Text, `(?i)403 ERROR`) {
			return no("source device refusal")
		}
		if !f.need("assertion", scalar(device.Text, "assertion")) {
			return f.failure()
		}
		f.v["disneyCookie"] = strings.ReplaceAll(clientData["disney_form"], "DISNEYASSERTION", url.QueryEscape(f.v["assertion"]))
		token := f.get(1)
		if f.err != nil {
			return f.failure()
		}
		if has(token.Text, `(?i)forbidden-location|403 ERROR`) {
			return no("source token refused location")
		}
		if !f.need("refreshToken", scalar(token.Text, "refresh_token")) {
			return f.failure()
		}
		f.v["disneyContent"] = strings.ReplaceAll(clientData["disney_graph"], "ILOVEDISNEY", f.v["refreshToken"])
		graph := f.get(2)
		home := f.get(3)
		if f.err != nil {
			return f.failure()
		}
		region := scalar(graph.Text, "countryCode")
		supported := scalar(graph.Text, "inSupportedLocation")
		if country(region) == "" {
			return unknown("countryCode missing")
		}
		if region == "JP" {
			return yes(region)
		}
		if has(home.Final, `preview|unavailable`) {
			return no("source unavailable destination")
		}
		if supported == "false" {
			return limited(region, "Available Soon")
		}
		if supported == "true" {
			return yes(region)
		}
		return unknown("inSupportedLocation missing")
	case "MediaUnlockTest_ESPNPlus":
		f.v["espnCookies"] = clientData["espn_form"]
		token := f.get(0)
		if f.err != nil {
			return f.failure()
		}
		if has(token.Text, `forbidden-location|403 ERROR`) {
			return no("source token refused location")
		}
		if !f.need("refreshToken", scalar(token.Text, "refresh_token")) {
			return f.failure()
		}
		f.v["espnContent"] = strings.ReplaceAll(clientData["espn_graph"], "ILOVESTAR", f.v["refreshToken"])
		graph := f.get(1)
		if f.err != nil {
			return f.failure()
		}
		region := scalar(graph.Text, "countryCode")
		supported := scalar(graph.Text, "inSupportedLocation")
		if country(region) == "" || supported == "" {
			return unknown("source supported-location data missing")
		}
		if region == "US" && supported == "true" {
			return yes("")
		}
		return no("source supported-location refusal")
	case "MediaUnlockTest_LineTVTW":
		home := f.get(0)
		js := rx(home.Text, `src="([^"]*/main-[a-z0-9]{8}[^"]*)`)
		if !f.need("mainJsUrl", js) {
			return f.failure()
		}
		// Resolve relative static asset paths but never execute the returned script.
		base, _ := url.Parse(f.v["mainJsUrl"])
		if !base.IsAbs() {
			u, _ := url.Parse("https://www.linetv.tw/")
			f.v["mainJsUrl"] = u.ResolveReference(base).String()
		}
		script := f.get(1)
		if !f.need("appId", rx(script.Text, `appId:"([^"]+)`)) {
			return f.failure()
		}
		result := f.get(2)
		if f.err != nil {
			return f.failure()
		}
		v := scalar(result.Text, "countryCode")
		if v == "" {
			return unknown("countryCode absent")
		}
		if v == "228" {
			return yes("")
		}
		return no("source country refusal")
	case "MediaUnlockTest_Niconico":
		video := f.get(0)
		home := f.get(1)
		if f.err != nil {
			return f.failure()
		}
		// Upstream selects an official-member-free programme containing 話 and not ON_AIR/playerProgram.
		decoded := html.UnescapeString(home.Text)
		var live string
		for _, part := range strings.Split(decoded, `"isOfficialChannelMemberFree":false`) {
			if strings.Contains(part, `"isOfficialChannelMemberFree":true`) || strings.Contains(part, "playerProgram") || strings.Contains(part, `"ON_AIR"`) || !strings.Contains(part, "話") {
				continue
			}
			if live = rx(part, `"id":"(lv[0-9]+)`); live != "" {
				break
			}
		}
		if !f.need("liveID", live) {
			return f.failure()
		}
		watch := f.get(2)
		if f.err != nil {
			return f.failure()
		}
		if strings.Contains(video.Text, "同じ地域") {
			return no("source video region refusal")
		}
		if strings.Contains(watch.Text, "notAllowedCountry") {
			return no("Official Live Unavailable")
		}
		if video.Status == 200 && watch.Status == 200 && video.Text != "" && watch.Text != "" {
			return yes("")
		}
		return f.failure()
	case "MediaUnlockTest_Joyn":
		auth := f.get(0)
		if !f.need("auth", scalar(auth.Text, "access_token")) {
			return f.failure()
		}
		r := f.get(1)
		if f.err != nil {
			return f.failure()
		}
		return sourcePair(r, `(?i)ENT_AssetNotAvailableInCountry`, `(?i)entitlement_token`)
	case "MediaUnlockTest_EPIX":
		token := f.get(0)
		if f.err != nil {
			return f.failure()
		}
		if has(token.Text, `(?i)403 ERROR`) {
			return no("source session refusal")
		}
		if !f.need("epixToken", scalar(token.Text, "session_token")) {
			return f.failure()
		}
		r := f.get(1)
		if f.err != nil {
			return f.failure()
		}
		return sourcePair(r, `(?i)MGM\+ is only available in the United States`, `(?i)StartBilling`)
	case "MediaUnlockTest_NPO_Start_Plus":
		token := f.get(0)
		if !f.need("token", scalar(token.Text, "token")) {
			return f.failure()
		}
		r := f.get(1)
		if f.err != nil {
			return f.failure()
		}
		if r.Status == 200 {
			return yes("")
		}
		if r.Status == 403 {
			return no("source stream-link refusal")
		}
		return f.failure()
	case "MediaUnlockTest_Starz":
		token := f.get(0)
		if !f.need("authorization", strings.TrimSpace(token.Text)) {
			return f.failure()
		}
		r := f.get(1)
		if f.err != nil {
			return f.failure()
		}
		access, country, proxy := scalar(r.Text, "isAllowedAccess"), scalar(r.Text, "isAllowedCountry"), scalar(r.Text, "isKnownProxy")
		if access == "" || country == "" || proxy == "" {
			return unknown("source geolocation flags absent")
		}
		if access == "true" && country == "true" && proxy == "false" {
			return yes("")
		}
		if access == "false" || proxy == "false" {
			return no("source access/proxy refusal")
		}
		return f.failure()
	case "MediaUnlockTest_DiscoveryPlus":
		bootstrap := f.get(0)
		if !f.need("baseApiUrl", scalar(bootstrap.Text, "baseApiUrl")) || !f.need("realm", scalar(bootstrap.Text, "realm")) {
			return f.failure()
		}
		if f.v["realm"] == "dplusapac" {
			return no("Not Yet Available in Asia Pacific")
		}
		token := f.get(1)
		if !f.need("token", scalar(token.Text, "token")) {
			return f.failure()
		}
		r := f.get(2)
		if f.err != nil {
			return f.failure()
		}
		v := sourcePair(r, `(?i)is unavailable in your|not yet available`, `(?i)relationships`)
		if v.status == "available" {
			v.region = country(scalar(r.Text, "mainTerritoryCode"))
		}
		return v
	case "MediaUnlockTest_TLCGO":
		token := f.get(0)
		if !f.need("token", scalar(token.Text, "token")) {
			return f.failure()
		}
		r := f.get(1)
		if f.err != nil {
			return f.failure()
		}
		v := sourcePair(r, `(?i)is not yet available`, `(?i)Episodes`)
		if v.status == "available" {
			v.region = country(scalar(r.Text, "mainTerritoryCode"))
		}
		return v
	case "MediaUnlockTest_AETV":
		first := f.get(0)
		if f.err != nil {
			return f.failure()
		}
		if has(first.Text, `(?i)GeoLocationBlocked`) {
			return no("source GeoLocationBlocked")
		}
		second := f.get(1)
		if f.err != nil {
			return f.failure()
		}
		region := rx(allText(second), `AETN-Country-Code=([A-Z]+)`)
		if region == "" {
			return unknown("country cookie missing")
		}
		if region == "US" || region == "CA" {
			return yes("")
		}
		return no("source US/CA territory criterion")
	case "GameTest_MathsSpot":
		home := f.get(0)
		if f.err != nil {
			return f.failure()
		}
		if has(home.Text, `(?i)FailureServiceNotInRegion`) {
			return no("source region failure")
		}
		path := rx(home.Text, `fetch\("([^\"]*reportEvent)`)
		path = strings.TrimPrefix(strings.TrimSuffix(path, "/reportEvent"), "/")
		if !f.need("apiPath", path) || !f.need("nggFeVersion", scalar(home.Text, "NEXT_PUBLIC_FE_VERSION")) || !f.need("region", scalar(home.Text, "countryCode")) {
			return f.failure()
		}
		for _, key := range []string{"fakeUAId", "fakeSessId", "fakeFesessId", "fakeVisitId"} {
			f.v[key] = randomID()[:21]
		}
		r := f.get(1)
		if f.err != nil {
			return f.failure()
		}
		switch scalar(r.Text, "status") {
		case "Success":
			return yes(f.v["region"])
		case "FailureServiceNotInRegion", "FailureProxyUserLimitExceeded":
			return no("source session/proxy refusal")
		}
		return f.failure()
	case "MediaUnlockTest_SonyLiv":
		home := f.get(0)
		if f.err != nil {
			return f.failure()
		}
		if strings.Contains(home.Text, "geolocation_notsupported") {
			return no("source geolocation unsupported")
		}
		token := rx(home.Text, `securityToken.*?resultObj"?\s*:\s*"([^"]+)`)
		if token == "" {
			token = rx(home.Text, `securityToken"?\s*:\s*"([^"]+)`)
		}
		if !f.need("jwtToken", token) {
			return f.failure()
		}
		country := f.get(1)
		if !f.need("region", scalar(country.Text, "country_code")) {
			return f.failure()
		}
		r := f.get(2)
		if f.err != nil {
			return f.failure()
		}
		switch scalar(r.Text, "resultCode") {
		case "OK":
			return yes(f.v["region"])
		case "KO":
			return no("source content prefetch refusal")
		}
		return f.failure()
	case "MediaUnlockTest_RakutenTVJP":
		f.v["contentId"] = "387878"
		r := f.get(0)
		if f.err != nil {
			return f.failure()
		}
		if has(r.Text, `(?i)IS_FOREIGN`) {
			return no("source IS_FOREIGN")
		}
		if !has(r.Text, `(?i)in_vod`) {
			return unknown("source playback data absent")
		}
		geo := f.get(1)
		if f.err != nil {
			return f.failure()
		}
		switch scalar(geo.Text, "is_domestic") {
		case "true":
			return yes("")
		case "false":
			return no("NBA Unavailable")
		}
		return f.failure()
	case "MediaUnlockTest_MaoriTV":
		home := f.get(0)
		if !f.need("indexJsPath", rx(home.Text, `src="(/assets/index-[a-z0-9]{8}[^"]+)`)) {
			return f.failure()
		}
		index := f.get(1)
		player := rx(index.Text, `(players\.brightcove\.net/[0-9]{13}/\w{9}_default/index.min.js)`)
		if !f.need("playerJsUrl", player) || !f.need("accountId", rx(player, `players\.brightcove\.net/([0-9]{13})`)) {
			return f.failure()
		}
		script := f.get(2)
		if !f.need("policyKey", rx(script.Text, `policyKey\s*:\s*"(BCpk[^"]+)`)) {
			return f.failure()
		}
		f.v["bcChannelId"] = "6278939271001"
		r := f.get(3)
		if f.err != nil {
			return f.failure()
		}
		return sourceErrorSubcode(r)
	case "MediaUnlockTest_TVer":
		token := f.get(0)
		if !f.need("platformUid", scalar(token.Text, "platform_uid")) || !f.need("platformToken", scalar(token.Text, "platform_token")) {
			return f.failure()
		}
		home := f.get(1)
		episode := rx(home.Text, `variety\.catchup\.recomend.*?"id"\s*:\s*"([a-z0-9]{10})`)
		if !f.need("episodeId", episode) {
			return f.failure()
		}
		meta := f.get(2)
		if !f.need("accountID", scalar(meta.Text, "accountID")) || !f.need("playerID", scalar(meta.Text, "playerID")) {
			return f.failure()
		}
		f.v["videoID"] = scalar(meta.Text, "videoID")
		f.v["videoRefID"] = scalar(meta.Text, "videoRefID")
		script := f.get(3)
		if !f.need("policyKey", rx(script.Text, `policyKey:"([^"]+)`)) {
			return f.failure()
		}
		step := 5
		if f.v["videoRefID"] == "" {
			step = 4
			if !f.need("deliveryConfigId", rx(script.Text, `deliveryConfigId:"([^"]+)`)) || f.v["videoID"] == "" {
				return f.failure()
			}
		}
		r := f.get(step)
		if f.err != nil {
			return f.failure()
		}
		return sourceErrorSubcode(r)
	case "MediaUnlockTest_wowow":
		f.v["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
		home := f.get(0)
		matches := regexp.MustCompile(`"link"\s*:\s*"([^"]*drama/original[^"]*)`).FindAllStringSubmatch(home.Text, 4)
		if len(matches) == 0 {
			return unknown("drama lineup links missing")
		}
		wod := ""
		for _, m := range matches {
			f.v["playUrl"] = m[1]
			page := f.get(1)
			if f.err != nil {
				return f.failure()
			}
			wod = rx(page.Text, `"(https://wod\.wowow\.co\.jp/content/[^"]+)`)
			if wod != "" {
				break
			}
		}
		if !f.need("wodUrl", wod) {
			return f.failure()
		}
		page := f.get(2)
		if !f.need("metaId", rx(page.Text, `https://wod\.wowow\.co\.jp/watch/([0-9]+)`)) {
			return f.failure()
		}
		f.v["vUid"] = randomID()
		r := f.get(3)
		if f.err != nil {
			return f.failure()
		}
		return classifyWOWOWPlayback(r)
	}
	return unknown("native implementation missing: " + id)
}
