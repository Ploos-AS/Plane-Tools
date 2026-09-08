package main

import (
	"strings"
	"testing"
)

func TestWebM526ADSBLiveFilterDebounce(t *testing.T) {
	js, err := webFS.ReadFile("web/adsb-live.js")
	if err != nil {
		t.Fatal(err)
	}
	body := string(js)

	for _, want := range []string{
		"const adsbFilterDebounceMS = 250",
		"let adsbFilterDebounceTimer = null",
		"function cancelADSBDebouncedRefresh",
		"function scheduleADSBDebouncedRefresh",
		"setTimeout(() => {",
		"}, adsbFilterDebounceMS)",
		"function refreshADSBImmediately",
		"addEventListener(\"input\", scheduleADSBDebouncedRefresh)",
		"addEventListener(\"change\", refreshADSBImmediately)",
		"#refresh-adsb\").addEventListener(\"click\", refreshADSBImmediately)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("adsb-live.js missing debounce contract %q", want)
		}
	}

	if strings.Contains(body, "addEventListener(\"input\", refreshADSB)") {
		t.Fatal("ADS-B filter input must not refresh immediately")
	}
}
