package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestADSBSnapshotFallsBackToRecentStaleData(t *testing.T) {
	var fail atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fail.Load() {
			http.Error(w, "receiver down", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"now":1788800400,"messages":42,"aircraft":[{"hex":"ABC123","flight":"SAS123"}]}`))
	}))
	defer server.Close()

	old := adsbReceiver
	defer func() { adsbReceiver = old; invalidateADSBCache() }()
	adsbReceiver = adsbConfig{BaseURL: server.URL, Timeout: time.Second}
	invalidateADSBCache()

	if _, _, err := fetchCachedADSBAircraft(t.Context()); err != nil {
		t.Fatal(err)
	}
	adsbCache.mu.Lock()
	adsbCache.fetchedAt = time.Now().Add(-2 * time.Second)
	adsbCache.mu.Unlock()
	fail.Store(true)

	rec := httptest.NewRecorder()
	adsbSnapshotHandler(rec, httptest.NewRequest(http.MethodGet, "/api/v1/adsb/snapshot", nil))
	body := rec.Body.String()
	for _, want := range []string{`"stale_data":true`, `"aircraft":1`, `"hex":"ABC123"`, `"error_code":"upstream_status"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("snapshot missing %s: %s", want, body)
		}
	}
}

func TestADSBSnapshotDoesNotServeExpiredStaleData(t *testing.T) {
	old := adsbReceiver
	defer func() { adsbReceiver = old; invalidateADSBCache() }()
	adsbReceiver = adsbConfig{BaseURL: "http://receiver.invalid", Timeout: time.Second}
	invalidateADSBCache()

	adsbCache.mu.Lock()
	adsbCache.key = currentADSBCacheKey()
	adsbCache.items = []adsbAircraft{{Hex: "OLD123"}}
	adsbCache.fetchedAt = time.Now().Add(-adsbStaleCacheMaxAge - time.Second)
	adsbCache.valid = true
	adsbCache.mu.Unlock()

	items, _, _, ok := staleADSBCache(time.Now())
	if ok || len(items) != 0 {
		t.Fatalf("expired stale cache was returned: %#v", items)
	}
}

func TestADSBSnapshotDoesNotCrossReceiverConfiguration(t *testing.T) {
	old := adsbReceiver
	defer func() { adsbReceiver = old; invalidateADSBCache() }()
	adsbReceiver = adsbConfig{BaseURL: "http://receiver-a.invalid", Timeout: time.Second}
	invalidateADSBCache()

	adsbCache.mu.Lock()
	adsbCache.key = currentADSBCacheKey()
	adsbCache.items = []adsbAircraft{{Hex: "AAAAAA"}}
	adsbCache.fetchedAt = time.Now().Add(-2 * time.Second)
	adsbCache.valid = true
	adsbCache.mu.Unlock()

	adsbReceiver.BaseURL = "http://receiver-b.invalid"
	if _, _, _, ok := staleADSBCache(time.Now()); ok {
		t.Fatal("stale cache crossed receiver configuration boundary")
	}
}
