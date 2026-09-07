package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestSpottingLocationCreateAndList(t *testing.T) {
	old := spottingDB
	spottingDB = &spottingLocationStore{path: filepath.Join(t.TempDir(), "spotting.json")}
	defer func() { spottingDB = old }()

	body := bytes.NewBufferString(`{"name":"Kjevik spot","latitude_deg":58.205,"longitude_deg":8.09}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/spotting-locations", body)
	rr := httptest.NewRecorder()
	spottingLocationsHandler(rr, req)
	if rr.Code != http.StatusCreated { t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String()) }

	req = httptest.NewRequest(http.MethodGet, "/api/v1/spotting-locations", nil)
	rr = httptest.NewRecorder()
	spottingLocationsHandler(rr, req)
	if rr.Code != http.StatusOK { t.Fatalf("list status=%d", rr.Code) }
	var got []spottingLocation
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil { t.Fatal(err) }
	if len(got) != 1 || got[0].Name != "Kjevik spot" { t.Fatalf("unexpected items: %+v", got) }
}

func TestSpottingLocationRejectsDuplicateName(t *testing.T) {
	old := spottingDB
	spottingDB = &spottingLocationStore{path: filepath.Join(t.TempDir(), "spotting.json")}
	defer func() { spottingDB = old }()

	first := &spottingLocation{Name: "Spot A", LatitudeDeg: 58, LongitudeDeg: 8}
	if err := spottingDB.add(first); err != nil { t.Fatal(err) }
	duplicate := &spottingLocation{Name: "spot a", LatitudeDeg: 59, LongitudeDeg: 9}
	if err := spottingDB.add(duplicate); err == nil { t.Fatal("expected duplicate error") }
}

func TestSpottingAnalysisFindsNearestAirport(t *testing.T) {
	oldAirports := airportDB
	airportDB = newAirportStore(seedAirports)
	defer func() { airportDB = oldAirports }()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spotting-locations/analyze?lat=58.2042&lon=8.0854&name=Kjevik", nil)
	rr := httptest.NewRecorder()
	spottingAnalysisHandler(rr, req)
	if rr.Code != http.StatusOK { t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String()) }
	var got spottingAnalysis
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil { t.Fatal(err) }
	if got.Airport.Ident != "ENCN" { t.Fatalf("unexpected airport: %+v", got.Airport) }
	if got.DistanceNM != 0 { t.Fatalf("expected zero distance, got %f", got.DistanceNM) }
	if got.BearingDeg < 0 || got.BearingDeg >= 360 { t.Fatalf("invalid bearing: %f", got.BearingDeg) }
}

func TestConfigureSpottingLocationStoreMissingFile(t *testing.T) {
	old := spottingDB
	defer func() { spottingDB = old }()
	path := filepath.Join(t.TempDir(), "missing.json")
	if err := configureSpottingLocationStore(path); err != nil { t.Fatal(err) }
	if spottingDB.path != path || len(spottingDB.items) != 0 { t.Fatalf("unexpected store: %+v", spottingDB) }
}
