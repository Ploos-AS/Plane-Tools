package main

import (
	"errors"
	"testing"
	"time"
)

func TestApplyADSBHealthFromFeedAge(t *testing.T) {
	now := time.Unix(2000, 0)
	cases := []struct {
		name string
		age  time.Duration
		want string
	}{
		{"fresh", 5 * time.Second, "healthy"},
		{"delayed", 30 * time.Second, "degraded"},
		{"stale", 90 * time.Second, "stale"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := adsbStatus{Configured: true, Reachable: true}
			envelope := adsbAircraftEnvelope{Now: float64(now.Add(-tc.age).Unix())}
			applyADSBHealth(&status, envelope, nil, now)
			if status.Health != tc.want {
				t.Fatalf("health = %q, want %q", status.Health, tc.want)
			}
		})
	}
}

func TestApplyADSBHealthOfflineWithoutSuccess(t *testing.T) {
	old := adsbLastSuccessUnixNS.Load()
	defer adsbLastSuccessUnixNS.Store(old)
	adsbLastSuccessUnixNS.Store(0)
	status := adsbStatus{Configured: true}
	applyADSBHealth(&status, adsbAircraftEnvelope{}, newADSBFetchError("timeout", "timeout"), time.Now())
	if status.Health != "offline" || status.HealthReason != "timeout" {
		t.Fatalf("status = %#v", status)
	}
}

func TestApplyADSBHealthDisabled(t *testing.T) {
	status := adsbStatus{}
	applyADSBHealth(&status, adsbAircraftEnvelope{}, errors.New("ignored"), time.Now())
	if status.Health != "offline" || status.HealthReason != "not_configured" {
		t.Fatalf("status = %#v", status)
	}
}
