//go:build linux

// Real local PTY only; no terminal queries or external measurements.
package render

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	lg "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/sys/unix"
)

func TestTTYWidthAndProfiles(t *testing.T) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	if err = unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	number, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	for _, name := range []string{"NO_COLOR", "CLICOLOR", "CLICOLOR_FORCE", "CI"} {
		t.Setenv(name, "")
	}
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("COLORTERM", "truecolor")
	for _, bg := range []string{"", "15;0", "0;15"} {
		t.Setenv("COLORFGBG", bg)
		for _, width := range []int{14, 18, 20, 30, 40, 80, 120, 160} {
			if err = unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 24, Col: uint16(width)}); err != nil {
				t.Fatal(err)
			}
			o := NewOutput(slave)
			want := width

			if o.Width != want || !o.Color || o.W.(*colorprofile.Writer).Profile != colorprofile.TrueColor {
				t.Fatal(o.Width, o.Color, o.W)
			}
			if strings.Contains(o.Theme.TableCell.Render("body"), "\x1b") || strings.Contains(o.Theme.Muted.Render("detail"), "\x1b") {
				t.Fatal("TTY information forced a foreground", bg)
			}
			var rendered bytes.Buffer
			measured := &Output{W: &rendered, Theme: o.Theme, Color: o.Color, Width: o.Width}
			if err := measured.PrintAccess(completeMock(t)); err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.Split(ansi.Strip(rendered.String()), "\n") {
				if lg.Width(line) > width {
					t.Fatal("real PTY width ignored", width, line)
				}
			}
		}
	}
	t.Setenv("CLICOLOR_FORCE", "1")
	t.Setenv("NO_COLOR", "not-empty")
	o := NewOutput(slave)
	if o.Color || o.W.(*colorprofile.Writer).Profile != colorprofile.Ascii {
		t.Fatal("NO_COLOR must win")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "")
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "dumb")
	o = NewOutput(slave)
	if o.Color || o.W.(*colorprofile.Writer).Profile != colorprofile.NoTTY {
		t.Fatal("dumb terminal behavior changed")
	}
}

func TestActualPTYSingleBoxWritesDeepLightAndNoColor(t *testing.T) {
	for _, width := range []int{14, 40, 100, 160} {
		for _, bg := range []string{"15;0", "0;15"} {
			for _, noColor := range []bool{false, true} {
				master, e := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
				if e != nil {
					t.Fatal(e)
				}
				if e = unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0); e != nil {
					t.Fatal(e)
				}
				number, e := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
				if e != nil {
					t.Fatal(e)
				}
				slave, e := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|unix.O_NOCTTY, 0)
				if e != nil {
					t.Fatal(e)
				}
				if e = unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 24, Col: uint16(width)}); e != nil {
					t.Fatal(e)
				}
				t.Setenv("TERM", "xterm-256color")
				t.Setenv("COLORTERM", "truecolor")
				t.Setenv("COLORFGBG", bg)
				t.Setenv("CLICOLOR_FORCE", "")
				t.Setenv("CI", "")
				if noColor {
					t.Setenv("NO_COLOR", "1")
				} else {
					t.Setenv("NO_COLOR", "")
				}
				done := make(chan string, 1)
				go func() { var b bytes.Buffer; io.Copy(&b, master); done <- b.String() }()
				if e = WriteAccess(slave, completeMock(t)); e != nil {
					t.Fatal(e)
				}
				slave.Close()
				var raw string
				select {
				case raw = <-done:
				case <-time.After(3 * time.Second):
					master.Close()
					t.Fatal("PTY capture timed out")
				}
				master.Close()
				text := strings.ReplaceAll(ansi.Strip(raw), "\r\n", "\n")
				if strings.Count(text, "╭") != 1 || strings.Count(text, "╰") != 1 {
					t.Fatal("PTY lost frame", width)
				}
				for _, line := range strings.Split(text, "\n") {
					if lg.Width(line) > width {
						t.Fatal("PTY overflow", width, line)
					}
				}
				// Existing ASCII profile preserves bold, but removes all colors.
				if noColor && (strings.Contains(raw, "38;") || strings.Contains(raw, "48;") || strings.Contains(raw, "\x1b]")) {
					t.Fatal("NO_COLOR lost on real PTY")
				}
				if !noColor && (!strings.Contains(raw, "\x1b[") || strings.Contains(raw, "\x1b]")) {
					t.Fatal("PTY color/OSC contract")
				}
			}
		}
	}
}
