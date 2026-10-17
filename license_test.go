package main

import (
	"bytes"
	"context"
	"serviceunlock/internal/render"
	"testing"
	"time"
)

func TestEmbeddedLicenseNoNetwork(t *testing.T) {
	out := new(bytes.Buffer)
	code := runCLI(context.Background(), []string{"--license"}, out, new(bytes.Buffer), func(context.Context, time.Duration, int) (measurements, error) {
		t.Fatal("license attempted measurement")
		return measurements{}, nil
	})
	if code != 0 || out.String() != licenseText+render.ThirdPartyLicenses || !bytes.Contains(out.Bytes(), []byte("Copyright (c) 2026 Remnawave")) {
		t.Fatal("missing MIT license")
	}
}
