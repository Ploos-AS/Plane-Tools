package main

import "time"

type adsbAvailabilityState struct {
	Known                bool
	Available            bool
	ObservedSeconds      int64
	AvailabilityPercent  float64
	CurrentUptimeSeconds int64
	LastOutageSeconds    int64
}

type timedHealthTransition struct {
	at       time.Time
	from     string
	to       string
}

func currentADSBAvailability(now time.Time) adsbAvailabilityState {
	adsbHistory.mu.Lock()
	events := append([]adsbHistoryEvent(nil), adsbHistory.events...)
	adsbHistory.mu.Unlock()

	transitions := make([]timedHealthTransition, 0, len(events))
	for _, event := range events {
		if event.Type != "health_transition" || event.ToHealth == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, event.At)
		if err != nil || at.After(now) {
			continue
		}
		transitions = append(transitions, timedHealthTransition{at: at, from: event.FromHealth, to: event.ToHealth})
	}
	if len(transitions) == 0 {
		return adsbAvailabilityState{}
	}

	start := transitions[0].at
	state := transitions[0].to
	cursor := start
	var healthyFor time.Duration
	var outageStarted time.Time
	var lastOutage time.Duration
	if state != "healthy" {
		outageStarted = start
	}

	for _, transition := range transitions[1:] {
		if transition.at.Before(cursor) {
			continue
		}
		if state == "healthy" {
			healthyFor += transition.at.Sub(cursor)
		}
		wasHealthy := state == "healthy"
		state = transition.to
		isHealthy := state == "healthy"
		if wasHealthy && !isHealthy {
			outageStarted = transition.at
		} else if !wasHealthy && isHealthy && !outageStarted.IsZero() {
			lastOutage = transition.at.Sub(outageStarted)
			outageStarted = time.Time{}
		}
		cursor = transition.at
	}

	if now.Before(cursor) {
		now = cursor
	}
	if state == "healthy" {
		healthyFor += now.Sub(cursor)
	}
	observed := now.Sub(start)
	result := adsbAvailabilityState{
		Known:               true,
		Available:           state == "healthy",
		ObservedSeconds:     int64(observed / time.Second),
		LastOutageSeconds:   int64(lastOutage / time.Second),
	}
	if observed > 0 {
		result.AvailabilityPercent = round(healthyFor.Seconds()/observed.Seconds()*100, 2)
	} else if state == "healthy" {
		result.AvailabilityPercent = 100
	}
	if state == "healthy" {
		for i := len(transitions) - 1; i >= 0; i-- {
			if transitions[i].to == "healthy" {
				result.CurrentUptimeSeconds = int64(now.Sub(transitions[i].at) / time.Second)
				break
			}
		}
	}
	return result
}
