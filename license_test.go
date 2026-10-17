package main

import (
	"bytes"
	"context"
	"serviceunlock/internal/access"
	"testing"
	"time"
)

func TestEmbeddedLicenseNoNetwork(t *testing.T) {
	out := new(bytes.Buffer)
	code := runCLI(context.Background(), []string{"--license"}, out, new(bytes.Buffer), func(context.Context, time.Duration, int) ([]access.Result, error) {
		t.Fatal("license attempted measurement")
		return nil, nil
	})
	if code != 0 || out.String() != licenseText || !bytes.Contains(out.Bytes(), []byte("Copyright (c) 2026 Remnawave")) {
		t.Fatal("missing MIT license")
	}
}
