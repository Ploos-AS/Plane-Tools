package main

import (
	"strings"
	"testing"
)

func TestWebUIRadarHooks(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	radar, err := webFS.ReadFile("web/adsb-radar.js")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"adsb-radar", "radar-range", "radar-status", "/adsb-radar.js", "Plane Tools M5."} {
		if !strings.Contains(string(index), want) {
			t.Fatalf("index.html missing %q", want)
		}
	}
	for _, want := range []string{"distance_nm", "bearing_deg", "track_deg", "drawRadar", "refreshADSB"} {
		if !strings.Contains(string(radar), want) {
			t.Fatalf("adsb-radar.js missing %q", want)
		}
	}
}
