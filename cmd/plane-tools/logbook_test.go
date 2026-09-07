package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestSpottingLogCreateAndList(t *testing.T) {
	oldLog := spottingLogDB
	oldSpots := spottingDB
	oldAirports := airportDB
	defer func() { spottingLogDB = oldLog; spottingDB = oldSpots; airportDB = oldAirports }()

	airportDB = newAirportStore(seedAirports)
	spottingDB = &spottingLocationStore{path: filepath.Join(t.TempDir(), "spots.json"), items: []spottingLocation{{ID: "kjevik-west", Name: "Kjevik west", LatitudeDeg: 58.21, LongitudeDeg: 8.07, AirportIdent: "ENCN"}}}
	spottingLogDB = &spottingLogStore{path: filepath.Join(t.TempDir(), "log.json")}

	body := bytes.NewBufferString(`{"observed_at":"2026-09-07T08:00:00Z","icao24":"4787a2","registration":"ln-test","spotting_location_id":"kjevik-west","notes":"test observation"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/spotting-log", body)
	rr := httptest.NewRecorder()
	spottingLogHandler(rr, req)
	if rr.Code != http.StatusCreated { t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String()) }

	var created spottingLogEntry
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil { t.Fatal(err) }
	if created.ID == "" || created.ICAO24 != "4787A2" || created.Registration != "LN-TEST" || created.AirportIdent != "ENCN" {
		t.Fatalf("unexpected created entry: %+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/spotting-log", nil)
	rr = httptest.NewRecorder()
	spottingLogHandler(rr, req)
	if rr.Code != http.StatusOK { t.Fatalf("list status=%d", rr.Code) }
	var got []spottingLogEntry
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil { t.Fatal(err) }
	if len(got) != 1 || got[0].ID != created.ID { t.Fatalf("unexpected log: %+v", got) }
}

func TestSpottingLogValidatesReferences(t *testing.T) {
	oldLog := spottingLogDB
	oldSpots := spottingDB
	oldAirports := airportDB
	defer func() { spottingLogDB = oldLog; spottingDB = oldSpots; airportDB = oldAirports }()

	airportDB = newAirportStore(seedAirports)
	spottingDB = &spottingLocationStore{path: filepath.Join(t.TempDir(), "spots.json")}
	spottingLogDB = &spottingLogStore{path: filepath.Join(t.TempDir(), "log.json")}

	for _, body := range []string{
		`{"observed_at":"bad","icao24":"4787A2"}`,
		`{"observed_at":"2026-09-07T08:00:00Z"}`,
		`{"observed_at":"2026-09-07T08:00:00Z","icao24":"BAD"}`,
		`{"observed_at":"2026-09-07T08:00:00Z","registration":"LN-X","spotting_location_id":"missing"}`,
		`{"observed_at":"2026-09-07T08:00:00Z","registration":"LN-X","airport_ident":"XXXX"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/spotting-log", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		spottingLogHandler(rr, req)
		if rr.Code != http.StatusBadRequest { t.Fatalf("body=%s status=%d response=%s", body, rr.Code, rr.Body.String()) }
	}
}

func TestConfigureSpottingLogStoreMissingFile(t *testing.T) {
	old := spottingLogDB
	defer func() { spottingLogDB = old }()
	path := filepath.Join(t.TempDir(), "missing.json")
	if err := configureSpottingLogStore(path); err != nil { t.Fatal(err) }
	if spottingLogDB.path != path || len(spottingLogDB.items) != 0 { t.Fatalf("unexpected store: %+v", spottingLogDB) }
}
