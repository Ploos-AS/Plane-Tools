package main

import (
	"testing"
	"time"
)

func TestADSBOutageStatisticsAggregateCompletedOutages(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 4, 0, 0, 0, time.UTC)

	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-6*time.Minute))
	recordADSBHealthTransition("degraded", "timeout", now.Add(-5*time.Minute-50*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-5*time.Minute-40*time.Second)) // 10s blip
	recordADSBHealthTransition("degraded", "timeout", now.Add(-5*time.Minute))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-4*time.Minute-40*time.Second)) // 20s timeout
	recordADSBHealthTransition("offline", "upstream_status", now.Add(-4*time.Minute))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-3*time.Minute-30*time.Second)) // 30s upstream
	recordADSBHealthTransition("offline", "unreachable", now.Add(-2*time.Minute))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-40*time.Second)) // 80s receiver outage

	stats := currentADSBOutageStatistics(now)
	if !stats.Known { t.Fatal("statistics not known") }
	if stats.Completed != 4 { t.Fatalf("completed = %d, want 4", stats.Completed) }
	if stats.ByClass.Blip != 1 || stats.ByClass.Timeout != 1 || stats.ByClass.UpstreamStatus != 1 || stats.ByClass.ReceiverOutage != 1 || stats.ByClass.ReceiverIssue != 0 {
		t.Fatalf("class counts = %#v", stats.ByClass)
	}
	if stats.LongestSeconds != 80 { t.Fatalf("longest = %d, want 80", stats.LongestSeconds) }
	if stats.MeanSeconds != 35 { t.Fatalf("mean = %v, want 35", stats.MeanSeconds) }
	if stats.TimeSinceLastOutageSeconds != 40 { t.Fatalf("since last = %d, want 40", stats.TimeSinceLastOutageSeconds) }
}

func TestADSBOutageStatisticsExcludeActiveOutage(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 4, 10, 0, 0, time.UTC)

	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-3*time.Minute))
	recordADSBHealthTransition("degraded", "timeout", now.Add(-2*time.Minute-30*time.Second))
	recordADSBHealthTransition("healthy", "feed_fresh", now.Add(-2*time.Minute)) // completed 30s
	recordADSBHealthTransition("offline", "unreachable", now.Add(-90*time.Second)) // still active

	stats := currentADSBOutageStatistics(now)
	if stats.Completed != 1 { t.Fatalf("completed = %d, want 1", stats.Completed) }
	if stats.ByClass.Timeout != 1 { t.Fatalf("timeout count = %d, want 1", stats.ByClass.Timeout) }
	if stats.LongestSeconds != 30 || stats.MeanSeconds != 30 {
		t.Fatalf("stats = %#v, active outage must not affect aggregates", stats)
	}
	if stats.TimeSinceLastOutageSeconds != 120 { t.Fatalf("since last = %d, want 120", stats.TimeSinceLastOutageSeconds) }
}

func TestADSBOutageStatisticsUnknownWithoutCompletedOutage(t *testing.T) {
	defer resetADSBHistory()
	resetADSBHistory()
	now := time.Date(2026, 9, 8, 4, 20, 0, 0, time.UTC)
	recordADSBHealthTransition("offline", "unreachable", now.Add(-30*time.Second))

	stats := currentADSBOutageStatistics(now)
	if stats.Known || stats.Completed != 0 { t.Fatalf("stats = %#v, want unknown", stats) }
}
