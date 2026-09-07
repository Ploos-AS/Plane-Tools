package main

import (
	"strings"
	"testing"
)

func TestWebM514ADSBHistoryHooks(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil { t.Fatal(err) }
	js, err := webFS.ReadFile("web/adsb-diagnostics.js")
	if err != nil { t.Fatal(err) }

	indexText := string(index)
	jsText := string(js)
	for _, want := range []string{"adsb-history", "Receiver history"} {
		if !strings.Contains(indexText, want) { t.Fatalf("index missing %q", want) }
	}
	for _, want := range []string{"/api/v1/adsb/history", "renderADSBHistory", "health_transition", "Receiver recovered", "Receiver error"} {
		if !strings.Contains(jsText, want) { t.Fatalf("diagnostics JS missing %q", want) }
	}
	for _, forbidden := range []string{"aircraft.json", "base_url", "receiver_lat", "receiver_lon"} {
		if strings.Contains(jsText, forbidden) { t.Fatalf("diagnostics JS must not contain %q", forbidden) }
	}
}
