// WOWOW playback/auth: structured evidence, not a substring in unrelated metadata.
// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"encoding/json"
	"strings"
)

func classifyWOWOWPlayback(r response) verdict {
	if r.Status != 200 && r.Status != 403 {
		return unknown("unexpected WOWOW playback HTTP status")
	}
	var data struct {
		Session *string `json:"playback_session_id"`
		Error   *struct {
			Code    *int    `json:"code"`
			Message *string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(r.Text), &data); err != nil {
		return unknown("malformed WOWOW playback JSON or field type")
	}
	if data.Error != nil {
		message := ""
		if data.Error.Message != nil {
			message = strings.ToLower(*data.Error.Message)
		}
		if (data.Error.Code != nil && *data.Error.Code == 2055) || strings.Contains(message, "vpn") || strings.Contains(message, "proxy") || strings.Contains(message, "outside japan") {
			return no("WOWOW playback authorization refused unsupported network/VPN (code2055 or explicit error reason)")
		}
		return unknown("unrecognized WOWOW playback error")
	}
	if r.Status == 200 && data.Session != nil && strings.TrimSpace(*data.Session) != "" {
		return yes("")
	}
	return unknown("WOWOW playback session evidence missing")
}
