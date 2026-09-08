package main

import (
	"strings"
	"testing"
)

func TestWebM57ADSBStatusHardeningHooks(t *testing.T) {
	js, err := webFS.ReadFile("web/adsb-live.js")
	if err != nil {
		t.Fatal(err)
	}
	body := string(js)
	for _, want := range []string{"status.error_code", "receiver request failed", "status.health"} {
		if !strings.Contains(body, want) {
			t.Fatalf("adsb-live.js missing %q", want)
		}
	}
	if strings.Contains(body, "status.base_url") {
		t.Fatal("adsb-live.js must not expose receiver base URL")
	}
}
