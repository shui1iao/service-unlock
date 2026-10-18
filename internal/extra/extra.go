// Native translations of selected RegionRestrictionCheck probes.
// Derived from lmc999/RegionRestrictionCheck b6d4a6f9a87fc6eae6d3e62d0092ececcec8e844.
// SPDX-License-Identifier: AGPL-3.0-only. Modified 2026-10-18. See LICENSE/NOTICE.
package extra

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"golang.org/x/text/language"
	"net"
	"strings"
	"sync"
	"time"
)

const SourceCommit = "b6d4a6f9a87fc6eae6d3e62d0092ececcec8e844"

//go:embed catalogue-source.json
var catalogueBytes []byte

type sourceEntry struct {
	Function        string   `json:"function"`
	Name            string   `json:"display_name"`
	Groups          []string `json:"groups"`
	Line            int      `json:"source_line"`
	IPv6Unsupported bool     `json:"ipv6_unsupported_guard"`
	Safety          string   `json:"safety_exception"`
}
type Check struct {
	ID, Name, SourceFunction string
	Groups                   []string
	SourceLine               int
	IPv6Unsupported          bool
	Safety                   string
}

var catalogue = func() []Check {
	var input []sourceEntry
	if err := json.Unmarshal(catalogueBytes, &input); err != nil {
		panic(err)
	}
	out := make([]Check, 0, len(input))
	for _, v := range input {
		name := v.Name
		if v.Function == "MediaUnlockTest_PrimeVideo" {
			name = "Amazon Prime Video (access)"
		}
		out = append(out, Check{v.Function, name, v.Function, v.Groups, v.Line, v.IPv6Unsupported, v.Safety})
	}
	return out
}()
var regionOrder = map[string][]string{}

func Regions() []string {
	return []string{"global", "tw", "hk", "jp", "na", "sa", "eu", "oc", "kr", "sea", "af"}
}

// Catalogue returns all 134 selected functions, including explicit safety Unknowns.
func Catalogue() []Check { return append([]Check(nil), catalogue...) }
func Selected(region string) ([]Check, error) {
	valid := false
	for _, r := range Regions() {
		if r == region {
			valid = true
		}
	}
	if !valid {
		return nil, fmt.Errorf("unsupported region %q", region)
	}
	// Original call order is material; generated order per group is fixed source data.
	byID := map[string]Check{}
	for _, c := range catalogue {
		byID[c.ID] = c
	}
	var ids []string
	ids = append(ids, groupIDs["global"]...)
	if region != "global" {
		ids = append(ids, groupIDs[region]...)
	}
	seen := map[string]bool{}
	out := []Check{}
	for _, id := range ids {
		if !seen[id] {
			out = append(out, byID[id])
			seen[id] = true
		}
	}
	return out, nil
}

type Result struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	SourceFunction string  `json:"source_function"`
	SourceLine     int     `json:"source_line"`
	Scope          string  `json:"scope"`
	Family         int     `json:"family"`
	Status         string  `json:"status"`
	Region         *string `json:"region"`
	Detail         string  `json:"detail"`
	Error          *string `json:"error"`
	CheckedAt      string  `json:"checked_at"`
	ElapsedMS      float64 `json:"elapsed_ms"`
}
type verdict struct{ status, region, detail string }

func yes(region string) verdict {
	return verdict{"available", country(region), "source positive criterion matched"}
}
func no(detail string) verdict              { return verdict{"blocked", "", detail} }
func limited(region, detail string) verdict { return verdict{"restricted", country(region), detail} }
func unknown(detail string) verdict         { return verdict{"error", "", detail} }
func country(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) != 2 || s[0] < 'A' || s[0] > 'Z' || s[1] < 'A' || s[1] > 'Z' {
		return ""
	}
	region, err := language.ParseRegion(s)
	if err != nil || !region.IsCountry() || region.IsPrivateUse() || region.String() == "ZZ" {
		return ""
	}
	return region.String()
}

// UDP connect performs local route selection only; no Write or packet is issued.
func SelectFamily(mode string) (int, error) {
	if mode == "4" {
		return 4, nil
	}
	if mode == "6" {
		return 6, nil
	}
	if mode != "auto" {
		return 0, fmt.Errorf("extra-family must be auto, 4, or 6")
	}
	for _, v := range []struct {
		network, address string
		family           int
	}{{"udp4", "1.1.1.1:53", 4}, {"udp6", "[2606:4700:4700::1111]:53", 6}} {
		conn, err := net.Dial(v.network, v.address)
		if err == nil {
			conn.Close()
			return v.family, nil
		}
	}
	return 0, fmt.Errorf("no local IPv4/IPv6 route")
}
func Run(ctx context.Context, client *Client, region string, checks []Check, workers int) []Result {
	if workers < 1 {
		workers = 4
	}
	out := make([]Result, len(checks))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for n := 0; n < workers; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				c := checks[i]
				start := time.Now()
				r := Result{ID: c.ID, Name: c.Name, SourceFunction: c.SourceFunction, SourceLine: c.SourceLine, Scope: "global", Family: client.Family, CheckedAt: start.UTC().Format("2006-01-02T15:04:05.000Z")}
				global := false
				for _, g := range c.Groups {
					if g == "global" {
						global = true
					}
				}
				if !global {
					r.Scope = region
				}
				v := unknown("cancelled")
				if ctx.Err() == nil {
					switch {
					case client.Family != 4 && client.Family != 6:
						v = unknown("no local IPv4/IPv6 route")
					case c.Safety != "":
						v = unknown("unsupported for safety: " + c.Safety)
					case c.IPv6Unsupported && client.Family == 6:
						v = unknown("source probe does not support IPv6")
					default:
						v = execute(ctx, client, c.ID)
					}
				}
				if ctx.Err() != nil {
					v = unknown(ctx.Err().Error())
				}
				r.Status = v.status
				r.Detail = v.detail
				if v.region != "" {
					r.Region = &v.region
				}
				if v.status == "error" {
					r.Error = &v.detail
				}
				r.ElapsedMS = float64(time.Since(start).Microseconds()) / 1000
				out[i] = r
			}
		}()
	}
	for i := range checks {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return out
}
