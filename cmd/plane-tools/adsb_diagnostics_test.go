package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func resetADSBDiagnostics() {
	adsbCacheHits.Store(0)
	adsbCacheMisses.Store(0)
	adsbCacheCoalesced.Store(0)
	adsbUpstreamFetches.Store(0)
	adsbUpstreamErrors.Store(0)
	adsbLastFetchLatencyNS.Store(0)
	adsbLastSuccessUnixNS.Store(0)
	invalidateADSBCache()
}

func TestADSBDiagnosticsCountsFetchAndCacheHit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"now":1788768000,"aircraft":[{"hex":"ABC123"}]}`))
	}))
	defer server.Close()

	old := adsbReceiver
	defer func() { adsbReceiver = old; resetADSBDiagnostics() }()
	adsbReceiver = adsbConfig{BaseURL: server.URL, Timeout: time.Second}
	resetADSBDiagnostics()

	if _, _, err := fetchCachedADSBAircraft(t.Context()); err != nil { t.Fatal(err) }
	if _, _, err := fetchCachedADSBAircraft(t.Context()); err != nil { t.Fatal(err) }

	rec := httptest.NewRecorder()
	adsbDiagnosticsHandler(rec, httptest.NewRequest(http.MethodGet, "/api/v1/adsb/diagnostics", nil))
	body := rec.Body.String()
	for _, want := range []string{`"cache_hits":1`, `"cache_misses":1`, `"upstream_fetches":1`, `"upstream_errors":0`, `"last_success_at"`} {
		if !strings.Contains(body, want) { t.Fatalf("body missing %s: %s", want, body) }
	}
}

func TestADSBDiagnosticsDoesNotExposeReceiverURL(t *testing.T) {
	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	adsbReceiver = adsbConfig{BaseURL: "http://user:secret@example.invalid", Timeout: time.Second}

	rec := httptest.NewRecorder()
	adsbDiagnosticsHandler(rec, httptest.NewRequest(http.MethodGet, "/api/v1/adsb/diagnostics", nil))
	body := rec.Body.String()
	for _, forbidden := range []string{"example.invalid", "user", "secret", "base_url"} {
		if strings.Contains(body, forbidden) { t.Fatalf("diagnostics leaked %q: %s", forbidden, body) }
	}
}
