// MIT License; Copyright (c) 2026 Remnawave. Offline rendering tests only.
package render

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	lg "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"serviceunlock/internal/access"
)

func mockReport(t *testing.T) Report {
	t.Helper()
	raw, err := os.ReadFile("testdata/input.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Duration int64 `json:"duration_ns"`
		Results  []struct {
			ID, Name, Region, Detail, Error string
			State                           access.State
			RTT                             int64 `json:"rtt_ns"`
		}
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	r := Report{Duration: time.Duration(fixture.Duration)}
	for _, x := range fixture.Results {
		res := access.Result{Check: access.Check{ID: x.ID, Name: x.Name}, State: x.State, Region: x.Region, Detail: x.Detail, RTT: time.Duration(x.RTT)}
		if x.Error != "" {
			res.Err = errors.New(x.Error)
		}
		r.Access = append(r.Access, res)
	}
	return r
}
func renderProfile(t *testing.T, r Report, profile colorprofile.Profile) string {
	t.Helper()
	var buf bytes.Buffer
	w := colorprofile.NewWriter(&buf, nil)
	w.Profile = profile
	o := &Output{W: w, Theme: newTheme(lg.LightDark(true)), Color: profile > colorprofile.Ascii, Width: 100}
	if err := o.PrintAccess(r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestAdapterPreservesWriteErrors(t *testing.T) {
	if err := WriteAccess(brokenWriter{}, mockReport(t)); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}
