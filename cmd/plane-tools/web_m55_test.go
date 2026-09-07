package main

import (
	"strings"
	"testing"
)

func TestWebUIRadarTrailHooks(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	radar, err := webFS.ReadFile("web/adsb-radar.js")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"radar-trails", "radar-clear-trails", "radar-trail-age", "Plane Tools"} {
		if !strings.Contains(string(index), want) {
			t.Fatalf("index.html missing %q", want)
		}
	}
	for _, want := range []string{"radarTrailHistory", "radarTrailMaxAgeMS", "radarTrailMaxPoints", "updateRadarTrails", "drawTrails", "radarTrailHistory.clear"} {
		if !strings.Contains(string(radar), want) {
			t.Fatalf("adsb-radar.js missing %q", want)
		}
	}
}
