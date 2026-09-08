package main

import (
	"strings"
	"testing"
)

func TestWebM527ADSBRefreshStateUX(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	live, err := webFS.ReadFile("web/adsb-live.js")
	if err != nil {
		t.Fatal(err)
	}

	indexText := string(index)
	if !strings.Contains(indexText, `id="adsb-refresh-state"`) {
		t.Fatal("index.html missing dedicated ADS-B refresh state surface")
	}
	if !strings.Contains(indexText, `id="adsb-status"`) {
		t.Fatal("index.html missing receiver health surface")
	}

	liveText := string(live)
	for _, want := range []string{
		"const adsbRefreshingDelayMS = 150",
		"function renderADSBRefreshAge()",
		"function startADSBRefreshState(sequence)",
		"function finishADSBRefreshState(sequence, updated)",
		`adsbRefreshStateEl.textContent = "Refreshing…"`,
		`adsbRefreshStateEl.textContent = "Updated just now."`,
		"Updated ${ageSeconds}s ago.",
		"sequence === adsbRefreshSequence",
		"setInterval(renderADSBRefreshAge, 1000)",
	} {
		if !strings.Contains(liveText, want) {
			t.Fatalf("adsb-live.js missing refresh-state contract %q", want)
		}
	}
}
