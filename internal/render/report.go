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
		label, style := "○ 未知", o.Theme.Error
		if r.Err == nil {
			switch r.State {
			case access.StateAvailable:
				label, style = "● 可用", o.Theme.Good
			case access.StateRestricted:
				label, style = "● 受限", o.Theme.Warn
				if r.Check.ID == "netflix_access" {
					label = "● 仅自制"
				}
			case access.StateBlocked:
				label, style = "● 不可用", o.Theme.Bad
			}
		}
		rows = append(rows, displayRow{r.Check.Name, label, regionName(r.Region), style})
	}
	return rows
}
func (o *Output) aiRows(results []ai.Result) []displayRow {
	rows := make([]displayRow, 0, len(results))
	for _, r := range results {
		label, style := "○ 未知", o.Theme.Error
		if r.Err == nil {
			switch r.State {
			case ai.StateReachable:
				label, style = "● API可达", o.Theme.Good
			case ai.StateBlocked:
				label, style = "● 被拒绝", o.Theme.Bad
			}
		}
		rows = append(rows, displayRow{r.Check.Name, label, "—", style})
	}
	return rows
}

// Count displayed observations, never infer a global unlock verdict. Each
// distinct category remains separate (country vs availability vs restriction).
func summary(rows []displayRow) string {
	counts := map[string]int{}
	var order []string
	for _, r := range rows {
		label := strings.TrimPrefix(strings.TrimPrefix(r.result, "● "), "○ ")
		if counts[label] == 0 {
			order = append(order, label)
		}
		counts[label]++
	}
	parts := make([]string, 0, len(order))
	for _, label := range order {
		parts = append(parts, fmt.Sprintf("%s %d项", label, counts[label]))
	}
	return strings.Join(parts, " · ")
}
func (o *Output) groups(r Report) []displayGroup {
	var groups []displayGroup
	add := func(title string, rows []displayRow) {
		if len(rows) > 0 {
			groups = append(groups, displayGroup{title, rows, summary(rows)})
		}
	}
	add("服务地区与访问观察", o.serviceRows(r.Services))
	add("服务可用性", o.accessRows(r.Access))
	add("AI接口可达性", o.aiRows(r.AI))
	return groups
}

// Measure ALL groups before rendering any table. Widths are terminal cell
// widths, not byte/rune counts. Reduce the largest column until the shared
// layout fits; wrap grapheme clusters rather than truncate country names.
func sharedWidths(groups []displayGroup, terminal int) (widths [3]int, padding int) {
	widths = [3]int{4, 4, 4}
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
	} // Minimum: three intact two-CJK headers plus two outer borders.
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
	headers := []string{wrap("服务", 0), wrap("结果", 1), wrap("地区", 2)}
	cells := make([][]string, 0, len(rows))
	for _, r := range rows {
		cells = append(cells, []string{wrap(r.service, 0), r.style.Render(wrap(r.result, 1)), wrap(r.region, 2)})
	}
	return table.New().Border(lg.RoundedBorder()).BorderStyle(o.Theme.Border).
		BorderColumn(false).BorderRow(false).Wrap(true).Headers(headers...).Rows(cells...).
		StyleFunc(func(row, col int) lg.Style {
			style := o.Theme.TableCell
			if row == table.HeaderRow {
				style = o.Theme.TableHead
			}
			height := 1
			if row >= 0 {
				for _, cell := range cells[row] {
					if h := lg.Height(cell); h > height {
						height = h
					}
				}
			}
			return style.Padding(0, padding).Width(widths[col] + 2*padding).Height(height)
		}).Render()
}

// Prefer complete count categories on each summary line; do not leave a
// separator dangling at the start of the next line on ordinary narrow screens.
func wrapSummary(text string, width int) string {
	var lines []string
	line := ""
	for _, part := range strings.Split(text, " · ") {
		if line != "" && lg.Width(line+" · "+part) <= width {
			line += " · " + part
			continue
		}
		if line != "" {
			lines = append(lines, line)
		}
		line = ansi.Hardwrap(part, width, true)
	}
	if line != "" {
		lines = append(lines, line)
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
