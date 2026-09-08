package main

import (
	"net/http"
	"time"
)

type adsbDiagnostics struct {
	Configured            bool    `json:"configured"`
	Health                string  `json:"health"`
	HealthReason          string  `json:"health_reason,omitempty"`
	Stability             string  `json:"stability"`
	Flapping              bool    `json:"flapping"`
	FlapTransitions       int     `json:"flap_transitions"`
	FlapWindowSeconds     int64   `json:"flap_window_seconds"`
	FeedAgeSeconds        float64 `json:"feed_age_seconds,omitempty"`
	LastSuccessAgeSeconds float64 `json:"last_success_age_seconds,omitempty"`
	CacheTTLMS            int64   `json:"cache_ttl_ms"`
	CacheHits             uint64  `json:"cache_hits"`
	CacheMisses           uint64  `json:"cache_misses"`
	CacheCoalesced        uint64  `json:"cache_coalesced"`
	UpstreamFetches       uint64  `json:"upstream_fetches"`
	UpstreamErrors        uint64  `json:"upstream_errors"`
	LastFetchLatencyMS    int64   `json:"last_fetch_latency_ms"`
	LastSuccessAt         string  `json:"last_success_at,omitempty"`
}

func adsbDiagnosticsHandler(w http.ResponseWriter, _ *http.Request) {
	now := time.Now()
	status := adsbStatus{Configured: adsbReceiver.BaseURL != ""}

	adsbCache.mu.Lock()
	cacheMatches := adsbCache.valid && adsbCache.key == currentADSBCacheKey()
	envelope := adsbCache.envelope
	adsbCache.mu.Unlock()

	if cacheMatches {
		status.Reachable = true
		applyADSBHealth(&status, envelope, nil, now)
	} else if !status.Configured {
		applyADSBHealth(&status, adsbAircraftEnvelope{}, nil, now)
	} else {
		status.Health = "offline"
		status.HealthReason = "no_cached_feed"
		if lastSuccessNS := adsbLastSuccessUnixNS.Load(); lastSuccessNS > 0 {
			age := now.Sub(time.Unix(0, lastSuccessNS))
			if age < 0 {
				age = 0
			}
			status.LastSuccessAgeSeconds = round(age.Seconds(), 1)
		}
	}

	flapping := currentADSBFlapping(now)
	stability := "stable"
	if flapping.Flapping {
		stability = "flapping"
	}

	diagnostics := adsbDiagnostics{
		Configured:            status.Configured,
		Health:                status.Health,
		HealthReason:          status.HealthReason,
		Stability:             stability,
		Flapping:              flapping.Flapping,
		FlapTransitions:       flapping.Transitions,
		FlapWindowSeconds:     int64(adsbFlappingWindow.Seconds()),
		FeedAgeSeconds:        status.FeedAgeSeconds,
		LastSuccessAgeSeconds: status.LastSuccessAgeSeconds,
		CacheTTLMS:            adsbCacheTTL.Milliseconds(),
		CacheHits:             adsbCacheHits.Load(),
		CacheMisses:           adsbCacheMisses.Load(),
		CacheCoalesced:        adsbCacheCoalesced.Load(),
		UpstreamFetches:       adsbUpstreamFetches.Load(),
		UpstreamErrors:        adsbUpstreamErrors.Load(),
		LastFetchLatencyMS:    time.Duration(adsbLastFetchLatencyNS.Load()).Milliseconds(),
	}
	if unixNS := adsbLastSuccessUnixNS.Load(); unixNS > 0 {
		diagnostics.LastSuccessAt = time.Unix(0, unixNS).UTC().Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, diagnostics)
}
