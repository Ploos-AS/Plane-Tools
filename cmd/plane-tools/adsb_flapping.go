package main

import (
	"sync"
	"time"
)

const (
	adsbFlappingWindow        = 2 * time.Minute
	adsbFlappingThreshold     = 4
	adsbFlappingRecoveryQuiet = 2 * time.Minute
)

type adsbFlappingState struct {
	Flapping                 bool
	Transitions              int
	Stabilizing              bool
	RecoveryRemainingSeconds int64
}

type adsbFlappingLatchState struct {
	mu     sync.Mutex
	active bool
}

var adsbFlappingLatch adsbFlappingLatchState

func resetADSBFlapping() {
	adsbFlappingLatch.mu.Lock()
	adsbFlappingLatch.active = false
	adsbFlappingLatch.mu.Unlock()
}

func currentADSBFlapping(now time.Time) adsbFlappingState {
	cutoff := now.Add(-adsbFlappingWindow)
	transitions := 0
	var latestBoundary time.Time

	adsbHistory.mu.Lock()
	for _, event := range adsbHistory.events {
		if event.Type != "health_transition" || event.FromHealth == "" || event.ToHealth == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, event.At)
		if err != nil || at.After(now) {
			continue
		}
		fromHealthy := event.FromHealth == "healthy"
		toHealthy := event.ToHealth == "healthy"
		if fromHealthy == toHealthy {
			continue
		}
		if latestBoundary.IsZero() || at.After(latestBoundary) {
			latestBoundary = at
		}
		if !at.Before(cutoff) {
			transitions++
		}
	}
	adsbHistory.mu.Unlock()

	rawFlapping := transitions >= adsbFlappingThreshold
	adsbFlappingLatch.mu.Lock()
	defer adsbFlappingLatch.mu.Unlock()
	if rawFlapping {
		adsbFlappingLatch.active = true
	}

	state := adsbFlappingState{Flapping: adsbFlappingLatch.active, Transitions: transitions}
	if !adsbFlappingLatch.active || rawFlapping {
		return state
	}

	if latestBoundary.IsZero() {
		adsbFlappingLatch.active = false
		state.Flapping = false
		return state
	}
	quietFor := now.Sub(latestBoundary)
	if quietFor < 0 {
		quietFor = 0
	}
	if quietFor >= adsbFlappingRecoveryQuiet {
		adsbFlappingLatch.active = false
		state.Flapping = false
		return state
	}

	remaining := adsbFlappingRecoveryQuiet - quietFor
	state.Stabilizing = true
	state.RecoveryRemainingSeconds = int64((remaining + time.Second - 1) / time.Second)
	return state
}
