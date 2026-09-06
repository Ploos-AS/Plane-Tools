package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeAircraftKeys(t *testing.T) {
	if got := normalizeICAO24(" 4787a2 "); got != "4787A2" {
		t.Fatalf("normalizeICAO24 = %q", got)
	}
	if got := normalizeRegistration(" ln-ngm "); got != "LN-NGM" {
		t.Fatalf("normalizeRegistration = %q", got)
	}
}

func TestAircraftLookupByICAO24(t *testing.T) {
	aircraftDB = newAircraftStore(seedAircraft)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/aircraft?icao24=4787a2", nil)
	rr := httptest.NewRecorder()
	aircraftLookupHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "LN-NGM") {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestAircraftLookupByRegistration(t *testing.T) {
	aircraftDB = newAircraftStore(seedAircraft)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/aircraft?registration=ln-ngm", nil)
	rr := httptest.NewRecorder()
	aircraftLookupHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestAircraftLookupRejectsInvalidICAO24(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/aircraft?icao24=XYZ", nil)
	rr := httptest.NewRecorder()
	aircraftLookupHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestAircraftLookupNotFound(t *testing.T) {
	aircraftDB = newAircraftStore(seedAircraft)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/aircraft?registration=ZZ-TEST", nil)
	rr := httptest.NewRecorder()
	aircraftLookupHandler(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestLoadAircraftCSVBuildsIndexes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aircraft.csv")
	content := "icao24,registration,type_code,manufacturer,model,operator\n4abcde,ln-test,a20n,Airbus,A320neo,Test Air\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	store, err := loadAircraftCSV(path)
	if err != nil {
		t.Fatalf("loadAircraftCSV: %v", err)
	}
	item, ok := store.byICAO24["4ABCDE"]
	if !ok || item.Registration != "LN-TEST" || item.TypeCode != "A20N" {
		t.Fatalf("unexpected ICAO24 index item: %#v", item)
	}
	if _, ok := store.byRegistration["LN-TEST"]; !ok {
		t.Fatal("registration index missing LN-TEST")
	}
}

func TestLoadAircraftCSVRejectsBadSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aircraft.csv")
	if err := os.WriteFile(path, []byte("icao24,registration\n4ABCDE,LN-TEST\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAircraftCSV(path); err == nil || !strings.Contains(err.Error(), "missing required column") {
		t.Fatalf("expected schema error, got %v", err)
	}
}

func TestConfigureAircraftStoreFallsBackWhenMissing(t *testing.T) {
	aircraftDB = nil
	if err := configureAircraftStore(filepath.Join(t.TempDir(), "missing.csv")); err != nil {
		t.Fatalf("configureAircraftStore: %v", err)
	}
	if _, ok := aircraftDB.byICAO24["4787A2"]; !ok {
		t.Fatal("seed fallback not loaded")
	}
}
