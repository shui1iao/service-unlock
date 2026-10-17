// Service Unlock v0.3.0. MIT License; Copyright (c) 2026 Remnawave.
// Derived from remnawave/geocheck v0.3.0; see LICENSE and README.md.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unicode"

	"serviceunlock/internal/access"
	"serviceunlock/internal/netx"
)

//go:embed LICENSE
var licenseText string

const help = `Service Unlock v0.3.0 — 7个平台 / 8项检查
Usage: service-unlock [--json] [--timeout SECONDS] [--workers N]
  --json       只输出JSON，error=未知，不代表未解锁
  --timeout N  每次GET总预算秒数（含重定向/正文），默认6
  --workers N  有界并发1–8，默认4
  --help       显示帮助，不联网
  --license    显示内嵌MIT许可，不联网
执行检测会访问真实平台；不需要Node/Python/Go运行环境。
`

type item struct {
	ID        string  `json:"id"`
	Platform  string  `json:"platform"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Detail    string  `json:"detail"`
	Region    *string `json:"region"`
	Error     *string `json:"error"`
	CheckedAt string  `json:"checked_at"`
	ElapsedMS float64 `json:"elapsed_ms"`
}
type report struct {
	SchemaVersion int               `json:"schema_version"`
	Source        map[string]string `json:"source"`
	GeneratedAt   string            `json:"generated_at"`
	Network       networkInfo       `json:"network"`
	Results       []item            `json:"results"`
}
type networkInfo struct {
	Mode           string  `json:"mode"`
	AddressFamily  string  `json:"address_family"`
	TimeoutSeconds float64 `json:"timeout_seconds"`
	TimeoutScope   string  `json:"timeout_scope"`
	Workers        int     `json:"workers"`
}

func utc(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }
func makeReport(results []access.Result, timeout float64, workers int) report {
	r := report{SchemaVersion: 1, GeneratedAt: utc(time.Now()),
		Source: map[string]string{"repository": "https://github.com/remnawave/geocheck", "tag": "v0.3.0",
			"commit": "50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc", "port": "Go v0.3.0 standalone extraction with conservative error handling"},
		Network: networkInfo{"direct", "system-selected", timeout, "request including redirects and body", workers},
		Results: make([]item, 0, len(results)),
	}
	for _, res := range results {
		platform := res.Check.Name
		if res.Check.ID == "chatgpt_web" || res.Check.ID == "chatgpt_app" {
			platform = "ChatGPT"
		}
		out := item{ID: res.Check.ID, Platform: platform, Name: res.Check.Name, Status: res.State.String(), Detail: res.Detail,
			CheckedAt: utc(res.CheckedAt), ElapsedMS: math.Round(float64(res.RTT)/float64(time.Millisecond)*1000) / 1000}
		if res.Region != "" {
			region := res.Region
			out.Region = &region
		}
		if res.State == access.StateError {
			message := res.Detail
			if res.Err != nil {
				message = res.Err.Error()
			}
			if message == "" {
				message = "could not reach a conclusion"
			}
			out.Error = &message
		}
		r.Results = append(r.Results, out)
	}
	return r
}
func printTable(w io.Writer, r report) error {
	if _, err := fmt.Fprintln(w, "7个平台 / 8项检查 · Go独立版\n直连，系统选择地址族；error = 未知，不代表未解锁。\n服务               状态         地区   耗时(ms)  详情 / 错误"); err != nil {
		return err
	}
	for _, result := range r.Results {
		detail, region := result.Detail, "-"
		if detail == "" {
			detail = "-"
		}
		if result.Region != nil {
			region = *result.Region
		}
		if result.Error != nil && *result.Error != result.Detail {
			detail += " | " + *result.Error
		}
		detail = strings.Map(func(r rune) rune {
			if !unicode.IsPrint(r) {
				return ' '
			}
			return r
		}, detail)
		if _, err := fmt.Fprintf(w, "%-18s %-12s %-5s %9.1f  %s\n", result.Name, result.Status, region, result.ElapsedMS, detail); err != nil {
			return err
		}
	}
	return nil
}

type runner func(context.Context, time.Duration, int) ([]access.Result, error)

func realRun(ctx context.Context, timeout time.Duration, workers int) ([]access.Result, error) {
	stack, err := netx.New(ctx, netx.Options{Timeout: timeout})
	if err != nil {
		return nil, err
	}
	defer stack.Close()
	return access.Run(ctx, stack, 0, access.Checks(), workers), nil
}
func runCLI(ctx context.Context, args []string, stdout, stderr io.Writer, run runner) int {
	flags := flag.NewFlagSet("service-unlock", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	asJSON := flags.Bool("json", false, "JSON")
	showLicense := flags.Bool("license", false, "show MIT license")
	timeout := flags.Float64("timeout", 6, "per-request timeout seconds")
	workers := flags.Int("workers", 4, "concurrency")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, help)
			return 0
		}
		fmt.Fprintln(stderr, "参数错误:", err)
		return 2
	}
	if flags.NArg() != 0 || math.IsNaN(*timeout) || math.IsInf(*timeout, 0) || *timeout <= 0 || *timeout > 2147483.647 || *workers < 1 || *workers > 8 {
		fmt.Fprintln(stderr, "参数错误: timeout须在(0,2147483.647]秒，workers须在1–8，不能有位置参数")
		return 2
	}
	if *showLicense {
		fmt.Fprint(stdout, licenseText)
		return 0
	}
	duration := time.Duration(math.Ceil(*timeout*1000)) * time.Millisecond
	results, err := run(ctx, duration, *workers)
	if ctx.Err() != nil {
		fmt.Fprintln(stderr, "检测已中断")
		return 130
	}
	if err != nil {
		fmt.Fprintln(stderr, "检测失败:", err)
		return 1
	}
	r := makeReport(results, *timeout, *workers)
	if *asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		err = encoder.Encode(r)
	} else {
		err = printTable(stdout, r)
	}
	if err != nil {
		fmt.Fprintln(stderr, "输出失败:", err)
		return 1
	}
	for _, r := range results {
		if r.State == access.StateError {
			return 1
		}
	}
	return 0
}
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	var interrupted atomic.Int32
	go func() {
		select {
		case sig := <-signals:
			interrupted.Store(int32(128 + int(sig.(syscall.Signal))))
			cancel()
		case <-ctx.Done():
		}
	}()
	code := runCLI(ctx, os.Args[1:], os.Stdout, os.Stderr, realRun)
	cancel()
	signal.Stop(signals)
	if value := interrupted.Load(); value != 0 {
		code = int(value)
	}
	os.Exit(code)
}
