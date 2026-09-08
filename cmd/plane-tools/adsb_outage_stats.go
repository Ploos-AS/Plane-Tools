package main

import "time"

type adsbOutageClassCounts struct {
	Blip           int `json:"blip"`
	Timeout        int `json:"timeout"`
	UpstreamStatus int `json:"upstream_status"`
	ReceiverIssue  int `json:"receiver_issue"`
	ReceiverOutage int `json:"receiver_outage"`
}

type adsbOutageStatistics struct {
	Known                       bool                   `json:"known"`
	Completed                   int                    `json:"completed"`
	ByClass                     adsbOutageClassCounts  `json:"by_class"`
	LongestSeconds              int64                  `json:"longest_seconds,omitempty"`
	MeanSeconds                 float64                `json:"mean_seconds,omitempty"`
	TimeSinceLastOutageSeconds  int64                  `json:"time_since_last_outage_seconds,omitempty"`
}

type completedADSBOutage struct {
	kind     string
	duration time.Duration
	endedAt  time.Time
}

func completedADSBOutages(now time.Time) []completedADSBOutage {
	adsbHistory.mu.Lock()
	events := append([]adsbHistoryEvent(nil), adsbHistory.events...)
	adsbHistory.mu.Unlock()

	var outages []completedADSBOutage
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
			kind, _ := classifyADSBOutage(duration, codes)
			outages = append(outages, completedADSBOutage{kind: kind, duration: duration, endedAt: at})
			started = time.Time{}
			codes = nil
			inOutage = false
		}
	}
	return outages
}

func currentADSBOutageStatistics(now time.Time) adsbOutageStatistics {
	outages := completedADSBOutages(now)
	if len(outages) == 0 {
		return adsbOutageStatistics{}
	}

	result := adsbOutageStatistics{Known: true, Completed: len(outages)}
	var total time.Duration
	var longest time.Duration
	for _, outage := range outages {
		total += outage.duration
		if outage.duration > longest {
			longest = outage.duration
		}
		switch outage.kind {
		case "blip":
			result.ByClass.Blip++
		case "timeout":
			result.ByClass.Timeout++
		case "upstream_status":
			result.ByClass.UpstreamStatus++
		case "receiver_outage":
			result.ByClass.ReceiverOutage++
		default:
			result.ByClass.ReceiverIssue++
		}
	}
	result.LongestSeconds = int64(longest / time.Second)
	result.MeanSeconds = round(total.Seconds()/float64(len(outages)), 2)
	lastEnded := outages[len(outages)-1].endedAt
	if now.After(lastEnded) {
		result.TimeSinceLastOutageSeconds = int64(now.Sub(lastEnded) / time.Second)
	}
	return result
}
