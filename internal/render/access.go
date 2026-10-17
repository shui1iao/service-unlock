// Access-only adapter around the original geocheck v0.3.0 renderer.
// MIT License; Copyright (c) 2026 Remnawave. See LICENSE.
package render

import (
	"bytes"
	"io"
	"os"

	lg "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// WriteAccess uses the upstream terminal detection for real file destinations.
// Other io.Writers have no terminal fd, so they use the upstream plain profile.
func WriteAccess(w io.Writer, r Report) error {
	var out *Output
	if f, ok := w.(*os.File); ok {
		out = NewOutput(f)
	} else {
		plain := colorprofile.NewWriter(w, nil)
		plain.Profile = colorprofile.NoTTY
		out = &Output{W: plain, Theme: newTheme(lg.LightDark(backgroundIsDark())), Width: 100}
	}
	return out.PrintAccess(r)
}

// PrintAccess preserves the original access table and footer, without the
// unrelated identity, geolocation, banner, routing, or API sections.
func (o *Output) PrintAccess(r Report) error {
	var buf bytes.Buffer
	inner := &Output{W: &buf, Theme: o.Theme, Color: o.Color, Width: o.Width}
	inner.accessTable(r)
	inner.footer(r)
	_, err := io.WriteString(o.W, buf.String())
	return err
}
