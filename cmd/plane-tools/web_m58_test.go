package main

import (
	"strings"
	"testing"
)

func TestWebM58SnapshotPollingHooks(t *testing.T) {
	js, err := webFS.ReadFile("web/adsb-m52.js")
	if err != nil {
		t.Fatal(err)
	}
	text := string(js)
	for _, want := range []string{"/api/v1/adsb/snapshot", "snapshot.status", "snapshot.aircraft", "scheduleADSBRefresh()"} {
		if !strings.Contains(text, want) {
			t.Fatalf("adsb-m52.js missing %q", want)
		}
	}
	if strings.Contains(text, `apiJSON("/api/v1/adsb/status")`) || strings.Contains(text, "/api/v1/adsb/aircraft/search${") {
		t.Fatal("ADS-B refresh still contains legacy double-fetch calls")
	}
}
