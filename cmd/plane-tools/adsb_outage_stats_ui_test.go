package main

import (
	"os"
	"strings"
	"testing"
)

func TestWebM519OutageStatisticsHooks(t *testing.T) {
	body, err := os.ReadFile("web/adsb-diagnostics.js")
	if err != nil { t.Fatal(err) }
	text := string(body)
	for _, want := range []string{
		"outage_statistics",
		"Completed outages",
		"longest_seconds",
		"mean_seconds",
		"time_since_last_outage_seconds",
		"by_class",
		"outage classes",
	} {
		if !strings.Contains(text, want) { t.Fatalf("diagnostics UI missing %q", want) }
	}
	if strings.Contains(text, "aircraft.json") || strings.Contains(text, "PLANE_TOOLS_ADSB_URL") {
		t.Fatal("outage statistics UI must not contact or expose the receiver directly")
	}
}
