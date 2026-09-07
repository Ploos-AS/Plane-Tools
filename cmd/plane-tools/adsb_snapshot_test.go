package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestADSBSnapshotUsesSingleUpstreamFetch(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/data/aircraft.json" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"now":1788768000,"messages":77,"aircraft":[{"hex":"4787a2","flight":"SAS123","r":"LN-AAA","alt_baro":12000},{"hex":"4787b2","flight":"WIF456","r":"LN-BBB","alt_baro":3000}]}`))
	}))
	defer server.Close()

	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	adsbReceiver = adsbConfig{BaseURL: server.URL, Timeout: time.Second}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/adsb/snapshot?q=SAS&min_altitude_ft=10000", nil)
	rec := httptest.NewRecorder()
	adsbSnapshotHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("upstream requests = %d, want 1", got)
	}

	var snapshot adsbSnapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if !snapshot.Status.Reachable || snapshot.Status.Aircraft != 2 || snapshot.Status.Messages != 77 {
		t.Fatalf("status = %#v", snapshot.Status)
	}
	if len(snapshot.Aircraft) != 1 || snapshot.Aircraft[0].Hex != "4787A2" {
		t.Fatalf("filtered aircraft = %#v", snapshot.Aircraft)
	}
}

func TestADSBSnapshotDisabledDoesNotFetch(t *testing.T) {
	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	adsbReceiver = adsbConfig{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/adsb/snapshot", nil)
	rec := httptest.NewRecorder()
	adsbSnapshotHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var snapshot adsbSnapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Status.Configured || snapshot.Status.Reachable || len(snapshot.Aircraft) != 0 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}
