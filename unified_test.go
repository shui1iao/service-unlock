package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/extra"
	"serviceunlock/internal/geo"
	"strings"
	"testing"
	"time"
)

func unifiedFixture(region string, unknown bool) measurements {
	when := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	m := measurements{}
	for _, c := range access.Checks() {
		s := access.StateAvailable
		detail := "OFFLINE MOCK"
		if unknown {
			s = access.StateError
		}
		m.Access = append(m.Access, access.Result{Check: c, State: s, Detail: detail, CheckedAt: when, RTT: time.Millisecond})
	}
	for _, c := range geo.ServiceChecks() {
		value := "US"
		if c.Kind == geo.KindAvailability {
			value = "yes"
		}
		if c.Kind == geo.KindBlocked {
			value = "no"
		}
		r := geo.Result{Check: c, Value: value, Detail: "OFFLINE MOCK", CheckedAt: when, RTT: time.Millisecond}
		if unknown {
			r.Err = errors.New("OFFLINE MOCK unknown")
			r.Value = ""
		}
		m.Services = append(m.Services, r)
	}
	for _, c := range ai.Checks() {
		r := ai.Result{Check: c, State: ai.StateReachable, Status: 401, Detail: "OFFLINE MOCK", CheckedAt: when, RTT: time.Millisecond}
		if unknown {
			r.State = ai.StateError
			r.Err = errors.New("OFFLINE MOCK unknown")
		}
		m.AI = append(m.AI, r)
	}
	checks, _ := extra.Selected(region)
	m.Selection = makeSelection(region, "4", 4, checks)
	for i, c := range checks {
		scope := "global"
		global := false
		for _, g := range c.Groups {
			if g == "global" {
				global = true
			}
		}
		if !global {
			scope = region
		}
		r := extra.Result{ID: c.ID, Name: c.Name, SourceFunction: c.SourceFunction, SourceLine: c.SourceLine, Scope: scope, Family: 4, Status: "available", Detail: "OFFLINE MOCK - NOT REAL RESULTS", CheckedAt: utc(when), ElapsedMS: 1}
		if i%4 == 1 {
			r.Status = "blocked"
		}
		if i%4 == 2 {
			r.Status = "restricted"
		}
		if c.Safety != "" || unknown {
			r.Status = "error"
			s := "OFFLINE MOCK inconclusive"
			if c.Safety != "" {
				s = "unsupported for safety: " + c.Safety
			}
			r.Error = &s
		}
		m.Extra = append(m.Extra, r)
	}
	return m
}
func TestUnifiedSelectionSameMeasurementEnvelope(t *testing.T) {
	counts := map[string]int{"global": 41, "tw": 52, "hk": 49, "jp": 67, "na": 78, "sa": 44, "eu": 71, "oc": 60, "kr": 49, "sea": 52, "af": 44}
	for _, region := range extra.Regions() {
		for _, unknown := range []bool{false, true} {
			t.Run(region+map[bool]string{false: "-mixed", true: "-unknown"}[unknown], func(t *testing.T) {
				m := unifiedFixture(region, unknown)
				calls := 0
				var out, errout bytes.Buffer
				code := runCLI(context.Background(), []string{"--machine", "--region", region, "--extra-family", "4"}, &out, &errout, func(ctx context.Context, d time.Duration, w int) (measurements, error) {
					calls++
					opts := ctx.Value(extraOptionsKey{}).(extraOptions)
					if opts.Region != region || opts.Family != "4" {
						t.Fatal(opts)
					}
					return m, nil
				})
				if code != 1 || calls != 1 || errout.Len() != 0 {
					t.Fatal(code, calls, errout.String())
				}
				var e struct {
					Format   string `json:"format"`
					Report   report `json:"report"`
					Terminal string `json:"terminal"`
				}
				if er := json.Unmarshal(out.Bytes(), &e); er != nil {
					t.Fatal(er)
				}
				if e.Format != "guko-unified-v1" || len(e.Report.Results) != 6 || len(e.Report.Services) != 21 || len(e.Report.AIEndpoints) != 8 || len(e.Report.ExtraChecks)+35 != counts[region] || len(e.Report.Selection.ExpectedIDs) != len(m.Extra) {
					t.Fatal("scope mismatch")
				}
				if !strings.Contains(e.Terminal, "\x1b[") || strings.Count(e.Terminal, "╭") != map[bool]int{true: 4, false: 5}[region == "global"] {
					t.Fatal("not a complete unified ANSI report")
				}
				if !reflect.DeepEqual(e.Report.Results, makeReport(m.Access, 6, 4).Results) {
					t.Fatal("native mapping changed")
				}
				for _, bad := range []string{"OneTrust", "YouTube CDN", "Steam Currency", "Netflix Preferred CDN", "Original RegionRestrictionCheck", "17 checks"} {
					if strings.Contains(e.Terminal, bad) {
						t.Fatal("out of scope", bad)
					}
				}
				if dir := os.Getenv("UNIFIED_ARTIFACT_DIR"); dir != "" {
					suffix := "mixed"
					if unknown {
						suffix = "unknown"
					}
					os.MkdirAll(dir, 0700)
					if er := os.WriteFile(filepath.Join(dir, region+"-"+suffix+".json"), out.Bytes(), 0600); er != nil {
						t.Fatal(er)
					}
					os.WriteFile(filepath.Join(dir, region+"-"+suffix+".ansi.txt"), []byte("OFFLINE MOCK - NOT REAL RESULTS\n"+e.Terminal), 0600)
				}
			})
		}
	}
}
func TestUnifiedRejectsUnmappedFlagsWithoutRun(t *testing.T) {
	for _, args := range [][]string{{"--region", "in"}, {"--region", "all"}, {"--extra-family", "46"}, {"--machine", "--json"}} {
		var out, err bytes.Buffer
		if code := runCLI(context.Background(), args, &out, &err, func(context.Context, time.Duration, int) (measurements, error) {
			t.Fatal("ran probes")
			return measurements{}, nil
		}); code != 2 {
			t.Fatal(code)
		}
	}
}
func TestNative35JSONFieldsBytePreserved(t *testing.T) {
	m := unifiedFixture("global", false)
	r := makeCompleteReport(m, 6, 4)
	old := makeReport(m.Access, 6, 4)
	if !reflect.DeepEqual(r.Results, old.Results) {
		t.Fatal("access JSON changed")
	}
	if r.Source["port"] != "Go v0.4.0 standalone three-group extraction with conservative error handling" || r.Network.AddressFamily != "system-selected" {
		t.Fatal("legacy metadata changed")
	}
	// The existing complete_test additionally exercises all original actual request paths.
	before, _ := json.Marshal(r)
	var b bytes.Buffer
	if e := writeMachine(&b, r, m, 0); e != nil {
		t.Fatal(e)
	}
	after, _ := json.Marshal(r)
	if !bytes.Equal(before, after) {
		t.Fatal("render/envelope mutated same measurement")
	}
}
