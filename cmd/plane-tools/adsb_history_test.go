package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func resetADSBHistory() {
	adsbHistory.mu.Lock()
	adsbHistory.events = nil
	adsbHistory.lastHealth = ""
	adsbHistory.lastFetchError = ""
	adsbHistory.mu.Unlock()
	resetADSBFlapping()
}

func TestADSBHistoryIsBoundedAndNewestFirst(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	base := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	for i := 0; i < adsbHistoryLimit+6; i++ {
		recordADSBFetchError(fmt.Sprintf("error_%d", i), base.Add(time.Duration(i)*time.Second))
	}

	rec := httptest.NewRecorder()
	adsbHistoryHandler(rec, httptest.NewRequest(http.MethodGet, "/api/v1/adsb/history", nil))
	var events []adsbHistoryEvent
	if err := json.Unmarshal(rec.Body.Bytes(), &events); err != nil { t.Fatal(err) }
	if len(events) != adsbHistoryLimit { t.Fatalf("history length = %d, want %d", len(events), adsbHistoryLimit) }
	wantNewest := base.Add(time.Duration(adsbHistoryLimit+5) * time.Second).Format(time.RFC3339Nano)
	if events[0].At != wantNewest { t.Fatalf("newest event = %s, want %s", events[0].At, wantNewest) }
}

func TestADSBHistoryDeduplicatesRepeatedFetchErrorsUntilRecovery(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 0, 5, 0, 0, time.UTC)

	recordADSBFetchError("timeout", now)
	recordADSBFetchError("timeout", now.Add(time.Second))
	recordADSBFetchError("timeout", now.Add(2*time.Second))
	recordADSBFetchSuccess(now.Add(3 * time.Second))
	recordADSBFetchError("timeout", now.Add(4*time.Second))

	adsbHistory.mu.Lock()
	events := append([]adsbHistoryEvent(nil), adsbHistory.events...)
	adsbHistory.mu.Unlock()
	if len(events) != 3 {
		t.Fatalf("events = %#v, want error + recovery + new error", events)
	}
	if events[0].Type != "error" || events[1].Type != "recovery" || events[2].Type != "error" {
		t.Fatalf("unexpected event sequence: %#v", events)
	}
	if events[0].At != now.Format(time.RFC3339Nano) {
		t.Fatalf("deduplicated error timestamp moved to %s, want first failure %s", events[0].At, now.Format(time.RFC3339Nano))
	}
}

func TestADSBHistoryRecordsRecoveryOnce(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 0, 10, 0, 0, time.UTC)
	recordADSBFetchSuccess(now)
	recordADSBFetchError("timeout", now.Add(time.Second))
	recordADSBFetchSuccess(now.Add(2 * time.Second))
	recordADSBFetchSuccess(now.Add(3 * time.Second))

	adsbHistory.mu.Lock()
	events := append([]adsbHistoryEvent(nil), adsbHistory.events...)
	adsbHistory.mu.Unlock()
	if len(events) != 2 { t.Fatalf("events = %#v, want error + recovery", events) }
	if events[0].Type != "error" || events[0].Code != "timeout" { t.Fatalf("first event = %#v", events[0]) }
	if events[1].Type != "recovery" || events[1].Code != "timeout" { t.Fatalf("second event = %#v", events[1]) }
}

func TestADSBHistoryDeduplicatesHealthTransitions(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 0, 20, 0, 0, time.UTC)
	recordADSBHealthTransition("healthy", "feed_fresh", now)
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(time.Second))
	recordADSBHealthTransition("degraded", "timeout", now.Add(2*time.Second))

	adsbHistory.mu.Lock()
	events := append([]adsbHistoryEvent(nil), adsbHistory.events...)
	adsbHistory.mu.Unlock()
	if len(events) != 2 { t.Fatalf("events = %#v, want 2 transitions", events) }
	if events[1].FromHealth != "healthy" || events[1].ToHealth != "degraded" || events[1].Code != "timeout" {
		t.Fatalf("transition = %#v", events[1])
	}
}
