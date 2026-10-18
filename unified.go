// Unified native candidate integration; RRC-derived extensions AGPL-3.0-only.
package main

import (
	"bytes"
	"encoding/json"
	"github.com/charmbracelet/colorprofile"
	"io"
	"serviceunlock/internal/extra"
	"serviceunlock/internal/render"
	"time"
)

type extraOptionsKey struct{}
type extraOptions struct{ Region, Family string }
type selection struct {
	Region           string   `json:"region"`
	RequestedFamily  string   `json:"extra_family_requested"`
	Family           int      `json:"extra_family"`
	ExpectedIDs      []string `json:"extra_expected_ids"`
	Source           string   `json:"extra_source_commit"`
	RequestedRegion  string   `json:"requested_region,omitempty"`
	DetectedCountry  string   `json:"detected_country,omitempty"`
	DetectionSource  string   `json:"detection_source,omitempty"`
	DetectionError   string   `json:"detection_error,omitempty"`
	ResolutionReason string   `json:"resolution_reason,omitempty"`
}

func makeSelection(region, mode string, family int, checks []extra.Check) *selection {
	s := &selection{Region: region, RequestedFamily: mode, Family: family, Source: extra.SourceCommit, ExpectedIDs: []string{}}
	for _, c := range checks {
		s.ExpectedIDs = append(s.ExpectedIDs, c.ID)
	}
	return s
}
func writeMachine(w io.Writer, r report, m measurements, elapsed time.Duration) error {
	var buf bytes.Buffer
	if notice := selectionNotice(r.Selection); notice != "" {
		buf.WriteString(notice + "\n")
	}
	// Explicit ANSI is confined to this machine capture; process environment is untouched.
	profile := colorprofile.NewWriter(&buf, nil)
	profile.Profile = colorprofile.TrueColor
	terminal := render.NewMachineOutput(profile)
	if err := terminal.PrintAccess(render.Report{Access: m.Access, Services: m.Services, AI: m.AI, Extra: m.Extra, Duration: elapsed}); err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(struct {
		Format   string `json:"format"`
		Report   report `json:"report"`
		Terminal string `json:"terminal"`
	}{"guko-unified-v1", r, buf.String()})
}
