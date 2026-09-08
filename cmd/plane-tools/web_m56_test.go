package main

import (
	"strings"
	"testing"
)

func TestWebM56RadarUsabilityHooks(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	radar, err := webFS.ReadFile("web/adsb-radar.js")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"radar-freeze", "radar-follow-selected", "radar-trail-age", "30 seconds", "5 minutes", "Plane Tools M5.6"} {
		if !strings.Contains(string(index), want) {
			t.Fatalf("index.html missing %q", want)
		}
	}
	for _, want := range []string{"selectedAircraftGraceMS", "selectedAircraftSnapshot", "ensureSelectedInRange", "beforeADSBRefreshForRadar", "radarFreeze.checked", "radarTrailMaxAgeMS = Number(radarTrailAge.value)"} {
		if !strings.Contains(string(radar), want) {
			t.Fatalf("adsb-radar.js missing usability hook %q", want)
		}
	}
}
