package main

import (
	"strings"
	"testing"
)

func TestWebM513ADSBDiagnosticsHooks(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil { t.Fatal(err) }
	js, err := webFS.ReadFile("web/adsb-diagnostics.js")
	if err != nil { t.Fatal(err) }

	for _, want := range []string{"adsb-diagnostics", "refresh-adsb-diagnostics", "/adsb-diagnostics.js", "ADS-B diagnostics"} {
		if !strings.Contains(string(index), want) { t.Fatalf("index.html missing %q", want) }
	}
	for _, want := range []string{"/api/v1/adsb/diagnostics", "Cache hit rate", "Coalesced requests", "Upstream errors", "Last latency", "setInterval(refreshADSBDiagnostics, 5000)"} {
		if !strings.Contains(string(js), want) { t.Fatalf("adsb-diagnostics.js missing %q", want) }
	}
	for _, forbidden := range []string{"PLANE_TOOLS_ADSB_URL", "/data/aircraft.json", "status.base_url"} {
		if strings.Contains(string(js), forbidden) { t.Fatalf("diagnostics UI must not contain %q", forbidden) }
	}
}
