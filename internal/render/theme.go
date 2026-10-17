// From remnawave/geocheck v0.3.0, commit 50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc.
// MIT License; Copyright (c) 2026 Remnawave. See LICENSE.
// Package render turns check results into terminal output and JSON.
package render

import (
	"io"
	"os"

	lg "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	xterm "github.com/charmbracelet/x/term"
)

// Theme holds every style the report uses.
type Theme struct {
	Title     lg.Style
	Subtitle  lg.Style
	Section   lg.Style
	Label     lg.Style
	Value     lg.Style
	Muted     lg.Style
	Error     lg.Style
	Good      lg.Style
	Warn      lg.Style
	Bad       lg.Style
	Info      lg.Style
	Accent    lg.Style
	TableHead lg.Style
	TableCell lg.Style
	Border    lg.Style
}

// Output bundles a writer with the theme appropriate for it.
type Output struct {
	W     io.Writer
	Theme Theme
	// Color reports whether the destination can show colour at all.
	Color bool
	// Width is the usable terminal width, or a sensible default when piped.
	Width int
}

// NewOutput inspects the destination and builds a matching writer and theme.
// Piped output, NO_COLOR and dumb terminals all degrade to plain text.
func NewOutput(f *os.File) *Output {
	env := os.Environ()
	profile := colorprofile.Detect(f, env)

	// colorprofile lets CLICOLOR_FORCE override NO_COLOR; NO_COLOR must win.
	if os.Getenv("NO_COLOR") != "" {
		profile = colorprofile.Ascii
	}

	isTTY := xterm.IsTerminal(f.Fd())
	colored := profile > colorprofile.Ascii

	width := 100
	if isTTY {
		if w, _, err := xterm.GetSize(f.Fd()); err == nil && w > 40 {
			width = w
		}
	}

	w := colorprofile.NewWriter(f, env)
	w.Profile = profile

	return &Output{W: w, Theme: newTheme(nil), Color: colored, Width: width}
}

// The terminal owns foreground/background selection. Never guess a background
// or send OSC queries. Only semantic status cells use the terminal ANSI palette.
// The legacy LightDark parameter is ignored, so fixtures can exercise both
// background settings without changing actual information colours.
func newTheme(_ lg.LightDarkFunc) Theme {
	plain := lg.NewStyle()
	bold := lg.NewStyle().Bold(true)
	return Theme{
		Title: bold, Subtitle: plain, Section: bold, Label: plain, Value: plain,
		Muted: plain, Error: bold,
		Good: lg.NewStyle().Bold(true).Foreground(lg.ANSIColor(2)),
		Warn: lg.NewStyle().Bold(true).Foreground(lg.ANSIColor(3)),
		Bad:  lg.NewStyle().Bold(true).Foreground(lg.ANSIColor(1)),
		Info: lg.NewStyle().Foreground(lg.ANSIColor(6)), Accent: bold,
		TableHead: bold.Padding(0, 1), TableCell: plain.Padding(0, 1), Border: plain,
	}
}
