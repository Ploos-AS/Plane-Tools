package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOurAirportsAndRunways(t *testing.T) {
	dir := t.TempDir()
	airports := filepath.Join(dir, "airports.csv")
	runways := filepath.Join(dir, "runways.csv")
	if err := os.WriteFile(airports, []byte(strings.Join([]string{
		"ident,type,name,latitude_deg,longitude_deg,elevation_ft,iso_country,iso_region,municipality,scheduled_service,gps_code,icao_code,iata_code",
		"ENGM,large_airport,Oslo Airport,60.1939,11.1004,681,NO,NO-32,Oslo,yes,ENGM,ENGM,OSL",
	}, "\n")+"\n"), 0o644); err != nil { t.Fatal(err) }
	if err := os.WriteFile(runways, []byte(strings.Join([]string{
		"airport_ident,length_ft,width_ft,surface,lighted,closed,le_ident,he_ident",
		"ENGM,11811,148,ASP,1,0,01L,19R",
	}, "\n")+"\n"), 0o644); err != nil { t.Fatal(err) }

	store, err := loadOurAirports(airports, runways)
	if err != nil { t.Fatal(err) }
	got, ok := store.byIATA["OSL"]
	if !ok || got.ICAO != "ENGM" { t.Fatalf("unexpected lookup: %+v %v", got, ok) }
	if len(got.Runways) != 1 || got.Runways[0].LowEnd != "01L" { t.Fatalf("unexpected runways: %+v", got.Runways) }
}

func TestLoadOurAirportsWithoutRunways(t *testing.T) {
	dir := t.TempDir()
	airports := filepath.Join(dir, "airports.csv")
	if err := os.WriteFile(airports, []byte("ident,type,name,latitude_deg,longitude_deg\nTEST,small_airport,Test Airport,1,2\n"), 0o644); err != nil { t.Fatal(err) }
	store, err := loadOurAirports(airports, filepath.Join(dir, "missing.csv"))
	if err != nil { t.Fatal(err) }
	if _, ok := store.byIdent["TEST"]; !ok { t.Fatal("airport not indexed") }
}

func TestAirportLookupByICAOAndIATA(t *testing.T) {
	old := airportDB
	airportDB = newAirportStore(seedAirports)
	defer func() { airportDB = old }()

	for _, path := range []string{"/api/v1/airport?icao=engm", "/api/v1/airport?iata=osl"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		airportLookupHandler(rr, req)
		if rr.Code != http.StatusOK { t.Fatalf("%s status=%d body=%s", path, rr.Code, rr.Body.String()) }
	}
}

func TestAirportLookupNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/airport?iata=ZZZ", nil)
	rr := httptest.NewRecorder()
	airportLookupHandler(rr, req)
	if rr.Code != http.StatusNotFound { t.Fatalf("status=%d", rr.Code) }
}
