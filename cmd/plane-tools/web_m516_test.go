package main

import (
	"strings"
	"testing"
)

func TestWebM516ADSBStabilizationHooks(t *testing.T) {
	js, err := webFS.ReadFile("web/adsb-diagnostics.js")
	if err != nil {
		t.Fatal(err)
	}
	body := string(js)
	for _, want := range []string{
		"data.stabilizing",
		"unstable / stabilizing",
		"flap_recovery_remaining_seconds",
		"flap_recovery_quiet_seconds",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("adsb-diagnostics.js missing %q", want)
		}
	}
	if strings.Contains(body, "aircraft.json") || strings.Contains(body, "PLANE_TOOLS_ADSB_URL") {
		t.Fatal("stabilization diagnostics must remain local-only")
	}
}
