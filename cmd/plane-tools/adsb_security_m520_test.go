package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func closedCredentialedADSBURL(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return "http://secret:token@" + address
}

func TestM520ReceiverFailureDoesNotExposeCredentials(t *testing.T) {
	old := adsbReceiver
	defer func() {
		adsbReceiver = old
		invalidateADSBCache()
	}()
	adsbReceiver = adsbConfig{BaseURL: closedCredentialedADSBURL(t), Timeout: 250 * time.Millisecond}
	invalidateADSBCache()

	checks := []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{name: "status", path: "/api/v1/adsb/status", handler: adsbStatusHandler},
		{name: "snapshot", path: "/api/v1/adsb/snapshot", handler: adsbSnapshotHandler},
		{name: "aircraft", path: "/api/v1/adsb/aircraft", handler: adsbAircraftHandler},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			invalidateADSBCache()
			rec := httptest.NewRecorder()
			check.handler(rec, httptest.NewRequest(http.MethodGet, check.path, nil))
			body := rec.Body.String()
			for _, forbidden := range []string{"secret", "token", "127.0.0.1", "base_url", "userinfo"} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("%s leaked %q: %s", check.name, forbidden, body)
				}
			}
			if !strings.Contains(body, "unreachable") {
				t.Fatalf("%s missing stable error code: %s", check.name, body)
			}
		})
	}
}

func TestM520PublicStatusAndSnapshotDoNotExposeReceiverCoordinates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"now":1788768000,"messages":1,"aircraft":[]}`))
	}))
	defer server.Close()

	old := adsbReceiver
	defer func() {
		adsbReceiver = old
		invalidateADSBCache()
	}()
	lat, lon := 58.123456, 8.654321
	adsbReceiver = adsbConfig{BaseURL: server.URL, Timeout: time.Second, Latitude: &lat, Longitude: &lon}
	invalidateADSBCache()

	checks := []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{name: "status", path: "/api/v1/adsb/status", handler: adsbStatusHandler},
		{name: "snapshot", path: "/api/v1/adsb/snapshot", handler: adsbSnapshotHandler},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			check.handler(rec, httptest.NewRequest(http.MethodGet, check.path, nil))
			body := rec.Body.String()
			if !strings.Contains(body, `"position_configured":true`) {
				t.Fatalf("%s lost position-configured signal: %s", check.name, body)
			}
			for _, forbidden := range []string{"receiver_lat", "receiver_lon", "58.123456", "8.654321"} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("%s exposed receiver coordinate %q: %s", check.name, forbidden, body)
				}
			}
		})
	}
}

func TestM520PublicErrorMessagesAreStable(t *testing.T) {
	err := newADSBFetchError("unreachable", "dial secret:token@receiver.example: %s", "connection refused")
	if got := adsbPublicErrorMessage(err); got != "ADS-B receiver is unreachable" {
		t.Fatalf("public error = %q", got)
	}
	if strings.Contains(adsbPublicErrorMessage(err), "secret") || strings.Contains(adsbPublicErrorMessage(err), "receiver.example") {
		t.Fatal("public error message exposed raw upstream detail")
	}
}
