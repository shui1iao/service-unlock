//go:build linux

// Real local PTY only; no terminal queries or external measurements.
package render

import (
	"fmt"
	"os"
	"testing"

	"github.com/charmbracelet/colorprofile"
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
	for _, width := range []int{120, 30} {
		if err = unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 24, Col: uint16(width)}); err != nil {
			t.Fatal(err)
		}
		o := NewOutput(slave)
		want := width
		if width <= 40 {
			want = 100
		}
		if o.Width != want || !o.Color || o.W.(*colorprofile.Writer).Profile != colorprofile.TrueColor {
			t.Fatal(o.Width, o.Color, o.W)
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
