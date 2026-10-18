// Three-column terminal presentation built on geocheck v0.3.0's lipgloss tables.
// MIT License; Copyright (c) 2026 Remnawave. See LICENSE.
package render

import (
	"fmt"
	"time"

	lg "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/ansi"
	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/extra"
	"serviceunlock/internal/geo"
)

// Duration remains available to callers but is deliberately not printed.
// Raw measurements and JSON are not changed by terminal presentation.
type Report struct {
	Access   []access.Result
	Services []geo.Result
	AI       []ai.Result
	Duration time.Duration
	Extra    []extra.Result
}
type displayRow struct {
	service, result, region string
	style                   lg.Style
}
type displayGroup struct {
	title string
	rows  []displayRow
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

// Groups preserve the original order and per-row semantics; no summaries.
func (o *Output) groups(r Report) []displayGroup {
	var groups []displayGroup
	if rows := o.serviceRows(r.Services); len(rows) > 0 {
		groups = append(groups, displayGroup{"Service regions & access", rows})
	}
	if rows := o.accessRows(r.Access); len(rows) > 0 {
		groups = append(groups, displayGroup{"Service availability", rows})
	}
	if rows := o.aiRows(r.AI); len(rows) > 0 {
		groups = append(groups, displayGroup{"AI endpoints", rows})
	}
	for _, scope := range []string{"global", "regional"} {
		selected := []extra.Result{}
		for _, result := range r.Extra {
			if (scope == "global") == (result.Scope == "global") {
				selected = append(selected, result)
			}
		}
		if len(selected) > 0 {
			rows := o.extraRows(selected)
			title := "Global services"
			if scope != "global" {
				title = "Regional services"
			}
			groups = append(groups, displayGroup{title, rows})
		}
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

// tablePart uses the existing table renderer for both regular three-column
// rows and a single spanning section cell. Only the first/last parts get outer
// rounded corners; the joins are full-width separators inside ONE frame.
func (o *Output) tablePart(data [][]string, widths []int, padding int, heading, top, last bool) string {
	border := lg.RoundedBorder()
	if !last {
		border.BottomLeft = border.MiddleLeft
		border.BottomRight = border.MiddleRight
	}
	return table.New().Border(border).BorderStyle(o.Theme.Border).
		BorderTop(top).BorderBottom(true).BorderHeader(false).BorderColumn(false).BorderRow(false).Wrap(true).Rows(data...).
		StyleFunc(func(row, col int) lg.Style {
			style := o.Theme.TableCell
			if heading {
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
func (o *Output) headerPart(widths [3]int, padding int) string {
	headers := []string{}
	for i, s := range []string{"Service", "Result", "Region"} {
		headers = append(headers, ansi.Hardwrap(s, widths[i], true))
	}
	return o.tablePart([][]string{headers}, widths[:], padding, true, true, false)
}
func (o *Output) bodyPart(rows []displayRow, widths [3]int, padding int, last bool) string {
	cells := make([][]string, 0, len(rows))
	for _, r := range rows {
		cells = append(cells, []string{ansi.Hardwrap(r.service, widths[0], true), r.style.Render(ansi.Hardwrap(r.result, widths[1], true)), ansi.Hardwrap(r.region, widths[2], true)})
	}
	return o.tablePart(cells, widths[:], padding, false, false, last)
}
func (o *Output) threeColumnTable(rows []displayRow, widths [3]int, padding int) string {
	return o.headerPart(widths, padding) + "\n" + o.bodyPart(rows, widths, padding, true)
}
func (o *Output) printGroups(groups []displayGroup) {
	if len(groups) == 0 {
		return
	}
	widths, padding := sharedWidths(groups, o.Width)
	interior := widths[0] + widths[1] + widths[2] + 6*padding
	fmt.Fprintln(o.W, o.headerPart(widths, padding))
	for i, g := range groups {
		title := ansi.Hardwrap(g.title, interior-2*padding, true)
		fmt.Fprintln(o.W, o.tablePart([][]string{{title}}, []int{interior - 2*padding}, padding, true, false, false))
		fmt.Fprintln(o.W, o.bodyPart(g.rows, widths, padding, i == len(groups)-1))
	}
}
