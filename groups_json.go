package main

import (
	"math"
	"serviceunlock/internal/ai"
	"time"
)

type serviceItem struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Status    string  `json:"status"`
	Value     *string `json:"value"`
	Detail    string  `json:"detail"`
	Error     *string `json:"error"`
	CheckedAt string  `json:"checked_at"`
	ElapsedMS float64 `json:"elapsed_ms"`
}
type endpointItem struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Vendor     string  `json:"vendor"`
	Status     string  `json:"status"`
	HTTPStatus int     `json:"http_status"`
	Detail     string  `json:"detail"`
	Error      *string `json:"error"`
	CheckedAt  string  `json:"checked_at"`
	ElapsedMS  float64 `json:"elapsed_ms"`
}

func milliseconds(d time.Duration) float64 {
	return math.Round(float64(d)/float64(time.Millisecond)*1000) / 1000
}
func makeCompleteReport(m measurements, timeout float64, workers int) report {
	r := makeReport(m.Access, timeout, workers)
	r.Services = make([]serviceItem, 0, len(m.Services))
	r.AIEndpoints = make([]endpointItem, 0, len(m.AI))
	for _, res := range m.Services {
		out := serviceItem{ID: res.Check.ID, Name: res.Check.Name, Type: res.Check.Kind.String(), Status: "observed", Detail: res.Detail, CheckedAt: utc(res.CheckedAt), ElapsedMS: milliseconds(res.RTT)}
		if res.Err != nil {
			out.Status = "error"
			message := res.Err.Error()
			out.Error = &message
		} else {
			value := res.Value
			out.Value = &value
		}
		r.Services = append(r.Services, out)
	}
	for _, res := range m.AI {
		out := endpointItem{ID: res.Check.ID, Name: res.Check.Name, Vendor: res.Check.Vendor, Status: res.State.String(), HTTPStatus: res.Status, Detail: res.Detail, CheckedAt: utc(res.CheckedAt), ElapsedMS: milliseconds(res.RTT)}
		if res.State == ai.StateError {
			message := res.Detail
			if res.Err != nil {
				message = res.Err.Error()
			}
			if message == "" {
				message = "could not reach a conclusion"
			}
			out.Error = &message
		}
		r.AIEndpoints = append(r.AIEndpoints, out)
	}
	return r
}
