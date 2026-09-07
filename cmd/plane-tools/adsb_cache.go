package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const adsbCacheTTL = time.Second
const adsbStaleCacheMaxAge = 5 * time.Minute

type adsbCacheState struct {
	mu        sync.Mutex
	key       string
	items     []adsbAircraft
	envelope  adsbAircraftEnvelope
	fetchedAt time.Time
	valid     bool
	inflight  chan struct{}
}

var adsbCache adsbCacheState
var adsbCacheHits atomic.Uint64
var adsbCacheMisses atomic.Uint64
var adsbCacheCoalesced atomic.Uint64
var adsbUpstreamFetches atomic.Uint64
var adsbUpstreamErrors atomic.Uint64
var adsbLastFetchLatencyNS atomic.Int64
var adsbLastSuccessUnixNS atomic.Int64

func currentADSBCacheKey() string {
	lat := ""
	lon := ""
	if adsbReceiver.Latitude != nil {
		lat = fmt.Sprintf("%.9f", *adsbReceiver.Latitude)
	}
	if adsbReceiver.Longitude != nil {
		lon = fmt.Sprintf("%.9f", *adsbReceiver.Longitude)
	}
	return fmt.Sprintf("%s|%d|%s|%s", adsbReceiver.BaseURL, adsbReceiver.Timeout.Nanoseconds(), lat, lon)
}

func invalidateADSBCache() {
	adsbCache.mu.Lock()
	adsbCache.key = ""
	adsbCache.items = nil
	adsbCache.envelope = adsbAircraftEnvelope{}
	adsbCache.fetchedAt = time.Time{}
	adsbCache.valid = false
	adsbCache.mu.Unlock()
}

func staleADSBCache(now time.Time) ([]adsbAircraft, adsbAircraftEnvelope, time.Duration, bool) {
	key := currentADSBCacheKey()
	adsbCache.mu.Lock()
	defer adsbCache.mu.Unlock()
	if !adsbCache.valid || adsbCache.key != key || adsbCache.fetchedAt.IsZero() {
		return nil, adsbAircraftEnvelope{}, 0, false
	}
	age := now.Sub(adsbCache.fetchedAt)
	if age < 0 {
		age = 0
	}
	if age > adsbStaleCacheMaxAge {
		return nil, adsbAircraftEnvelope{}, age, false
	}
	return adsbCache.items, adsbCache.envelope, age, true
}

func fetchCachedADSBAircraft(ctx context.Context) ([]adsbAircraft, adsbAircraftEnvelope, error) {
	for {
		key := currentADSBCacheKey()
		adsbCache.mu.Lock()
		if adsbCache.valid && adsbCache.key == key && time.Since(adsbCache.fetchedAt) < adsbCacheTTL {
			items := adsbCache.items
			envelope := adsbCache.envelope
			adsbCache.mu.Unlock()
			adsbCacheHits.Add(1)
			return items, envelope, nil
		}
		if adsbCache.inflight != nil {
			inflight := adsbCache.inflight
			adsbCache.mu.Unlock()
			adsbCacheCoalesced.Add(1)
			select {
			case <-ctx.Done():
				return nil, adsbAircraftEnvelope{}, ctx.Err()
			case <-inflight:
				continue
			}
		}

		inflight := make(chan struct{})
		adsbCache.inflight = inflight
		adsbCache.mu.Unlock()
		adsbCacheMisses.Add(1)
		items, envelope, err := fetchADSBAircraft(ctx)
		now := time.Now()
		if err != nil {
			recordADSBFetchError(adsbErrorCode(err), now)
		} else {
			recordADSBFetchSuccess(now)
		}

		adsbCache.mu.Lock()
		if err == nil && currentADSBCacheKey() == key {
			adsbCache.key = key
			adsbCache.items = items
			adsbCache.envelope = envelope
			adsbCache.fetchedAt = now
			adsbCache.valid = true
		}
		adsbCache.inflight = nil
		close(inflight)
		adsbCache.mu.Unlock()

		return items, envelope, err
	}
}
