package main

import (
	"strings"
	"testing"
)

func TestWebM523ADSBOwnership(t *testing.T) {
	app, err := webFS.ReadFile("web/app.js")
	if err != nil {
		t.Fatal(err)
	}
	live, err := webFS.ReadFile("web/adsb-live.js")
	if err != nil {
		t.Fatal(err)
	}

	appText := string(app)
	for _, forbidden := range []string{
		"const adsbStatusEl",
		"const adsbAircraftEl",
		"const adsbAutoRefresh",
		"function formatLiveValue",
		"function renderADSBAircraft",
		"function refreshADSB",
		"function scheduleADSBRefresh",
		"/api/v1/adsb/status",
		"/api/v1/adsb/aircraft",
	} {
		if strings.Contains(appText, forbidden) {
			t.Fatalf("app.js still contains legacy ADS-B ownership %q", forbidden)
		}
	}

	liveText := string(live)
	for _, required := range []string{
		"const adsbStatusEl",
		"const adsbAircraftEl",
		"const adsbAutoRefresh",
		"function formatLiveValue",
		"function renderADSBAircraft",
		"async function refreshADSB",
		"function scheduleADSBRefresh",
		"/api/v1/adsb/snapshot",
		"setADSBLiveHooks",
	} {
		if !strings.Contains(liveText, required) {
			t.Fatalf("adsb-live.js missing live ADS-B ownership %q", required)
		}
	}
}
