package render

import (
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	"strings"
)

// Region names come exclusively from the pinned x/text CLDR English
// dictionary. ParseRegion handles ISO 3166 alpha-2/alpha-3 equivalence; neither
// the terminal nor this function infers a region from other observations.
var englishRegions = display.Regions(language.English)

func regionName(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 2 && len(code) != 3 {
		return ""
	}
	for _, c := range code {
		if c < 'A' || c > 'Z' {
			return ""
		}
	}
	region, err := language.ParseRegion(code)
	if err != nil || !region.IsCountry() || region.IsPrivateUse() {
		return ""
	}
	name := englishRegions.Name(region)
	if name == "" || name == region.String() || name == "Unknown Region" {
		return ""
	}
	return name
}
