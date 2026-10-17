// Three-group adapter around the original geocheck v0.3.0 renderer.
// MIT License; Copyright (c) 2026 Remnawave. See LICENSE.
package render

import (
	"bytes"
	"io"
	"os"

	"github.com/charmbracelet/colorprofile"
)

// WriteAccess writes the selected three groups using upstream terminal detection.
// Other io.Writers have no terminal fd, so they use the upstream plain profile.
func WriteAccess(w io.Writer, r Report) error {
	var out *Output
	if f, ok := w.(*os.File); ok {
		out = NewOutput(f)
	} else {
		plain := colorprofile.NewWriter(w, nil)
		plain.Profile = colorprofile.NoTTY
		out = &Output{W: plain, Theme: newTheme(nil), Width: 100}
	}
	return out.PrintAccess(r)
}

// PrintAccess retains the existing entrypoint and access layout while adding
// only Popular services and AI endpoints; no identity/GeoIP/CDN/routing sections.
func (o *Output) PrintAccess(r Report) error {
	var buf bytes.Buffer
	inner := &Output{W: &buf, Theme: o.Theme, Color: o.Color, Width: o.Width}
	if len(r.Services) > 0 {
		inner.servicesTable(r)
	}
	if len(r.Access) > 0 {
		inner.accessTable(r)
	}
	if len(r.AI) > 0 {
		inner.aiTable(r)
	}
	inner.footer(r)
	_, err := io.WriteString(o.W, buf.String())
	return err
}
