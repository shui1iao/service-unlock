// Abema ip/check semantics from fixed RRC snapshot; explicit anonymous_ip
// refusal confirmed by JP2026-10-18 captured HTTP403 JSON.
// SPDX-License-Identifier: AGPL-3.0-only
package extra

import "encoding/json"

func classifyAbema(r response) verdict {
	if r.Status != 200 && r.Status != 403 {
		return unknown("unexpected Abema HTTP status")
	}
	var data struct {
		Message *string `json:"message"`
		Country *string `json:"isoCountryCode"`
	}
	if err := json.Unmarshal([]byte(r.Text), &data); err != nil {
		return unknown("malformed Abema JSON or field type")
	}
	// Only the captured, service-specific denial code is recognized. Missing
	// country, arbitrary text/other messages and empty objects remain Unknown.
	if data.Message != nil && *data.Message == "anonymous_ip" {
		return no("Abema refused anonymous_ip")
	}
	if r.Status != 200 {
		return unknown("Abema response has no recognized refusal")
	}
	if data.Country == nil || country(*data.Country) == "" {
		return unknown("missing country")
	}
	if *data.Country == "JP" {
		return yes(*data.Country)
	}
	return limited(*data.Country, "Oversea Only")
}
