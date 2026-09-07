package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchADSBAircraft(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/data/aircraft.json" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"now":1788768000,"messages":12345,"aircraft":[{"hex":"4787a2","flight":"SAS123 ","r":"ln-test","t":"a320","lat":58.1,"lon":8.2,"alt_baro":12000,"gs":250.5,"track":91.2,"seen":0.4,"messages":42},{"hex":"4ca001","alt_baro":"ground"}]}`))
	}))
	defer server.Close()

	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	if err := configureADSBReceiver(server.URL, time.Second); err != nil {
		t.Fatal(err)
	}
	items, envelope, err := fetchADSBAircraft(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if envelope.Messages != 12345 || len(items) != 2 {
		t.Fatalf("envelope = %#v items = %#v", envelope, items)
	}
	first := items[0]
	if first.Hex != "4787A2" || first.Flight != "SAS123" || first.Registration != "LN-TEST" || first.TypeCode != "A320" {
		t.Fatalf("normalization failed: %#v", first)
	}
	if first.Altitude == nil || *first.Altitude != 12000 || first.GroundSpeed == nil || *first.GroundSpeed != 250.5 {
		t.Fatalf("numeric fields failed: %#v", first)
	}
	if items[1].Altitude != nil {
		t.Fatalf("ground altitude should not be numeric: %#v", items[1])
	}
}

func TestConfigureADSBReceiver(t *testing.T) {
	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	if err := configureADSBReceiver("ftp://receiver", time.Second); err == nil {
		t.Fatal("expected scheme error")
	}
	if err := configureADSBReceiver("http://receiver:8080/", 2*time.Second); err != nil {
		t.Fatal(err)
	}
	if adsbReceiver.BaseURL != "http://receiver:8080" || adsbReceiver.Timeout != 2*time.Second {
		t.Fatalf("config = %#v", adsbReceiver)
	}
	if err := configureADSBReceiver("", time.Second); err != nil {
		t.Fatal(err)
	}
	if adsbReceiver.BaseURL != "" {
		t.Fatalf("expected disabled config: %#v", adsbReceiver)
	}
}

func TestADSBStatusHandlerUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	adsbReceiver = adsbConfig{BaseURL: server.URL, Timeout: time.Second}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/adsb/status", nil)
	rec := httptest.NewRecorder()
	adsbStatusHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !containsAll(body, `"configured":true`, `"reachable":false`, `"error_code":"upstream_status"`, `"error"`) {
		t.Fatalf("body = %s", body)
	}
}

func TestADSBStatusDoesNotExposeCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"now":1788768000,"messages":1,"aircraft":[]}`))
	}))
	defer server.Close()
	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	adsbReceiver = adsbConfig{BaseURL: strings.Replace(server.URL, "http://", "http://secret:token@", 1), Timeout: time.Second}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/adsb/status", nil)
	rec := httptest.NewRecorder()
	adsbStatusHandler(rec, req)
	body := rec.Body.String()
	if strings.Contains(body, "secret") || strings.Contains(body, "token") || strings.Contains(body, "base_url") {
		t.Fatalf("status leaked receiver URL or credentials: %s", body)
	}
}

func TestFetchADSBAircraftInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"aircraft":`))
	}))
	defer server.Close()
	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	adsbReceiver = adsbConfig{BaseURL: server.URL, Timeout: time.Second}
	_, _, err := fetchADSBAircraft(context.Background())
	if err == nil || adsbErrorCode(err) != "invalid_json" {
		t.Fatalf("err = %v code = %q", err, adsbErrorCode(err))
	}
}

func TestFetchADSBAircraftResponseTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", int(adsbMaxResponseBytes+1))))
	}))
	defer server.Close()
	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	adsbReceiver = adsbConfig{BaseURL: server.URL, Timeout: 2 * time.Second}
	_, _, err := fetchADSBAircraft(context.Background())
	if err == nil || adsbErrorCode(err) != "response_too_large" {
		t.Fatalf("err = %v code = %q", err, adsbErrorCode(err))
	}
}

func containsAll(value string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}
