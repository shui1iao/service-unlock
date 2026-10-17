// Group-level display explanations only; raw measurements/JSON are untouched.
package render

import (
	"fmt"
	"strings"

	"serviceunlock/internal/access"
	"serviceunlock/internal/geo"
)

func rowLabel(row displayRow) string {
	return strings.TrimPrefix(strings.TrimPrefix(row.result, "● "), "○ ")
}
func counted(n int, singular, plural string) string {
	if n == 0 {
		return ""
	}
	if n == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}
func nonempty(parts ...string) string {
	var out []string
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, ", ")
}
func statusLine(label string, n int, detail string) string {
	return fmt.Sprintf("%s %d: %s", label, n, detail)
}

func serviceSummary(rows []displayRow, results []geo.Result) string {
	var regions, accessible, noMarkers, denied, markers, unknown int
	for i, row := range rows {
		switch rowLabel(row) {
		case "Yes":
			switch results[i].Check.Kind {
			case geo.KindCountry:
				regions++
			case geo.KindAvailability:
				accessible++
			case geo.KindBlocked:
				noMarkers++
			}
		case "No":
			switch results[i].Check.Kind {
			case geo.KindAvailability:
				denied++
			case geo.KindBlocked:
				markers++
			}
		default:
			unknown++
		}
	}
	var lines []string
	if n := regions + accessible + noMarkers; n > 0 {
		lines = append(lines, statusLine("Yes", n, nonempty(
			counted(regions, "region identified", "regions identified"),
			counted(accessible, "accessible", "accessible"),
			counted(noMarkers, "without restriction markers", "without restriction markers"))))
	}
	if n := denied + markers; n > 0 {
		lines = append(lines, statusLine("No", n, nonempty(
			counted(denied, "access denied", "access denied"), counted(markers, "with restriction markers", "with restriction markers"))))
	}
	if unknown > 0 {
		lines = append(lines, statusLine("Unknown", unknown, "inconclusive"))
	}
	return strings.Join(lines, "\n")
}
func accessSummary(rows []displayRow, results []access.Result) string {
	var yes, no, originals, restricted, unknown int
	for i, row := range rows {
		switch rowLabel(row) {
		case "Yes":
			yes++
		case "No":
			no++
		case "Limited":
			if results[i].Check.ID == "netflix_access" {
				originals++
			} else {
				restricted++
			}
		default:
			unknown++
		}
	}
	var lines []string
	if yes > 0 {
		lines = append(lines, statusLine("Yes", yes, "available"))
	}
	if no > 0 {
		lines = append(lines, statusLine("No", no, "blocked"))
	}
	if n := originals + restricted; n > 0 {
		detail := "originals-only"
		if originals == 0 {
			detail = "restricted"
		}
		if originals > 0 && restricted > 0 {
			detail = nonempty(counted(originals, "originals-only", "originals-only"), counted(restricted, "restricted", "restricted"))
		}
		lines = append(lines, statusLine("Limited", n, detail))
	}
	if unknown > 0 {
		lines = append(lines, statusLine("Unknown", unknown, "inconclusive"))
	}
	return strings.Join(lines, "\n")
}
func aiSummary(rows []displayRow) string {
	counts := map[string]int{}
	for _, row := range rows {
		counts[rowLabel(row)]++
	}
	var lines []string
	for _, category := range []struct{ label, detail string }{{"Yes", "API reachable"}, {"No", "region/request refused"}, {"Unknown", "inconclusive"}} {
		if n := counts[category.label]; n > 0 {
			lines = append(lines, statusLine(category.label, n, category.detail))
		}
	}
	return strings.Join(lines, "\n")
}
