package main

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
)

func TestWebM524ADSBLiveModuleName(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), `<script src="/adsb-live.js"></script>`) {
		t.Fatal("index.html must load adsb-live.js")
	}
	if strings.Contains(string(index), "adsb-m52.js") {
		t.Fatal("index.html still references historical adsb-m52.js")
	}

	if _, err := webFS.ReadFile("web/adsb-live.js"); err != nil {
		t.Fatalf("adsb-live.js missing: %v", err)
	}
	if _, err := webFS.ReadFile("web/adsb-m52.js"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("historical adsb-m52.js must be absent, err = %v", err)
	}
}
