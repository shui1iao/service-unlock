package main

import (
	"bytes"
	"context"
	"encoding/json"
	"serviceunlock/internal/access"
	"serviceunlock/internal/ai"
	"serviceunlock/internal/extra"
	"serviceunlock/internal/geo"
	"strings"
	"testing"
	"time"
)

func TestAppJCOMRemovedWithoutRemovingChatGPTWebRegionOrOpenAI(t *testing.T) {
	if len(access.Checks()) != 6 {
		t.Fatalf("access should contain6, got%d", len(access.Checks()))
	}
	web := false
	for _, c := range access.Checks() {
		if c.ID == "chatgpt_app" || strings.Contains(strings.ToLower(c.ID+c.Name), "notebook") {
			t.Fatal("removed access registered", c.ID)
		}
		if c.ID == "chatgpt_web" {
			web = true
		}
	}
	region := false
	for _, c := range geo.ServiceChecks() {
		if c.ID == "chatgpt" {
			region = true
		}
	}
	api := false
	for _, c := range ai.Checks() {
		if c.ID == "openai" {
			api = true
		}
	}
	if !web || !region || !api {
		t.Fatal("other ChatGPT/OpenAI capabilities removed")
	}
	stack, trace := mockCompleteStack(t)
	m, err := runGroups(context.Background(), stack, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Access) != 6 {
		t.Fatal("actual request catalogue")
	}
	for endpoint := range trace.seen {
		if strings.Contains(endpoint, "ios.chat.openai.com") || strings.Contains(endpoint, "auth.id2.zaq.ne.jp") || strings.Contains(endpoint, "notebook") {
			t.Fatal("removed endpoint requested", endpoint)
		}
	}
	counts := map[string]int{"global": 41, "tw": 52, "hk": 49, "jp": 67, "na": 78, "sa": 44, "eu": 71, "oc": 60, "kr": 49, "sea": 52, "af": 44}
	if len(extra.Catalogue()) != 134 {
		t.Fatal("extra catalogue")
	}
	for _, r := range extra.Catalogue() {
		if r.ID == "MediaUnlockTest_JCOM_ON_DEMAND" {
			t.Fatal("removed JCOM registered")
		}
	}
	for _, r := range extra.Regions() {
		checks, e := extra.Selected(r)
		if e != nil {
			t.Fatal(e)
		}
		f := unifiedFixture(r, true)
		var out, stderr bytes.Buffer
		code := runCLI(context.Background(), []string{"--machine", "--region", r, "--extra-family", "4"}, &out, &stderr, func(context.Context, time.Duration, int) (measurements, error) { return f, nil })
		if code != 1 {
			t.Fatal(code, stderr.String())
		}
		text := out.String()
		for _, removed := range []string{"chatgpt_app", "ChatGPT (app)", "MediaUnlockTest_JCOM_ON_DEMAND", "J:com On Demand", "notebooklm_access"} {
			if strings.Contains(text, removed) {
				t.Fatal("removed output", r, removed)
			}
		}
		var doc struct {
			Report report `json:"report"`
		}
		if e = json.Unmarshal(out.Bytes(), &doc); e != nil {
			t.Fatal(e)
		}
		if len(doc.Report.Results) != 6 || len(doc.Report.Results)+len(doc.Report.Services)+len(doc.Report.AIEndpoints)+len(doc.Report.ExtraChecks) != counts[r] || len(checks) != len(f.Extra) {
			t.Fatal("derived counts", r)
		}
	}
}
