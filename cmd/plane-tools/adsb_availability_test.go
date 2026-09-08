package main

import (
	"testing"
	"time"
)

func TestADSBAvailabilityHealthyWindow(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 2, 0, 0, 0, time.UTC)
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-10*time.Minute))

	state := currentADSBAvailability(now)
	if !state.Known || !state.Available { t.Fatalf("state = %#v, want known and available", state) }
	if state.ObservedSeconds != 600 { t.Fatalf("observed = %d, want 600", state.ObservedSeconds) }
	if state.CurrentUptimeSeconds != 600 { t.Fatalf("uptime = %d, want 600", state.CurrentUptimeSeconds) }
	if state.AvailabilityPercent != 100 { t.Fatalf("availability = %v, want 100", state.AvailabilityPercent) }
	if state.LastOutageSeconds != 0 { t.Fatalf("last outage = %d, want 0", state.LastOutageSeconds) }
}

func TestADSBAvailabilityTracksCompletedOutage(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 2, 20, 0, 0, time.UTC)
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-10*time.Minute))
	recordADSBHealthTransition("offline", "unreachable", now.Add(-6*time.Minute))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-4*time.Minute))

	state := currentADSBAvailability(now)
	if !state.Available { t.Fatalf("state = %#v, want available", state) }
	if state.ObservedSeconds != 600 { t.Fatalf("observed = %d, want 600", state.ObservedSeconds) }
	if state.LastOutageSeconds != 120 { t.Fatalf("last outage = %d, want 120", state.LastOutageSeconds) }
	if state.CurrentUptimeSeconds != 240 { t.Fatalf("uptime = %d, want 240", state.CurrentUptimeSeconds) }
	if state.AvailabilityPercent != 80 { t.Fatalf("availability = %v, want 80", state.AvailabilityPercent) }
}

func TestADSBAvailabilityActiveOutage(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 2, 40, 0, 0, time.UTC)
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-5*time.Minute))
	recordADSBHealthTransition("degraded", "timeout", now.Add(-time.Minute))

	state := currentADSBAvailability(now)
	if !state.Known || state.Available { t.Fatalf("state = %#v, want known and unavailable", state) }
	if state.CurrentUptimeSeconds != 0 { t.Fatalf("uptime = %d, want 0 during outage", state.CurrentUptimeSeconds) }
	if state.LastOutageSeconds != 0 { t.Fatalf("completed outage = %d, want 0", state.LastOutageSeconds) }
	if state.AvailabilityPercent != 80 { t.Fatalf("availability = %v, want 80", state.AvailabilityPercent) }
}

func TestADSBAvailabilityUnknownWithoutHealthHistory(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	state := currentADSBAvailability(time.Date(2026, 9, 8, 3, 0, 0, 0, time.UTC))
	if state.Known { t.Fatalf("state = %#v, want unknown", state) }
}
