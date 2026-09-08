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

func appendADSBHistoryEventLocked(event adsbHistoryEvent) {
	adsbHistory.events = append(adsbHistory.events, event)
	if len(adsbHistory.events) > adsbHistoryLimit {
		adsbHistory.events = append([]adsbHistoryEvent(nil), adsbHistory.events[len(adsbHistory.events)-adsbHistoryLimit:]...)
	}
}

func appendADSBHistoryEvent(event adsbHistoryEvent) {
	adsbHistory.mu.Lock()
	appendADSBHistoryEventLocked(event)
	adsbHistory.mu.Unlock()
}

func recordADSBFetchError(code string, now time.Time) {
	adsbHistory.mu.Lock()
	if code == adsbHistory.lastFetchError {
		adsbHistory.mu.Unlock()
		return
	}
	adsbHistory.lastFetchError = code
	appendADSBHistoryEventLocked(adsbHistoryEvent{At: now.UTC().Format(time.RFC3339Nano), Type: "error", Code: code})
	adsbHistory.mu.Unlock()
}

func recordADSBFetchSuccess(now time.Time) {
	adsbHistory.mu.Lock()
	previous := adsbHistory.lastFetchError
	adsbHistory.lastFetchError = ""
	if previous != "" {
		appendADSBHistoryEventLocked(adsbHistoryEvent{At: now.UTC().Format(time.RFC3339Nano), Type: "recovery", Code: previous})
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
	appendADSBHistoryEventLocked(adsbHistoryEvent{
		At: now.UTC().Format(time.RFC3339Nano), Type: "health_transition", Code: reason, FromHealth: previous, ToHealth: health,
	})
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
