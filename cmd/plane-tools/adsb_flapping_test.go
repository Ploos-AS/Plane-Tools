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

func TestADSBFlappingStaysLatchedDuringRecoveryQuietPeriod(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	base := time.Date(2026, 9, 8, 1, 0, 0, 0, time.UTC)

	recordADSBHealthTransition("healthy", "feed_fresh", base.Add(-90*time.Second))
	recordADSBHealthTransition("degraded", "timeout", base.Add(-70*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", base.Add(-50*time.Second))
	recordADSBHealthTransition("offline", "unreachable", base.Add(-30*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", base.Add(-10*time.Second))
	if !currentADSBFlapping(base).Flapping { t.Fatal("expected initial flapping latch") }

	state := currentADSBFlapping(base.Add(100 * time.Second))
	if !state.Flapping || !state.Stabilizing { t.Fatalf("state = %#v, want latched stabilizing", state) }
	if state.RecoveryRemainingSeconds != 10 { t.Fatalf("recovery remaining = %d, want 10", state.RecoveryRemainingSeconds) }
}

func TestADSBFlappingRecoveryTimerResetsOnNewBoundary(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	base := time.Date(2026, 9, 8, 1, 10, 0, 0, time.UTC)

	recordADSBHealthTransition("healthy", "feed_fresh", base.Add(-90*time.Second))
	recordADSBHealthTransition("degraded", "timeout", base.Add(-70*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", base.Add(-50*time.Second))
	recordADSBHealthTransition("offline", "unreachable", base.Add(-30*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", base.Add(-10*time.Second))
	if !currentADSBFlapping(base).Flapping { t.Fatal("expected initial flapping latch") }

	recordADSBHealthTransition("degraded", "timeout", base.Add(90*time.Second))
	state := currentADSBFlapping(base.Add(100 * time.Second))
	if !state.Flapping || !state.Stabilizing { t.Fatalf("state = %#v, want stabilizing", state) }
	if state.RecoveryRemainingSeconds != 110 { t.Fatalf("recovery remaining = %d, want 110", state.RecoveryRemainingSeconds) }
}

func TestADSBFlappingClearsAfterFullQuietPeriod(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	base := time.Date(2026, 9, 8, 1, 20, 0, 0, time.UTC)

	recordADSBHealthTransition("healthy", "feed_fresh", base.Add(-90*time.Second))
	recordADSBHealthTransition("degraded", "timeout", base.Add(-70*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", base.Add(-50*time.Second))
	recordADSBHealthTransition("offline", "unreachable", base.Add(-30*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", base.Add(-10*time.Second))
	if !currentADSBFlapping(base).Flapping { t.Fatal("expected initial flapping latch") }

	state := currentADSBFlapping(base.Add(110 * time.Second))
	if state.Flapping { t.Fatalf("state = %#v, want stable after 120 seconds quiet", state) }
	if state.Stabilizing || state.RecoveryRemainingSeconds != 0 { t.Fatalf("state = %#v, want recovery complete", state) }
}
