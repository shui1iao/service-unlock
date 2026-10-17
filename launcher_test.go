// POSIX launcher regressions. All downloads are mocked; no GitHub/platform I/O.
// MIT License; Copyright (c) 2026 Remnawave.
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

type launchFixture struct {
	root, bin, tmp, release, marker, log, script string
	env                                          []string
}

func writeFixture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0700); err != nil {
		t.Fatal(err)
	}
}
func newLaunchFixture(t *testing.T, arch, downloader, verifier string) *launchFixture {
	t.Helper()
	f := &launchFixture{root: t.TempDir()}
	f.bin = filepath.Join(f.root, "bin")
	f.tmp = filepath.Join(f.root, "tmp")
	f.release = filepath.Join(f.root, "release")
	f.marker = filepath.Join(f.root, "ran")
	f.log = filepath.Join(f.root, "urls")
	for _, p := range []string{f.bin, f.tmp, f.release} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	f.script, _ = filepath.Abs("check.sh")
	for _, name := range []string{"mktemp", "rm", "chmod", "cp"} {
		real, err := exec.LookPath(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, filepath.Join(f.bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	if verifier != "none" {
		real, err := exec.LookPath(verifier)
		if err != nil {
			t.Skipf("optional checksum tool not installed: %s", verifier)
		}
		if err := os.Symlink(real, filepath.Join(f.bin, verifier)); err != nil {
			t.Fatal(err)
		}
	}
	writeFixture(t, filepath.Join(f.bin, "uname"), `#!/bin/sh
case "$1" in -s) printf '%s\n' "${MOCK_OS:-Linux}";; -m) printf '%s\n' "$MOCK_ARCH";; *) exit 1;; esac
`)
	if downloader != "none" {
		writeFixture(t, filepath.Join(f.bin, downloader), `#!/bin/sh
out=
url=
while [ "$#" -gt 0 ]; do
 case "$1" in -o|-O) out=$2; shift;; -*) :;; *) url=$1;; esac
 shift
done
base=https://github.com/shui1iao/service-unlock/releases/download/v0.3.1
case "$url" in "$base/SHA256SUMS"|"$base/service-unlock_linux_amd64"|"$base/service-unlock_linux_arm64") :;; *) exit 91;; esac
printf '%s\n' "$url" >> "$MOCK_LOG"
if [ "${MOCK_HANG:-0}" = 1 ]; then
 printf '%s' "$$" > "$DOWNLOAD_PID"
 trap 'exit 143' TERM
 while :; do :; done
fi
name=${url##*/}
case "${MOCK_FAIL:-}:$name" in manifest:SHA256SUMS|binary:service-unlock_*) printf 'HTTP failure' > "$out"; exit 22;; esac
cp "$MOCK_RELEASE/$name" "$out"
`)
	}
	f.env = append(os.Environ(), "PATH="+f.bin, "TMPDIR="+f.tmp, "MOCK_ARCH="+arch, "MOCK_RELEASE="+f.release, "MOCK_LOG="+f.log, "FIXTURE_MARKER="+f.marker, "FIXTURE_PID="+filepath.Join(f.root, "probe.pid"), "DOWNLOAD_PID="+filepath.Join(f.root, "download.pid"))
	f.setBinary(t, `#!/bin/sh
printf '%s\n' "$@"
printf 'ran' > "$FIXTURE_MARKER"
exit "${FIXTURE_EXIT:-0}"
`)
	writeFixture(t, filepath.Join(f.root, "sentinel"), "outside temporary download directory")
	return f
}
func (f *launchFixture) setBinary(t *testing.T, body string) {
	t.Helper()
	var sums strings.Builder
	for _, arch := range []string{"amd64", "arm64"} {
		name := "service-unlock_linux_" + arch
		writeFixture(t, filepath.Join(f.release, name), body)
		fmt.Fprintf(&sums, "%x  %s\n", sha256.Sum256([]byte(body)), name)
	}
	writeFixture(t, filepath.Join(f.release, "SHA256SUMS"), sums.String())
}
func (f *launchFixture) run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command("/bin/sh", append([]string{f.script}, args...)...)
	cmd.Env = f.env
	out, errout := new(bytes.Buffer), new(bytes.Buffer)
	cmd.Stdout = out
	cmd.Stderr = errout
	err := cmd.Run()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	f.clean(t)
	return code, out.String(), errout.String()
}
func (f *launchFixture) clean(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(f.tmp)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary download directory not cleaned: %v %v", entries, err)
	}
	if _, err := os.Stat(filepath.Join(f.root, "sentinel")); err != nil {
		t.Fatal("unowned file removed", err)
	}
}
func (f *launchFixture) notExecuted(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(f.marker); !os.IsNotExist(err) {
		t.Fatal("unverified executable ran")
	}
}
func TestLauncherArchitecturesArgumentsExitAndURLs(t *testing.T) {
	for _, arch := range []string{"x86_64", "amd64", "aarch64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			f := newLaunchFixture(t, arch, "curl", "sha256sum")
			f.env = append(f.env, "FIXTURE_EXIT=7")
			code, out, errout := f.run(t, "--json", "hello world", "--workers", "2")
			if code != 7 || out != "--json\nhello world\n--workers\n2\n" || errout != "" {
				t.Fatalf("code=%d out=%q err=%q", code, out, errout)
			}
			target := "amd64"
			if arch == "aarch64" || arch == "arm64" {
				target = "arm64"
			}
			urls, _ := os.ReadFile(f.log)
			want := "https://github.com/shui1iao/service-unlock/releases/download/v0.3.1/"
			if string(urls) != want+"SHA256SUMS\n"+want+"service-unlock_linux_"+target+"\n" {
				t.Fatal(string(urls))
			}
		})
	}
}
func TestLauncherWgetAndChecksumFallbacks(t *testing.T) {
	for _, verifier := range []string{"sha256sum", "shasum", "openssl"} {
		t.Run(verifier, func(t *testing.T) {
			f := newLaunchFixture(t, "amd64", "wget", verifier)
			code, _, errout := f.run(t, "--json")
			if code != 0 {
				t.Fatal(code, errout)
			}
		})
	}
}
func TestLauncherUnsupportedAndMissingTools(t *testing.T) {
	for _, mode := range []string{"arch", "os", "downloader", "checksum"} {
		t.Run(mode, func(t *testing.T) {
			arch, downloader, verifier := "x86_64", "curl", "sha256sum"
			if mode == "arch" {
				arch = "riscv64"
			}
			if mode == "downloader" {
				downloader = "none"
			}
			if mode == "checksum" {
				verifier = "none"
			}
			f := newLaunchFixture(t, arch, downloader, verifier)
			if mode == "os" {
				f.env = append(f.env, "MOCK_OS=Darwin")
			}
			code, out, errout := f.run(t)
			if code == 0 || out != "" || errout == "" {
				t.Fatal(code, out, errout)
			}
			f.notExecuted(t)
		})
	}
}
func TestLauncherFailureRefusesExecutionAndCleans(t *testing.T) {
	for _, mode := range []string{"manifest-download", "binary-download", "hash", "missing-entry", "duplicate-entry", "malformed-hash", "checksum-tool"} {
		t.Run(mode, func(t *testing.T) {
			f := newLaunchFixture(t, "amd64", "curl", "sha256sum")
			sumFile := filepath.Join(f.release, "SHA256SUMS")
			switch mode {
			case "manifest-download":
				f.env = append(f.env, "MOCK_FAIL=manifest")
			case "binary-download":
				f.env = append(f.env, "MOCK_FAIL=binary")
			case "hash":
				writeFixture(t, sumFile, strings.Repeat("0", 64)+"  service-unlock_linux_amd64\n")
			case "missing-entry":
				writeFixture(t, sumFile, strings.Repeat("0", 64)+"  service-unlock_linux_arm64\n")
			case "duplicate-entry":
				data, _ := os.ReadFile(sumFile)
				writeFixture(t, sumFile, string(data)+string(data))
			case "malformed-hash":
				writeFixture(t, sumFile, "not-a-hash  service-unlock_linux_amd64\n")
			case "checksum-tool":
				p := filepath.Join(f.bin, "sha256sum")
				os.Remove(p)
				writeFixture(t, p, "#!/bin/sh\nexit 1\n")
			}
			code, out, errout := f.run(t, "--json")
			if code == 0 || out != "" || errout == "" {
				t.Fatal(code, out, errout)
			}
			f.notExecuted(t)
		})
	}
}
func TestLauncherSignalStopsChildAndCleans(t *testing.T) {
	for _, mode := range []string{"probe", "download", "probe-int", "download-int"} {
		t.Run(mode, func(t *testing.T) {
			f := newLaunchFixture(t, "amd64", "curl", "sha256sum")
			pidFile := filepath.Join(f.root, "probe.pid")
			if strings.HasPrefix(mode, "probe") {
				f.setBinary(t, `#!/bin/sh
printf '%s' "$$" > "$FIXTURE_PID"
trap 'exit 143' TERM
while :; do :; done
`)
			} else {
				f.env = append(f.env, "MOCK_HANG=1")
				pidFile = filepath.Join(f.root, "download.pid")
			}
			cmd := exec.Command("/bin/sh", f.script, "--json")
			cmd.Env = f.env
			cmd.Stdout = new(bytes.Buffer)
			cmd.Stderr = new(bytes.Buffer)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			var pid int
			t.Cleanup(func() {
				cmd.Process.Kill()
				if pid > 0 {
					syscall.Kill(pid, syscall.SIGKILL)
				}
			})
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				data, _ := os.ReadFile(pidFile)
				pid, _ = strconv.Atoi(string(data))
				if pid > 0 {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
			if pid == 0 {
				t.Fatal("fixture child did not start")
			}
			sig := syscall.SIGTERM
			if strings.HasSuffix(mode, "-int") {
				sig = syscall.SIGINT
			}
			if err := cmd.Process.Signal(sig); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case err := <-done:
				exit, ok := err.(*exec.ExitError)
				if !ok || exit.ExitCode() != 128+int(sig) {
					t.Fatalf("unexpected exit: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("signal left launcher hanging")
			}
			if err := syscall.Kill(pid, 0); err != syscall.ESRCH {
				t.Fatalf("child left running: %d %v", pid, err)
			}
			pid = 0 // Already reaped: do not signal a subsequently reused PID.
			f.clean(t)
		})
	}
}
