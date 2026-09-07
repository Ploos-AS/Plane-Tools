package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestADSBCacheCoalescesConcurrentFetches(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"now":1788768000,"messages":7,"aircraft":[{"hex":"4787a2"}]}`))
	}))
	defer server.Close()

	old := adsbReceiver
	defer func() {
		adsbReceiver = old
		invalidateADSBCache()
	}()
	if err := configureADSBReceiver(server.URL, time.Second); err != nil {
		t.Fatal(err)
	}

	const readers = 12
	var wg sync.WaitGroup
	errs := make(chan error, readers)
	wg.Add(readers)
	for range readers {
		go func() {
			defer wg.Done()
			items, _, err := fetchCachedADSBAircraft(context.Background())
			if err == nil && len(items) != 1 {
				t.Errorf("items = %d, want 1", len(items))
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls = %d, want 1", got)
	}

	if _, _, err := fetchCachedADSBAircraft(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("cached upstream calls = %d, want 1", got)
	}

	adsbCache.mu.Lock()
	adsbCache.fetchedAt = time.Now().Add(-2 * adsbCacheTTL)
	adsbCache.mu.Unlock()
	if _, _, err := fetchCachedADSBAircraft(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("expired cache upstream calls = %d, want 2", got)
	}
}

func TestADSBCacheSharedAcrossReadHandlers(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"now":1788768000,"messages":11,"aircraft":[{"hex":"4787a2","flight":"SAS123"}]}`))
	}))
	defer server.Close()

	old := adsbReceiver
	defer func() {
		adsbReceiver = old
		invalidateADSBCache()
	}()
	if err := configureADSBReceiver(server.URL, time.Second); err != nil {
		t.Fatal(err)
	}

	requests := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/api/v1/adsb/status", adsbStatusHandler},
		{"/api/v1/adsb/aircraft", adsbAircraftHandler},
		{"/api/v1/adsb/aircraft/search?q=SAS", adsbAircraftSearchHandler},
		{"/api/v1/adsb/snapshot?q=SAS", adsbSnapshotHandler},
	}
	for _, tc := range requests {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		tc.handler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", tc.path, rec.Code, rec.Body.String())
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls across read handlers = %d, want 1", got)
	}
}

func TestADSBCacheDoesNotCacheErrors(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call := calls.Add(1)
		if call == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"aircraft":[{"hex":"4787a2"}]}`))
	}))
	defer server.Close()

	old := adsbReceiver
	defer func() {
		adsbReceiver = old
		invalidateADSBCache()
	}()
	if err := configureADSBReceiver(server.URL, time.Second); err != nil {
		t.Fatal(err)
	}

	if _, _, err := fetchCachedADSBAircraft(context.Background()); err == nil {
		t.Fatal("expected first fetch to fail")
	}
	items, _, err := fetchCachedADSBAircraft(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls after recovery = %d, want 2", got)
	}
}
