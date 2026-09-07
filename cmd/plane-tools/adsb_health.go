package main

import "time"

const (
	adsbHealthyAge  = 15 * time.Second
	adsbDegradedAge = 60 * time.Second
)

func applyADSBHealth(status *adsbStatus, envelope adsbAircraftEnvelope, fetchErr error, now time.Time) {
	defer func() { recordADSBHealthTransition(status.Health, status.HealthReason, now) }()

	if !status.Configured {
		status.Health = "offline"
		status.HealthReason = "not_configured"
		return
	}

	if fetchErr != nil {
		lastSuccessNS := adsbLastSuccessUnixNS.Load()
		if lastSuccessNS <= 0 {
			status.Health = "offline"
			status.HealthReason = adsbErrorCode(fetchErr)
			return
		}
		age := now.Sub(time.Unix(0, lastSuccessNS))
		if age < 0 {
			age = 0
		}
		status.LastSuccessAgeSeconds = round(age.Seconds(), 1)
		if age <= adsbDegradedAge {
			status.Health = "degraded"
			status.HealthReason = adsbErrorCode(fetchErr)
			return
		}
		status.Health = "stale"
		status.HealthReason = adsbErrorCode(fetchErr)
		return
	}

	if envelope.Now <= 0 {
		status.Health = "degraded"
		status.HealthReason = "feed_timestamp_missing"
		return
	}

	feedAt := time.Unix(int64(envelope.Now), 0)
	age := now.Sub(feedAt)
	if age < 0 {
		age = 0
	}
	status.FeedAgeSeconds = round(age.Seconds(), 1)
	switch {
	case age <= adsbHealthyAge:
		status.Health = "healthy"
		status.HealthReason = "feed_fresh"
	case age <= adsbDegradedAge:
		status.Health = "degraded"
		status.HealthReason = "feed_delayed"
	default:
		status.Health = "stale"
		status.HealthReason = "feed_stale"
	}
}
