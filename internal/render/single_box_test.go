package render

import (
	"bytes"
	lg "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"regexp"
	"strings"
	"testing"
)

func TestSingleFrameNarrowSectionTitlesAndAllRowsLossless(t *testing.T) {
	r := completeMock(t)
	r.Services[0].Check.Name = "Fixture 中文 e\u0301"
	for _, width := range []int{14, 18, 20, 24, 30, 40, 60, 80, 100, 160} {
		var buf bytes.Buffer
		o := &Output{Theme: newTheme(nil), W: &buf, Width: width}
		groups := o.groups(r)
		o.printGroups(groups)
		text := ansi.Strip(buf.String())
		outer := 0
		for _, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
			if strings.HasPrefix(line, "╭") {
				outer = lg.Width(line)
			}
			if lg.Width(line) != outer || outer > width {
				t.Fatal("broken frame", width, lg.Width(line), outer, line)
			}
		}
		for _, corner := range []string{"╭", "╮", "╰", "╯"} {
			if strings.Count(text, corner) != 1 {
				t.Fatal("multiple frames", width, corner)
			}
		}
		// Reconstruct only full-span section rows delimited by library separators.
		// A long section title must wrap without truncation or escaping the frame.
		chunks := strings.Split(text, "├"+strings.Repeat("─", outer-2)+"┤\n")
		if len(chunks) != 2*len(groups)+1 {
			t.Fatal("section separators", width, len(chunks))
		}
		for i, g := range groups {
			got := ""
			for _, line := range strings.Split(chunks[1+2*i], "\n") {
				if strings.HasPrefix(line, "│") {
					got += strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "│"), "│"))
				}
			}
			if compact(got) != compact(g.title) {
				t.Fatal("section title truncated", width, got, g.title)
			}
		}
		if regexp.MustCompile(`(Yes|No|Limited|Unknown) [0-9]+:`).MatchString(text) {
			t.Fatal("summary", width)
		}
	}
}
