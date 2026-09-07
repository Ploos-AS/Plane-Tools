package main

import (
	"context"
	"sync"
	"time"
)

const adsbCacheTTL = time.Second

type adsbCacheState struct {
	mu        sync.Mutex
	items     []adsbAircraft
	envelope  adsbAircraftEnvelope
	fetchedAt time.Time
	valid     bool
	inflight  chan struct{}
}

var adsbCache adsbCacheState

func invalidateADSBCache() {
	adsbCache.mu.Lock()
	adsbCache.items = nil
	adsbCache.envelope = adsbAircraftEnvelope{}
	adsbCache.fetchedAt = time.Time{}
	adsbCache.valid = false
	adsbCache.mu.Unlock()
}

func fetchCachedADSBAircraft(ctx context.Context) ([]adsbAircraft, adsbAircraftEnvelope, error) {
	for {
		adsbCache.mu.Lock()
		if adsbCache.valid && time.Since(adsbCache.fetchedAt) < adsbCacheTTL {
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
		if err == nil {
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
