package render

import (
	"errors"
	"testing"
	"time"

	"serviceunlock/internal/ai"
	"serviceunlock/internal/geo"
)

func completeMock(t *testing.T) Report {
	r := mockReport(t)
	for _, c := range geo.ServiceChecks() {
		value := "US"
		if c.Kind == geo.KindAvailability {
			value = "yes"
		}
		if c.Kind == geo.KindBlocked {
			value = "no"
		}
		res := geo.Result{Check: c, Value: value, Detail: "offline observation; not an unlock verdict", RTT: 20 * time.Millisecond}
		if c.ID == "apple" {
			res.Value = ""
			res.Err = errors.New("mock timeout; result unknown")
		}
		r.Services = append(r.Services, res)
	}
	for _, c := range ai.Checks() {
		res := ai.Result{Check: c, State: ai.StateReachable, Status: 401, Detail: "answered, credentials not supplied", RTT: 20 * time.Millisecond}
		if c.ID == "anthropic" {
			res.State = ai.StateBlocked
			res.Status = 403
			res.Detail = "the API refused this region"
		}
		if c.ID == "gemini" {
			res.Status = 403
		}
		if c.ID == "zhipu" {
			res.State = ai.StateError
			res.Status = 429
			res.Detail = "unexpected HTTP 429"
		}
		r.AI = append(r.AI, res)
	}
	return r
}
