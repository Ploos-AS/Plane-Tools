package main

import (
	"net/http"
	"net/http/httptest"
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/aircraft?registration=ZZ-TEST", nil)
	rr := httptest.NewRecorder()
	aircraftLookupHandler(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}
