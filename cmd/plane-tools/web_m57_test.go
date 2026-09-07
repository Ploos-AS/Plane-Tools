package main

import (
	"strings"
	"testing"
)

func TestWebM57ADSBStatusHardeningHooks(t *testing.T) {
	js, err := webFS.ReadFile("web/adsb-m52.js")
	if err != nil {
		t.Fatal(err)
	}
	body := string(js)
	for _, want := range []string{"status.error_code", "receiver request failed", "Receiver unavailable"} {
		if !strings.Contains(body, want) {
			t.Fatalf("adsb-m52.js missing %q", want)
		}
	}
	if strings.Contains(body, "status.base_url") {
		t.Fatal("adsb-m52.js must not expose receiver base URL")
	}
}
