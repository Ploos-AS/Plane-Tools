package main

import (
	"os"
	"strings"
	"testing"
)

func TestWebM522ADSBLiveHooksReplaceRadarWrappers(t *testing.T) {
	liveBody, err := os.ReadFile("web/adsb-live.js")
	if err != nil {
		t.Fatal(err)
	}
	radarBody, err := os.ReadFile("web/adsb-radar.js")
	if err != nil {
		t.Fatal(err)
	}
	live := string(liveBody)
	radar := string(radarBody)

	for _, want := range []string{
		"function setADSBLiveHooks",
		"adsbBeforeRefreshHook",
		"adsbAfterRenderHook",
		"adsbBeforeRefreshHook() === false",
		"adsbAfterRenderHook(items)",
	} {
		if !strings.Contains(live, want) {
			t.Fatalf("ADS-B live module missing %q", want)
		}
	}
	for _, want := range []string{
		"setADSBLiveHooks({",
		"beforeRefresh: beforeADSBRefreshForRadar",
		"afterRender: afterADSBAircraftRenderForRadar",
		"if (!radarFreeze.checked) return true",
		"return false",
	} {
		if !strings.Contains(radar, want) {
			t.Fatalf("ADS-B radar hook integration missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"upstreamRefreshADSB",
		"previousRenderADSBAircraft",
		"refreshADSB = async function",
		"renderADSBAircraft = function",
	} {
		if strings.Contains(radar, forbidden) {
			t.Fatalf("ADS-B radar must not wrap global live functions: found %q", forbidden)
		}
	}
}
