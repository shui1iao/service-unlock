// Three-column terminal presentation built on geocheck v0.3.0's lipgloss tables.
// MIT License; Copyright (c) 2026 Remnawave. See LICENSE.
package render

import (
	"fmt"
	"strings"
	"time"

	lg "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/ansi"
	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/geo"
)

// Duration remains available to callers but is deliberately not printed.
// Raw measurements and JSON are not changed by terminal presentation.
type Report struct {
	Access   []access.Result
	Services []geo.Result
	AI       []ai.Result
	Duration time.Duration
}
type displayRow struct {
	service, result, region string
	style                   lg.Style
}
type displayGroup struct {
	title   string
	rows    []displayRow
	summary string
}

func (o *Output) accessRows(results []access.Result) []displayRow {
	rows := make([]displayRow, 0, len(results))
	for _, r := range results {
		label, style := "○ Unknown", o.Theme.Error
		if r.Err == nil {
			switch r.State {
			case access.StateAvailable:
				label, style = "● Yes", o.Theme.Good
			case access.StateRestricted:
				label, style = "● Limited", o.Theme.Warn
			case access.StateBlocked:
				label, style = "● No", o.Theme.Bad
			}
		}
		rows = append(rows, displayRow{r.Check.Name, label, regionCode(r.Region), style})
	}
	return rows
}
func (o *Output) aiRows(results []ai.Result) []displayRow {
	rows := make([]displayRow, 0, len(results))
	for _, r := range results {
		label, style := "○ Unknown", o.Theme.Error
		if r.Err == nil {
			switch r.State {
			case ai.StateReachable:
				label, style = "● Yes", o.Theme.Good
			case ai.StateBlocked:
				label, style = "● No", o.Theme.Bad
			}
		}
		rows = append(rows, displayRow{r.Check.Name, label, "", style})
	}
	return rows
}

// Detailed summaries classify only the rows already selected for display.
// They do not reinterpret HTTP replies, promote Unknown using a region, or
// make an overall unlock/login verdict.
func (o *Output) groups(r Report) []displayGroup {
	var groups []displayGroup
	if rows := o.serviceRows(r.Services); len(rows) > 0 {
		groups = append(groups, displayGroup{"Service regions & access", rows, serviceSummary(rows, r.Services)})
	}
	if rows := o.accessRows(r.Access); len(rows) > 0 {
		groups = append(groups, displayGroup{"Service availability", rows, accessSummary(rows, r.Access)})
	}
	if rows := o.aiRows(r.AI); len(rows) > 0 {
		groups = append(groups, displayGroup{"AI endpoints", rows, aiSummary(rows)})
	}
	return groups
}

// Measure ALL groups before rendering any table. Widths are terminal cell
// widths, not byte/rune counts. Reduce the largest column until the shared
// layout fits; wrap grapheme clusters rather than truncate country names.
func sharedWidths(groups []displayGroup, terminal int) (widths [3]int, padding int) {
	widths = [3]int{7, 6, 6}
	for _, g := range groups {
		for _, r := range g.rows {
			for col, value := range []string{r.service, r.result, r.region} {
				if w := lg.Width(value); w > widths[col] {
					widths[col] = w
				}
			}
		}
	}
	if terminal <= 0 {
		terminal = 100
	}
	if terminal < 14 {
		terminal = 14
	} // Minimum: 4 terminal cells per column plus two outer borders; headers wrap.
	padding = 1
	if terminal < 20 {
		padding = 0
	}
	budget := terminal - 2 - 6*padding
	for widths[0]+widths[1]+widths[2] > budget {
		col := 0
		for i := 1; i < 3; i++ {
			if widths[i] > widths[col] {
				col = i
			}
		}
		if widths[col] <= 4 {
			break
		}
		widths[col]--
	}
	return widths, padding
}

func (o *Output) threeColumnTable(rows []displayRow, widths [3]int, padding int) string {
	wrap := func(s string, col int) string { return ansi.Hardwrap(s, widths[col], true) }
	headers := []string{wrap("Service", 0), wrap("Result", 1), wrap("Region", 2)}
	cells := make([][]string, 0, len(rows))
	for _, r := range rows {
		cells = append(cells, []string{wrap(r.service, 0), r.style.Render(wrap(r.result, 1)), wrap(r.region, 2)})
	}
	// lipgloss intentionally truncates its built-in header to one line. Render
	// the header as a wrapping table row and join its library-generated separator
	// to the body. This is one visual rounded table with identical fixed widths,
	// not a handcrafted border or a shortened/abbreviated header.
	part := func(data [][]string, header bool) string {
		border := lg.RoundedBorder()
		if header {
			border.BottomLeft = border.MiddleLeft
			border.BottomRight = border.MiddleRight
		}
		return table.New().Border(border).BorderStyle(o.Theme.Border).
			BorderTop(header).BorderBottom(true).BorderHeader(false).BorderColumn(false).BorderRow(false).Wrap(true).Rows(data...).
			StyleFunc(func(row, col int) lg.Style {
				style := o.Theme.TableCell
				if header {
					style = o.Theme.TableHead
				}
				height := 1
				for _, cell := range data[row] {
					if h := lg.Height(cell); h > height {
						height = h
					}
				}
				return style.Padding(0, padding).Width(widths[col] + 2*padding).Height(height)
			}).Render()
	}
	return part([][]string{headers}, true) + "\n" + part(cells, false)
}

// Each status gets its own logical line. Word-wrap where possible; hard-wrap
// only words longer than the actual table width so even narrow screens fit.
func wrapSummary(text string, width int) string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		lines = append(lines, ansi.Wrap(line, width, ""))
	}
	return strings.Join(lines, "\n")
}

func (o *Output) printGroups(groups []displayGroup) {
	widths, padding := sharedWidths(groups, o.Width)
	outer := 2 + widths[0] + widths[1] + widths[2] + 6*padding
	for _, g := range groups {
		title := ansi.Hardwrap(g.title, outer, true)
		if w := lg.Width(title); !strings.Contains(title, "\n") && w+1 < outer {
			title += " " + strings.Repeat("─", outer-w-1)
		}
		fmt.Fprintln(o.W, o.Theme.Section.Render(title))
		fmt.Fprintln(o.W, o.threeColumnTable(g.rows, widths, padding))
		fmt.Fprintln(o.W, wrapSummary(g.summary, outer))
		fmt.Fprintln(o.W)
	}
}
