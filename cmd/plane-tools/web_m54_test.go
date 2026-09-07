package main

import (
	"strings"
	"testing"
)

func TestWebM54RadarInteractionHooks(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	radar, err := webFS.ReadFile("web/adsb-radar.js")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"radar-detail", "adsb-radar", "radar-range", "Plane Tools M5."} {
		if !strings.Contains(string(index), want) {
			t.Fatalf("index.html missing %q", want)
		}
	}
	for _, want := range []string{"radarHitTargets", "selectAircraft", "radar-add-sighting", "live-aircraft-row", "prefillSighting"} {
		if !strings.Contains(string(radar), want) {
			t.Fatalf("adsb-radar.js missing interaction hook %q", want)
		}
	}
}
