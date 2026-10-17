// Adapted from remnawave/geocheck v0.3.0 internal/render/report.go.
// Original table shape retained; information now uses terminal-default foreground.
// Commit 50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc.
// MIT License; Copyright (c) 2026 Remnawave. See LICENSE.
package render

import (
	lg "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"fmt"
	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/geo"
	"strconv"
	"strings"
	"time"
)

type Report struct {
	Access   []access.Result
	Services []geo.Result
	AI       []ai.Result
	Duration time.Duration
}

func (o *Output) line(format string, args ...any) {
	fmt.Fprintf(o.W, format+"\n", args...)
}

func (o *Output) accessTable(r Report) {
	t := o.Theme
	o.section("Stash checks")

	rows := make([][]string, 0, len(r.Access))
	for _, res := range r.Access {
		region := res.Region
		if region == "" {
			region = "—"
		}
		rows = append(rows, []string{
			res.Check.Name,
			o.accessStateCell(res.State),
			t.Muted.Render(region),
			t.Muted.Render(clip(res.Detail, 52)),
			o.accessTimeCell(res),
		})
	}
	o.line("%s", o.table([]string{"Service", "Status", "Region", "Detail", "Time"}, rows))

	s := access.Summarize(r.Access)
	var parts []string
	add := func(n int, label string, style lg.Style) {
		if n > 0 {
			parts = append(parts, style.Render(fmt.Sprintf("%d %s", n, label)))
		}
	}
	add(s.Available, "available", t.Good)
	add(s.Restricted, "restricted", t.Warn)
	add(s.Blocked, "blocked", t.Bad)
	add(s.Errors, "inconclusive", t.Error)
	if len(parts) > 0 {
		o.line("  %s", strings.Join(parts, t.Muted.Render(" · ")))
	}
	o.line("")
}

func (o *Output) accessStateCell(s access.State) string {
	t := o.Theme
	switch s {
	case access.StateAvailable:
		return t.Good.Render("● available")
	case access.StateRestricted:
		return t.Warn.Render("● restricted")
	case access.StateBlocked:
		return t.Bad.Render("● blocked")
	default:
		return t.Error.Render("○ error")
	}
}

func (o *Output) accessTimeCell(res access.Result) string {
	if res.Err != nil {
		return o.Theme.Muted.Render("—")
	}
	return o.rttCell(res.RTT)
}

func (o *Output) footer(r Report) {
	t := o.Theme
	if r.Duration > 0 {
		o.line("  %s", t.Muted.Render(fmt.Sprintf("completed in %.1fs", r.Duration.Seconds())))
	}
	o.line("")
}

func (o *Output) section(title string) {
	t := o.Theme
	width := o.Width - 4
	if width < 20 {
		width = 20
	}
	rule := strings.Repeat("─", maxInt(0, width-lg.Width(title)-3))
	o.line("  %s %s", t.Section.Render(title), t.Border.Render(rule))
}

func (o *Output) table(headers []string, rows [][]string) string {
	t := o.Theme
	return table.New().
		Border(lg.RoundedBorder()).
		BorderStyle(t.Border).
		BorderColumn(false).
		BorderRow(false).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, _ int) lg.Style {
			if row == table.HeaderRow {
				return t.TableHead
			}
			return t.TableCell
		}).
		Render()
}

func (o *Output) rttCell(d time.Duration) string {
	return o.Theme.Value.Render(formatRTT(d))
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func formatRTT(d time.Duration) string {
	if d == 0 {
		return "—"
	}
	ms := d.Seconds() * 1000
	switch {
	case ms < 10:
		return fmt.Sprintf("%.2f ms", ms)
	case ms < 100:
		return fmt.Sprintf("%.1f ms", ms)
	default:
		return fmt.Sprintf("%.0f ms", ms)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (o *Output) aiTable(r Report) {
	t := o.Theme
	o.section("AI endpoints")

	rows := make([][]string, 0, len(r.AI))
	for _, res := range r.AI {
		status := "—"
		if res.Status > 0 {
			status = strconv.Itoa(res.Status)
		}
		rows = append(rows, []string{
			res.Check.Name,
			o.aiStateCell(res.State),
			t.Muted.Render(status),
			t.Muted.Render(clip(res.Detail, 46)),
			o.aiTimeCell(res),
		})
	}
	o.line("%s", o.table([]string{"Endpoint", "Status", "HTTP", "Detail", "Time"}, rows))

	s := ai.Summarize(r.AI)
	var parts []string
	add := func(n int, label string, style lg.Style) {
		if n > 0 {
			parts = append(parts, style.Render(fmt.Sprintf("%d %s", n, label)))
		}
	}
	add(s.Reachable, "reachable", t.Good)
	add(s.Blocked, "blocked", t.Bad)
	add(s.Failed, "inconclusive", t.Error)
	if len(parts) > 0 {
		o.line("  %s", strings.Join(parts, t.Muted.Render(" · ")))
	}
	o.line("")
}

func (o *Output) aiStateCell(s ai.State) string {
	t := o.Theme
	switch s {
	case ai.StateReachable:
		return t.Good.Render("● reachable")
	case ai.StateBlocked:
		return t.Bad.Render("● blocked")
	default:
		return t.Error.Render("○ error")
	}
}

func (o *Output) aiTimeCell(res ai.Result) string {
	if res.RTT <= 0 {
		return o.Theme.Muted.Render("—")
	}
	return o.rttCell(res.RTT)
}
