package render

import (
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	"strings"
)

// ParseRegion provides the pinned library's ISO alpha-2/alpha-3 mapping.
// Keep the existing country/private-use/known-region checks; return only the
// canonical alpha-2 code, never a name or a region inferred from another probe.
var englishRegions = display.Regions(language.English)

func regionCode(code string) string {
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
	canonical := region.String()
	if len(canonical) != 2 || canonical[0] < 'A' || canonical[0] > 'Z' || canonical[1] < 'A' || canonical[1] > 'Z' {
		return ""
	}
	return canonical
}
