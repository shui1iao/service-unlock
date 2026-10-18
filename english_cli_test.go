package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
	"unicode"
)

func assertEnglish(t *testing.T, text string) {
	t.Helper()
	if strings.IndexFunc(text, func(c rune) bool { return unicode.Is(unicode.Han, c) }) >= 0 {
		t.Fatal("non-English CLI output", text)
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestEnglishCLIMessagesAndUnchangedExitCodes(t *testing.T) {
	never := func(context.Context, time.Duration, int) (measurements, error) {
		t.Fatal("validation/help ran probes")
		return measurements{}, nil
	}
	for _, tc := range []struct {
		args []string
		code int
		want string
	}{{[]string{"--help"}, 0, "original 35 checks + global 6 + selected region"}, {[]string{"--workers", "0"}, 2, "Invalid arguments:"}, {[]string{"--unknown"}, 2, "Invalid arguments:"}, {[]string{"--timeout", "NaN"}, 2, "Invalid arguments:"}} {
		out, errout := new(bytes.Buffer), new(bytes.Buffer)
		if code := runCLI(context.Background(), tc.args, out, errout, never); code != tc.code {
			t.Fatal(code)
		}
		text := out.String() + errout.String()
		assertEnglish(t, text)
		if !strings.Contains(text, tc.want) {
			t.Fatal(text)
		}
	}
	for _, tc := range []struct {
		cancel, writeFailure bool
		want                 string
		code                 int
	}{{true, false, "Checks interrupted", 130}, {false, false, "Check execution failed:", 1}, {false, true, "Output failed:", 1}} {
		ctx, cancel := context.WithCancel(context.Background())
		if tc.cancel {
			cancel()
		}
		out, errout := new(bytes.Buffer), new(bytes.Buffer)
		var dst io.Writer = out
		if tc.writeFailure {
			dst = failWriter{}
		}
		run := func(context.Context, time.Duration, int) (measurements, error) {
			if tc.writeFailure {
				return measurements{Access: fixtureResults()}, nil
			}
			return measurements{}, errors.New("mock failure")
		}
		code := runCLI(ctx, nil, dst, errout, run)
		cancel()
		if code != tc.code || !strings.Contains(errout.String(), tc.want) {
			t.Fatal(code, errout)
		}
		assertEnglish(t, errout.String())
	}
}
func TestLauncherEnglishDiagnosticsFixedVersionAndNoExecutionOnFailure(t *testing.T) {
	script, err := os.ReadFile("check.sh")
	if err != nil {
		t.Fatal(err)
	}
	assertEnglish(t, string(script))
	if !strings.Contains(string(script), "BASE=https://github.com/shui1iao/service-unlock/releases/download/v0.5.3\n") {
		t.Fatal("launcher version changed")
	}
	f := newLaunchFixture(t, "riscv64", "curl", "sha256sum")
	code, out, errout := f.run(t)
	assertEnglish(t, errout)
	if code != 1 || out != "" || !strings.Contains(errout, "Unsupported architecture") {
		t.Fatal(code, out, errout)
	}
	f.notExecuted(t)
	f = newLaunchFixture(t, "amd64", "curl", "sha256sum")
	f.env = append(f.env, "MOCK_FAIL=manifest")
	code, out, errout = f.run(t)
	assertEnglish(t, errout)
	if code != 1 || out != "" || !strings.Contains(errout, "refusing to execute") {
		t.Fatal(code, out, errout)
	}
	f.notExecuted(t)
}
