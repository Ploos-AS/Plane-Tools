package main

import (
	"testing"
	"time"
)

func TestADSBOutageClassifiesBlip(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 2, 10, 0, 0, time.UTC)
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-30*time.Second))
	recordADSBHealthTransition("degraded", "timeout", now.Add(-20*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-10*time.Second))

	state := currentADSBOutages(now)
	if !state.Last.Known || state.Last.Kind != "blip" || state.Last.Code != "timeout" || state.Last.DurationSeconds != 10 {
		t.Fatalf("last outage = %#v", state.Last)
	}
}

func TestADSBOutageClassifiesTimeoutPeriod(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 2, 20, 0, 0, time.UTC)
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-60*time.Second))
	recordADSBHealthTransition("degraded", "timeout", now.Add(-50*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-20*time.Second))

	state := currentADSBOutages(now)
	if state.Last.Kind != "timeout" || state.Last.DurationSeconds != 30 {
		t.Fatalf("last outage = %#v", state.Last)
	}
}

func TestADSBOutageClassifiesUpstreamStatus(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 2, 30, 0, 0, time.UTC)
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-60*time.Second))
	recordADSBHealthTransition("degraded", "upstream_status", now.Add(-50*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-20*time.Second))

	state := currentADSBOutages(now)
	if state.Last.Kind != "upstream_status" || state.Last.Code != "upstream_status" {
		t.Fatalf("last outage = %#v", state.Last)
	}
}

func TestADSBOutageClassifiesLongReceiverOutage(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 2, 40, 0, 0, time.UTC)
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-2*time.Minute))
	recordADSBHealthTransition("offline", "unreachable", now.Add(-110*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-20*time.Second))

	state := currentADSBOutages(now)
	if state.Last.Kind != "receiver_outage" || state.Last.DurationSeconds != 90 || state.Last.Code != "unreachable" {
		t.Fatalf("last outage = %#v", state.Last)
	}
}

func TestADSBOutageReportsActiveOutage(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 2, 50, 0, 0, time.UTC)
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-60*time.Second))
	recordADSBHealthTransition("degraded", "timeout", now.Add(-40*time.Second))

	state := currentADSBOutages(now)
	if !state.Current.Known || !state.Current.Active || state.Current.Kind != "timeout" || state.Current.DurationSeconds != 40 {
		t.Fatalf("current outage = %#v", state.Current)
	}
	if state.Last.Known {
		t.Fatalf("unexpected completed outage = %#v", state.Last)
	}
}
