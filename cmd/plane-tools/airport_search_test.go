package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAirportSearchByNameAndMunicipality(t *testing.T) {
	old := airportDB
	airportDB = newAirportStore(seedAirports)
	defer func() { airportDB = old }()

	for _, q := range []string{"gardermoen", "kristiansand"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/airports/search?q="+q, nil)
		rr := httptest.NewRecorder()
		airportSearchHandler(rr, req)
		if rr.Code != http.StatusOK { t.Fatalf("q=%s status=%d body=%s", q, rr.Code, rr.Body.String()) }
		var got []airport
		if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil { t.Fatal(err) }
		if len(got) != 1 { t.Fatalf("q=%s got %d matches", q, len(got)) }
	}
}

func TestAirportNearbySortsByDistance(t *testing.T) {
	old := airportDB
	airportDB = newAirportStore(seedAirports)
	defer func() { airportDB = old }()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/airports/nearby?lat=58.2042&lon=8.0854&radius_nm=150", nil)
	rr := httptest.NewRecorder()
	airportNearbyHandler(rr, req)
	if rr.Code != http.StatusOK { t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String()) }
	var got []nearbyAirport
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil { t.Fatal(err) }
	if len(got) < 2 { t.Fatalf("expected at least 2 airports, got %d", len(got)) }
	if got[0].Airport.Ident != "ENCN" || got[0].DistanceNM != 0 { t.Fatalf("unexpected first result: %+v", got[0]) }
	if got[0].DistanceNM > got[1].DistanceNM { t.Fatalf("results not sorted: %+v", got) }
}

func TestAirportNearbyValidatesRadiusAndCoordinates(t *testing.T) {
	for _, path := range []string{
		"/api/v1/airports/nearby?lat=91&lon=0",
		"/api/v1/airports/nearby?lat=58&lon=8&radius_nm=0",
		"/api/v1/airports/nearby?lat=58&lon=8&limit=101",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		airportNearbyHandler(rr, req)
		if rr.Code != http.StatusBadRequest { t.Fatalf("%s status=%d", path, rr.Code) }
	}
}
