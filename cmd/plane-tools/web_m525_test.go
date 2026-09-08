package main

import (
	"strings"
	"testing"
)

func TestWebM525ADSBLiveRefreshConcurrency(t *testing.T) {
	body, err := webFS.ReadFile("web/adsb-live.js")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)

	for _, want := range []string{
		"let adsbRefreshSequence = 0",
		"let adsbRefreshController = null",
		"const sequence = ++adsbRefreshSequence",
		"adsbRefreshController.abort()",
		"new AbortController()",
		"{signal: controller.signal}",
		"if (sequence !== adsbRefreshSequence) return",
		"error?.name === \"AbortError\"",
		"if (sequence === adsbRefreshSequence) adsbRefreshController = null",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("adsb-live.js missing concurrency guard %q", want)
		}
	}
}
