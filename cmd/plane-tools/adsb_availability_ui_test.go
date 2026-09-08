package main

import (
	"os"
	"strings"
	"testing"
)

func TestWebM517AvailabilityHooks(t *testing.T) {
	body, err := os.ReadFile("web/adsb-diagnostics.js")
	if err != nil { t.Fatal(err) }
	text := string(body)
	for _, want := range []string{
		"availability_percent",
		"availability_window_seconds",
		"current_uptime_seconds",
		"last_outage_seconds",
		"Availability",
		"Current uptime",
	} {
		if !strings.Contains(text, want) { t.Fatalf("diagnostics UI missing %q", want) }
	}
	if strings.Contains(text, "aircraft.json") || strings.Contains(text, "PLANE_TOOLS_ADSB_URL") {
		t.Fatal("availability UI must not contact or expose the receiver directly")
	}
}
