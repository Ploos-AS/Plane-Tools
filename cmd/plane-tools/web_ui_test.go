package main

import (
	"strings"
	"testing"
)

func TestWebUIAirportAndLocationHooks(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	app, err := webFS.ReadFile("web/app.js")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"airport-search-form", "nearby-form", "location-form", "location-list", "log-location"} {
		if !strings.Contains(string(index), want) {
			t.Fatalf("index.html missing %q", want)
		}
	}
	for _, want := range []string{"/api/v1/airports/search", "/api/v1/airports/nearby", "/api/v1/spotting-locations"} {
		if !strings.Contains(string(app), want) {
			t.Fatalf("app.js missing API hook %q", want)
		}
	}
}

func TestWebUILiveADSBHooks(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	app, err := webFS.ReadFile("web/app.js")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"refresh-adsb", "adsb-status", "adsb-auto-refresh", "adsb-aircraft", "add-sighting-card"} {
		if !strings.Contains(string(index), want) {
			t.Fatalf("index.html missing %q", want)
		}
	}
	for _, want := range []string{"/api/v1/adsb/status", "/api/v1/adsb/aircraft", "add-live-sighting", "setInterval(refreshADSB, 5000)"} {
		if !strings.Contains(string(app), want) {
			t.Fatalf("app.js missing live ADS-B hook %q", want)
		}
	}
}
