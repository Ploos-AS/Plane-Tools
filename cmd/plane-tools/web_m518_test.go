package main

import (
	"strings"
	"testing"
)

func TestWebM518OutageDiagnosticsHooks(t *testing.T) {
	body, err := webFS.ReadFile("web/adsb-diagnostics.js")
	if err != nil { t.Fatal(err) }
	text := string(body)
	for _, token := range []string{
		"current_outage",
		"last_outage",
		"formatOutageKind",
		"active outage",
		"last classified outage",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("adsb-diagnostics.js missing %q", token)
		}
	}
	for _, forbidden := range []string{"aircraft.json", "PLANE_TOOLS_ADSB_URL"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("diagnostics UI must not reference receiver source %q", forbidden)
		}
	}
}
