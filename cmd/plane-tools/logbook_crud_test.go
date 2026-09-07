package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestSpottingLogFilterUpdateDelete(t *testing.T) {
	oldLog := spottingLogDB
	oldSpotting := spottingDB
	oldAirports := airportDB
	spottingLogDB = &spottingLogStore{path: filepath.Join(t.TempDir(), "log.json")}
	spottingDB = &spottingLocationStore{path: filepath.Join(t.TempDir(), "spots.json"), items: []spottingLocation{{ID: "kjevik-west", Name: "Kjevik west", LatitudeDeg: 58.2, LongitudeDeg: 8.08, AirportIdent: "ENCN"}}}
	airportDB = newAirportStore(seedAirports)
	defer func() {
		spottingLogDB = oldLog
		spottingDB = oldSpotting
		airportDB = oldAirports
	}()

	create := func(body string) spottingLogEntry {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/spotting-log", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		spottingLogHandler(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
		}
		var item spottingLogEntry
		if err := json.Unmarshal(rr.Body.Bytes(), &item); err != nil {
			t.Fatal(err)
		}
		return item
	}

	first := create(`{"observed_at":"2026-09-06T10:00:00Z","icao24":"4787a2","registration":"ln-ngm","spotting_location_id":"kjevik-west","notes":"first"}`)
	_ = create(`{"observed_at":"2026-09-07T11:00:00Z","icao24":"4ca8e6","registration":"ei-dcl","airport_ident":"ENGM","notes":"second"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spotting-log?registration=LN-NGM&from=2026-09-06&to=2026-09-06", nil)
	rr := httptest.NewRecorder()
	spottingLogHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("filter status=%d body=%s", rr.Code, rr.Body.String())
	}
	var filtered []spottingLogEntry
	if err := json.Unmarshal(rr.Body.Bytes(), &filtered); err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].ID != first.ID || filtered[0].AirportIdent != "ENCN" {
		t.Fatalf("unexpected filtered result: %+v", filtered)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/v1/spotting-log/"+first.ID, bytes.NewBufferString(`{"observed_at":"2026-09-06T12:30:00Z","icao24":"4787A2","registration":"LN-NGM","airport_ident":"ENCN","notes":"updated"}`))
	req.SetPathValue("id", first.ID)
	rr = httptest.NewRecorder()
	spottingLogItemHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rr.Code, rr.Body.String())
	}
	var updated spottingLogEntry
	if err := json.Unmarshal(rr.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.ID != first.ID || updated.Notes != "updated" || updated.ObservedAt != "2026-09-06T12:30:00Z" {
		t.Fatalf("unexpected updated item: %+v", updated)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/spotting-log/"+first.ID, nil)
	req.SetPathValue("id", first.ID)
	rr = httptest.NewRecorder()
	spottingLogItemHandler(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(spottingLogDB.items) != 1 {
		t.Fatalf("expected one remaining entry, got %d", len(spottingLogDB.items))
	}
}

func TestSpottingLogFilterValidation(t *testing.T) {
	for _, path := range []string{
		"/api/v1/spotting-log?icao24=BAD",
		"/api/v1/spotting-log?from=not-a-date",
		"/api/v1/spotting-log?from=2026-09-08&to=2026-09-07",
		"/api/v1/spotting-log?limit=1001",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		spottingLogHandler(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d body=%s", path, rr.Code, rr.Body.String())
		}
	}
}

func TestSpottingLogItemNotFound(t *testing.T) {
	old := spottingLogDB
	spottingLogDB = &spottingLogStore{path: filepath.Join(t.TempDir(), "log.json")}
	defer func() { spottingLogDB = old }()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/spotting-log/missing", nil)
	req.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()
	spottingLogItemHandler(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
