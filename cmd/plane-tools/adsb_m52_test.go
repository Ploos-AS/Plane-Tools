package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConfigureADSBPosition(t *testing.T) {
	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	adsbReceiver = adsbConfig{}
	if err := configureADSBPosition("58.2042", "8.0854"); err != nil { t.Fatal(err) }
	if adsbReceiver.Latitude == nil || adsbReceiver.Longitude == nil { t.Fatal("position not configured") }
	if *adsbReceiver.Latitude != 58.2042 || *adsbReceiver.Longitude != 8.0854 { t.Fatalf("position = %#v", adsbReceiver) }
	if err := configureADSBPosition("58.2", ""); err == nil { t.Fatal("expected paired coordinate validation") }
	if err := configureADSBPosition("91", "8"); err == nil { t.Fatal("expected coordinate range validation") }
}

func TestFetchADSBAircraftAddsReceiverGeometry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"aircraft":[{"hex":"4787a2","lat":58.3042,"lon":8.0854}]}`))
	}))
	defer server.Close()
	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	adsbReceiver = adsbConfig{BaseURL: server.URL, Timeout: time.Second}
	if err := configureADSBPosition("58.2042", "8.0854"); err != nil { t.Fatal(err) }
	items, _, err := fetchADSBAircraft(t.Context())
	if err != nil { t.Fatal(err) }
	if len(items) != 1 || items[0].DistanceNM == nil || items[0].BearingDeg == nil { t.Fatalf("items = %#v", items) }
	if *items[0].DistanceNM < 5.9 || *items[0].DistanceNM > 6.1 { t.Fatalf("distance = %v", *items[0].DistanceNM) }
	if *items[0].BearingDeg < 359 || *items[0].BearingDeg > 360 { t.Fatalf("bearing = %v", *items[0].BearingDeg) }
}

func TestADSBAircraftSearchHandlerFiltersAndSorts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"aircraft":[{"hex":"AAAAAA","flight":"SAS100","r":"LN-AAA","lat":58.25,"lon":8.0854,"alt_baro":5000},{"hex":"BBBBBB","flight":"WIF200","r":"LN-BBB","lat":58.40,"lon":8.0854,"alt_baro":12000}]}`))
	}))
	defer server.Close()
	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	adsbReceiver = adsbConfig{BaseURL: server.URL, Timeout: time.Second}
	if err := configureADSBPosition("58.2042", "8.0854"); err != nil { t.Fatal(err) }

	req := httptest.NewRequest(http.MethodGet, "/api/v1/adsb/aircraft/search?q=LN&max_distance_nm=20&min_altitude_ft=4000&sort=altitude&order=desc", nil)
	rec := httptest.NewRecorder()
	adsbAircraftSearchHandler(rec, req)
	if rec.Code != http.StatusOK { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
	var items []adsbAircraft
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil { t.Fatal(err) }
	if len(items) != 2 { t.Fatalf("items=%#v", items) }
	if items[0].Registration != "LN-BBB" || items[1].Registration != "LN-AAA" { t.Fatalf("sort=%#v", items) }
}

func TestADSBAircraftSearchRequiresDistanceWhenFiltered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"aircraft":[{"hex":"AAAAAA","lat":58.25,"lon":8.08}]}`))
	}))
	defer server.Close()
	old := adsbReceiver
	defer func() { adsbReceiver = old }()
	adsbReceiver = adsbConfig{BaseURL: server.URL, Timeout: time.Second}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/adsb/aircraft/search?max_distance_nm=20", nil)
	rec := httptest.NewRecorder()
	adsbAircraftSearchHandler(rec, req)
	if rec.Code != http.StatusOK { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
	var items []adsbAircraft
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil { t.Fatal(err) }
	if len(items) != 0 { t.Fatalf("expected unlocated distance filter to exclude aircraft: %#v", items) }
}
