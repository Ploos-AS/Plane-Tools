package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const adsbCacheTTL = time.Second

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

func fetchCachedADSBAircraft(ctx context.Context) ([]adsbAircraft, adsbAircraftEnvelope, error) {
	for {
		key := currentADSBCacheKey()
		adsbCache.mu.Lock()
		if adsbCache.valid && adsbCache.key == key && time.Since(adsbCache.fetchedAt) < adsbCacheTTL {
			items := adsbCache.items
			envelope := adsbCache.envelope
			adsbCache.mu.Unlock()
			return items, envelope, nil
		}
		if adsbCache.inflight != nil {
			inflight := adsbCache.inflight
			adsbCache.mu.Unlock()
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

		items, envelope, err := fetchADSBAircraft(ctx)

		adsbCache.mu.Lock()
		if err == nil && currentADSBCacheKey() == key {
			adsbCache.key = key
			adsbCache.items = items
			adsbCache.envelope = envelope
			adsbCache.fetchedAt = time.Now()
			adsbCache.valid = true
		}
		adsbCache.inflight = nil
		close(inflight)
		adsbCache.mu.Unlock()

		return items, envelope, err
	}
}
