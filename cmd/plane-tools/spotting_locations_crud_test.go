package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpottingLocationCreateUpdateDeleteWithAirportBinding(t *testing.T) {
	oldSpotting := spottingDB
	oldAirport := airportDB
	spottingDB = &spottingLocationStore{path: filepath.Join(t.TempDir(), "spotting.json")}
	airportDB = newAirportStore(seedAirports)
	defer func() { spottingDB = oldSpotting; airportDB = oldAirport }()

	create := httptest.NewRequest(http.MethodPost, "/api/v1/spotting-locations", strings.NewReader(`{"name":"Kjevik spot","latitude_deg":58.20,"longitude_deg":8.08,"airport_ident":"encn"}`))
	createRR := httptest.NewRecorder()
	spottingLocationsHandler(createRR, create)
	if createRR.Code != http.StatusCreated { t.Fatalf("create status=%d body=%s", createRR.Code, createRR.Body.String()) }
	var created spottingLocation
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil { t.Fatal(err) }
	if created.ID == "" || created.AirportIdent != "ENCN" { t.Fatalf("unexpected created item: %+v", created) }

	update := httptest.NewRequest(http.MethodPut, "/api/v1/spotting-locations/"+created.ID, strings.NewReader(`{"name":"Kjevik west","latitude_deg":58.21,"longitude_deg":8.07,"airport_ident":"ENCN"}`))
	update.SetPathValue("id", created.ID)
	updateRR := httptest.NewRecorder()
	spottingLocationItemHandler(updateRR, update)
	if updateRR.Code != http.StatusOK { t.Fatalf("update status=%d body=%s", updateRR.Code, updateRR.Body.String()) }

	analysis := httptest.NewRequest(http.MethodGet, "/api/v1/spotting-locations/analyze?id="+created.ID, nil)
	analysisRR := httptest.NewRecorder()
	spottingAnalysisHandler(analysisRR, analysis)
	if analysisRR.Code != http.StatusOK { t.Fatalf("analysis status=%d body=%s", analysisRR.Code, analysisRR.Body.String()) }
	var result spottingAnalysis
	if err := json.Unmarshal(analysisRR.Body.Bytes(), &result); err != nil { t.Fatal(err) }
	if !result.Bound || result.Airport.Ident != "ENCN" { t.Fatalf("unexpected analysis: %+v", result) }

	remove := httptest.NewRequest(http.MethodDelete, "/api/v1/spotting-locations/"+created.ID, nil)
	remove.SetPathValue("id", created.ID)
	removeRR := httptest.NewRecorder()
	spottingLocationItemHandler(removeRR, remove)
	if removeRR.Code != http.StatusNoContent { t.Fatalf("delete status=%d body=%s", removeRR.Code, removeRR.Body.String()) }
	if _, ok := spottingDB.get(created.ID); ok { t.Fatal("deleted location still present") }
}

func TestSpottingLocationRejectsUnknownAirportBinding(t *testing.T) {
	oldAirport := airportDB
	airportDB = newAirportStore(seedAirports)
	defer func() { airportDB = oldAirport }()
	item := spottingLocation{Name: "Bad", LatitudeDeg: 58, LongitudeDeg: 8, AirportIdent: "ZZZZ"}
	if err := normalizeAndValidateSpottingLocation(&item); err == nil { t.Fatal("expected unknown airport binding error") }
}

func TestSlugSpottingID(t *testing.T) {
	if got := slugSpottingID(" Kjevik West / Spot "); got != "kjevik-west-spot" { t.Fatalf("got %q", got) }
}
