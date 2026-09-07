package main

import (
	"net/http"
	"time"
)

type adsbDiagnostics struct {
	CacheTTLMS         int64  `json:"cache_ttl_ms"`
	CacheHits          uint64 `json:"cache_hits"`
	CacheMisses        uint64 `json:"cache_misses"`
	CacheCoalesced     uint64 `json:"cache_coalesced"`
	UpstreamFetches    uint64 `json:"upstream_fetches"`
	UpstreamErrors     uint64 `json:"upstream_errors"`
	LastFetchLatencyMS int64  `json:"last_fetch_latency_ms"`
	LastSuccessAt      string `json:"last_success_at,omitempty"`
}

func adsbDiagnosticsHandler(w http.ResponseWriter, _ *http.Request) {
	diagnostics := adsbDiagnostics{
		CacheTTLMS:         adsbCacheTTL.Milliseconds(),
		CacheHits:          adsbCacheHits.Load(),
		CacheMisses:        adsbCacheMisses.Load(),
		CacheCoalesced:     adsbCacheCoalesced.Load(),
		UpstreamFetches:    adsbUpstreamFetches.Load(),
		UpstreamErrors:     adsbUpstreamErrors.Load(),
		LastFetchLatencyMS: time.Duration(adsbLastFetchLatencyNS.Load()).Milliseconds(),
	}
	if unixNS := adsbLastSuccessUnixNS.Load(); unixNS > 0 {
		diagnostics.LastSuccessAt = time.Unix(0, unixNS).UTC().Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, diagnostics)
}
