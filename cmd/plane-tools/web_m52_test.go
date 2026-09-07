package main

import (
	"strings"
	"testing"
)

func TestWebUIM52ADSBFIlterHooks(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil { t.Fatal(err) }
	script, err := webFS.ReadFile("web/adsb-m52.js")
	if err != nil { t.Fatal(err) }
	for _, want := range []string{"adsb-filter-form", "max_distance_nm", "min_altitude_ft", "max_altitude_ft", "/adsb-m52.js"} {
		if !strings.Contains(string(index), want) { t.Fatalf("index.html missing %q", want) }
	}
	for _, want := range []string{"adsbFilterQuery", "distance_nm", "bearing_deg", "PLANE_TOOLS_ADSB_LAT/LON", "renderADSBAircraft"} {
		if !strings.Contains(string(script), want) { t.Fatalf("adsb-m52.js missing %q", want) }
	}
}
