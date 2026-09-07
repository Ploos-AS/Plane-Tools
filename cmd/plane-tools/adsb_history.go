package main

import (
	"net/http"
	"sync"
	"time"
)

const adsbHistoryLimit = 64

type adsbHistoryEvent struct {
	At         string `json:"at"`
	Type       string `json:"type"`
	Code       string `json:"code,omitempty"`
	FromHealth string `json:"from_health,omitempty"`
	ToHealth   string `json:"to_health,omitempty"`
}

type adsbHistoryState struct {
	mu             sync.Mutex
	events         []adsbHistoryEvent
	lastHealth     string
	lastFetchError string
}

var adsbHistory adsbHistoryState

func appendADSBHistoryEvent(event adsbHistoryEvent) {
	adsbHistory.mu.Lock()
	defer adsbHistory.mu.Unlock()
	adsbHistory.events = append(adsbHistory.events, event)
	if len(adsbHistory.events) > adsbHistoryLimit {
		copy(adsbHistory.events, adsbHistory.events[len(adsbHistory.events)-adsbHistoryLimit:])
		adsbHistory.events = adsbHistory.events[:adsbHistoryLimit]
	}
}

func recordADSBFetchError(code string, now time.Time) {
	adsbHistory.mu.Lock()
	adsbHistory.lastFetchError = code
	adsbHistory.events = append(adsbHistory.events, adsbHistoryEvent{At: now.UTC().Format(time.RFC3339Nano), Type: "error", Code: code})
	if len(adsbHistory.events) > adsbHistoryLimit {
		adsbHistory.events = append([]adsbHistoryEvent(nil), adsbHistory.events[len(adsbHistory.events)-adsbHistoryLimit:]...)
	}
	adsbHistory.mu.Unlock()
}

func recordADSBFetchSuccess(now time.Time) {
	adsbHistory.mu.Lock()
	previous := adsbHistory.lastFetchError
	adsbHistory.lastFetchError = ""
	if previous != "" {
		adsbHistory.events = append(adsbHistory.events, adsbHistoryEvent{At: now.UTC().Format(time.RFC3339Nano), Type: "recovery", Code: previous})
		if len(adsbHistory.events) > adsbHistoryLimit {
			adsbHistory.events = append([]adsbHistoryEvent(nil), adsbHistory.events[len(adsbHistory.events)-adsbHistoryLimit:]...)
		}
	}
	adsbHistory.mu.Unlock()
}

func recordADSBHealthTransition(health, reason string, now time.Time) {
	if health == "" {
		return
	}
	adsbHistory.mu.Lock()
	previous := adsbHistory.lastHealth
	if previous == health {
		adsbHistory.mu.Unlock()
		return
	}
	adsbHistory.lastHealth = health
	adsbHistory.events = append(adsbHistory.events, adsbHistoryEvent{
		At: now.UTC().Format(time.RFC3339Nano), Type: "health_transition", Code: reason, FromHealth: previous, ToHealth: health,
	})
	if len(adsbHistory.events) > adsbHistoryLimit {
		adsbHistory.events = append([]adsbHistoryEvent(nil), adsbHistory.events[len(adsbHistory.events)-adsbHistoryLimit:]...)
	}
	adsbHistory.mu.Unlock()
}

func adsbHistoryHandler(w http.ResponseWriter, _ *http.Request) {
	adsbHistory.mu.Lock()
	events := make([]adsbHistoryEvent, len(adsbHistory.events))
	for i := range adsbHistory.events {
		events[len(adsbHistory.events)-1-i] = adsbHistory.events[i]
	}
	adsbHistory.mu.Unlock()
	writeJSON(w, http.StatusOK, events)
}
