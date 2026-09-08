package main

import (
	"strings"
	"testing"
)

func TestWebM512StaleADSBSnapshotHooks(t *testing.T) {
	js, err := webFS.ReadFile("web/adsb-live.js")
	if err != nil {
		t.Fatal(err)
	}
	body := string(js)
	for _, want := range []string{"status.stale_data", "stale_data_age_seconds", "cached aircraft", "renderADSBAircraft(snapshot.aircraft"} {
		if !strings.Contains(body, want) {
			t.Fatalf("adsb-live.js missing %q", want)
		}
	}
}
