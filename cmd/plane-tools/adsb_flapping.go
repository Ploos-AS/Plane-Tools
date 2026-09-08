package main

import "time"

const (
	adsbFlappingWindow    = 2 * time.Minute
	adsbFlappingThreshold = 4
)

type adsbFlappingState struct {
	Flapping    bool
	Transitions int
}

func currentADSBFlapping(now time.Time) adsbFlappingState {
	cutoff := now.Add(-adsbFlappingWindow)
	transitions := 0

	adsbHistory.mu.Lock()
	defer adsbHistory.mu.Unlock()
	for _, event := range adsbHistory.events {
		if event.Type != "health_transition" || event.FromHealth == "" || event.ToHealth == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, event.At)
		if err != nil || at.Before(cutoff) || at.After(now) {
			continue
		}
		fromHealthy := event.FromHealth == "healthy"
		toHealthy := event.ToHealth == "healthy"
		if fromHealthy != toHealthy {
			transitions++
		}
	}

	return adsbFlappingState{
		Flapping:    transitions >= adsbFlappingThreshold,
		Transitions: transitions,
	}
}
