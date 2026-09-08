package main

import (
	"testing"
	"time"
)

func TestADSBFlappingDetectsHealthyBoundaryOscillation(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 0, 30, 0, 0, time.UTC)

	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-90*time.Second))
	recordADSBHealthTransition("degraded", "timeout", now.Add(-70*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-50*time.Second))
	recordADSBHealthTransition("offline", "unreachable", now.Add(-30*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-10*time.Second))

	state := currentADSBFlapping(now)
	if !state.Flapping { t.Fatalf("flapping = false, transitions=%d", state.Transitions) }
	if state.Transitions != adsbFlappingThreshold { t.Fatalf("transitions = %d, want %d", state.Transitions, adsbFlappingThreshold) }
}

func TestADSBFlappingIgnoresOldTransitions(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 0, 40, 0, 0, time.UTC)

	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-4*time.Minute))
	recordADSBHealthTransition("degraded", "timeout", now.Add(-3*time.Minute))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-150*time.Second))
	recordADSBHealthTransition("offline", "unreachable", now.Add(-130*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-30*time.Second))

	state := currentADSBFlapping(now)
	if state.Flapping { t.Fatalf("flapping = true, transitions=%d", state.Transitions) }
	if state.Transitions != 1 { t.Fatalf("transitions = %d, want 1", state.Transitions) }
}

func TestADSBFlappingIgnoresNonHealthyToNonHealthyTransitions(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 0, 50, 0, 0, time.UTC)

	recordADSBHealthTransition("degraded", "timeout", now.Add(-50*time.Second))
	recordADSBHealthTransition("stale", "timeout", now.Add(-40*time.Second))
	recordADSBHealthTransition("offline", "unreachable", now.Add(-30*time.Second))
	recordADSBHealthTransition("degraded", "timeout", now.Add(-20*time.Second))

	state := currentADSBFlapping(now)
	if state.Flapping || state.Transitions != 0 {
		t.Fatalf("state = %#v, want stable with zero healthy-boundary transitions", state)
	}
}
