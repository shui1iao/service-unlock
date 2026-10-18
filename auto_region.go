// Automatic scheduling only; existing service probe semantics are untouched.
package main

import (
	"context"
	"fmt"
	"serviceunlock/internal/extra"
	"serviceunlock/internal/netx"
	"strings"
)

// Same bounded country groups as GUKO's established STREAM_REGION_BY_COUNTRY.
// This is not a content-provider locale/country list or an external GeoIP DB.
var countryGroups = func() map[string]string {
	out := map[string]string{"TW": "tw", "HK": "hk", "MO": "hk", "JP": "jp", "KR": "kr"}
	groups := map[string]string{
		"sea": "SG MY TH ID PH VN KH LA MM BN TL",
		"na":  "US CA MX GL BM",
		"sa":  "BR AR CL CO PE VE UY PY BO EC GY SR",
		"oc":  "AU NZ FJ PG WS TO VU NC PF",
		"af":  "ZA NG KE GH EG MA DZ TN ET TZ UG SN CI AO MZ BW NA ZW ZM",
		"eu":  "GB UK IE FR DE NL BE LU ES PT IT CH AT PL CZ SK HU RO BG GR CY MT DK SE NO FI IS EE LV LT SI HR RS BA ME MK AL MD UA",
	}
	for group, codes := range groups {
		for _, code := range strings.Fields(codes) {
			out[code] = group
		}
	}
	return out
}()
var groupNames = map[string]string{"global": "Global", "tw": "Taiwan", "hk": "Hong Kong", "jp": "Japan", "na": "North America", "sa": "South America", "eu": "Europe", "oc": "Oceania", "kr": "Korea", "sea": "Southeast Asia", "af": "Africa"}

func selectRegion(ctx context.Context, requested string, client *extra.Client, familyErr error) *selection {
	if requested == "" {
		requested = "auto"
	}
	s := &selection{Region: requested, RequestedRegion: requested, ResolutionReason: "explicit_region_override"}
	if requested != "auto" {
		return s
	}
	s.Region = "global"
	if familyErr != nil || (client.Family != 4 && client.Family != 6) {
		s.DetectionError = "no_extra_family_route"
		s.ResolutionReason = "detection_failed"
		return s
	}
	detected := client.DetectCountry(ctx)
	s.DetectedCountry, s.DetectionSource, s.DetectionError = detected.Country, detected.Source, detected.Error
	if detected.Error != "" {
		s.ResolutionReason = "detection_failed"
		return s
	}
	if group, ok := countryGroups[detected.Country]; ok {
		s.Region = group
		s.ResolutionReason = "country_mapped"
	} else {
		s.ResolutionReason = "unsupported_region_country"
	}
	return s
}
func selectionNotice(s *selection) string {
	if s == nil || s.RequestedRegion != "auto" {
		return ""
	}
	if s.ResolutionReason == "country_mapped" {
		return fmt.Sprintf("Automatic region: %s -> %s (%s)", s.DetectedCountry, groupNames[s.Region], s.Region)
	}
	if s.ResolutionReason == "unsupported_region_country" {
		return fmt.Sprintf("Automatic region: %s -> Global (unsupported_region_country)", s.DetectedCountry)
	}
	return "Automatic region: unavailable -> Global (detection_failed)"
}

// Injectable *clients*, not alternative probes: tests exercise the actual
// detection, original35 and extra.Run request paths through mock transports.
func runWithClients(ctx context.Context, workers int, stack *netx.Stack, client *extra.Client, familyErr error) (measurements, error) {
	options, _ := ctx.Value(extraOptionsKey{}).(extraOptions)
	if options.Region == "" {
		options = extraOptions{"auto", "auto"}
	}
	resolved := selectRegion(ctx, options.Region, client, familyErr)
	if ctx.Err() != nil {
		return measurements{}, ctx.Err()
	}
	checks, err := extra.Selected(resolved.Region)
	if err != nil {
		return measurements{}, err
	}
	m, err := runGroups(ctx, stack, workers)
	if err != nil {
		return m, err
	}
	m.Extra = extra.Run(ctx, client, resolved.Region, checks, workers)
	if familyErr != nil {
		for i := range m.Extra {
			detail := familyErr.Error()
			m.Extra[i].Status = "error"
			m.Extra[i].Error = &detail
			m.Extra[i].Detail = detail
		}
	}
	m.Selection = makeSelection(resolved.Region, options.Family, client.Family, checks)
	m.Selection.RequestedRegion = resolved.RequestedRegion
	m.Selection.DetectedCountry = resolved.DetectedCountry
	m.Selection.DetectionSource = resolved.DetectionSource
	m.Selection.DetectionError = resolved.DetectionError
	m.Selection.ResolutionReason = resolved.ResolutionReason
	return m, ctx.Err()
}
