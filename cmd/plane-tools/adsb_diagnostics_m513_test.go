package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestADSBDiagnosticsHealthUsesCachedFeedWithoutUpstreamFetch(t *testing.T) {
	old := adsbReceiver
	defer func() { adsbReceiver = old; resetADSBDiagnostics() }()
	adsbReceiver = adsbConfig{BaseURL: "http://receiver.invalid", Timeout: time.Second}
	resetADSBDiagnostics()

	now := time.Now().UTC()
	key := currentADSBCacheKey()
	adsbCache.mu.Lock()
	adsbCache.key = key
	adsbCache.envelope = adsbAircraftEnvelope{Now: float64(now.Unix())}
	adsbCache.fetchedAt = now
	adsbCache.valid = true
	adsbCache.mu.Unlock()
	adsbLastSuccessUnixNS.Store(now.UnixNano())

	before := adsbUpstreamFetches.Load()
	rec := httptest.NewRecorder()
	adsbDiagnosticsHandler(rec, httptest.NewRequest(http.MethodGet, "/api/v1/adsb/diagnostics", nil))
	body := rec.Body.String()

	if after := adsbUpstreamFetches.Load(); after != before {
		t.Fatalf("diagnostics triggered upstream fetch: before=%d after=%d", before, after)
	}
	for _, want := range []string{`"configured":true`, `"health":"healthy"`, `"health_reason":"feed_fresh"`} {
		if !strings.Contains(body, want) { t.Fatalf("body missing %s: %s", want, body) }
	}
}
