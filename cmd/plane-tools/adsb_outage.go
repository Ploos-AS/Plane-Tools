package main

import "time"

const (
	adsbOutageBlipMax     = 15 * time.Second
	adsbOutageLongMinimum = 60 * time.Second
)

type adsbOutageInfo struct {
	Known           bool
	Active          bool
	Kind            string
	Code            string
	DurationSeconds int64
	StartedAt       string
	EndedAt         string
}

type adsbOutageState struct {
	Current adsbOutageInfo
	Last    adsbOutageInfo
}

func classifyADSBOutage(duration time.Duration, codes []string) (string, string) {
	code := preferredADSBOutageCode(codes)
	switch {
	case duration < adsbOutageBlipMax:
		return "blip", code
	case duration >= adsbOutageLongMinimum:
		return "receiver_outage", code
	case code == "upstream_status":
		return "upstream_status", code
	case code == "timeout":
		return "timeout", code
	default:
		return "receiver_issue", code
	}
}

func preferredADSBOutageCode(codes []string) string {
	for _, code := range codes {
		if code == "upstream_status" {
			return code
		}
	}
	for _, code := range codes {
		if code == "timeout" {
			return code
		}
	}
	for i := len(codes) - 1; i >= 0; i-- {
		if codes[i] != "" && codes[i] != "feed_fresh" {
			return codes[i]
		}
	}
	return ""
}

func currentADSBOutages(now time.Time) adsbOutageState {
	adsbHistory.mu.Lock()
	events := append([]adsbHistoryEvent(nil), adsbHistory.events...)
	adsbHistory.mu.Unlock()

	var result adsbOutageState
	var started time.Time
	var codes []string
	inOutage := false

	for _, event := range events {
		if event.Type != "health_transition" || event.ToHealth == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, event.At)
		if err != nil || at.After(now) {
			continue
		}
		toHealthy := event.ToHealth == "healthy"
		if !inOutage && !toHealthy {
			started = at
			codes = codes[:0]
			if event.Code != "" {
				codes = append(codes, event.Code)
			}
			inOutage = true
			continue
		}
		if inOutage && !toHealthy {
			if event.Code != "" {
				codes = append(codes, event.Code)
			}
			continue
		}
		if inOutage && toHealthy {
			duration := at.Sub(started)
			if duration < 0 {
				duration = 0
			}
			kind, code := classifyADSBOutage(duration, codes)
			result.Last = adsbOutageInfo{
				Known: true, Kind: kind, Code: code, DurationSeconds: int64(duration / time.Second),
				StartedAt: started.UTC().Format(time.RFC3339Nano), EndedAt: at.UTC().Format(time.RFC3339Nano),
			}
			started = time.Time{}
			codes = nil
			inOutage = false
		}
	}

	if inOutage && !started.IsZero() {
		duration := now.Sub(started)
		if duration < 0 {
			duration = 0
		}
		kind, code := classifyADSBOutage(duration, codes)
		result.Current = adsbOutageInfo{
			Known: true, Active: true, Kind: kind, Code: code, DurationSeconds: int64(duration / time.Second),
			StartedAt: started.UTC().Format(time.RFC3339Nano),
		}
	}
	return result
}
