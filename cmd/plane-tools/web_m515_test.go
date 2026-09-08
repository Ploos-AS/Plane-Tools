package main

import (
	"strings"
	"testing"
)

func TestWebM515FlappingDiagnosticsHooks(t *testing.T) {
	js, err := webFS.ReadFile("web/adsb-diagnostics.js")
	if err != nil { t.Fatal(err) }
	body := string(js)
	for _, want := range []string{"data.flapping", "unstable / flapping", "flap_transitions", "flap_window_seconds", "dataset.stability"} {
		if !strings.Contains(body, want) { t.Fatalf("adsb-diagnostics.js missing %q", want) }
	}
	for _, forbidden := range []string{"PLANE_TOOLS_ADSB_URL", "aircraft.json", "receiver_lat", "receiver_lon"} {
		if strings.Contains(body, forbidden) { t.Fatalf("adsb-diagnostics.js unexpectedly contains %q", forbidden) }
	}
}
