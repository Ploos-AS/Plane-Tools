package main

import (
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGreatCircleNM(t *testing.T) {
	got := greatCircleNM(58.2042, 8.0854, 59.9111, 10.7528)
	if math.Abs(got-131.45) > 0.5 {
		t.Fatalf("distance = %.2f NM, want about 131.45", got)
	}
}

func TestInitialBearing(t *testing.T) {
	got := initialBearing(58.2042, 8.0854, 59.9111, 10.7528)
	if got < 35 || got > 45 {
		t.Fatalf("bearing = %.2f, want roughly 40 degrees", got)
	}
}

func TestConvert(t *testing.T) {
	tests := []struct {
		value, want, tolerance float64
		from, to               string
	}{
		{1, 1.852, 0.0001, "nm", "km"},
		{100, 185.2, 0.0001, "kt", "kph"},
		{1000, 304.8, 0.0001, "ft", "m"},
		{1.852, 1, 0.0001, "km", "nm"},
		{185.2, 100, 0.0001, "kph", "kt"},
		{304.8, 1000, 0.0001, "m", "ft"},
	}

	for _, tt := range tests {
		got, err := convert(tt.value, tt.from, tt.to)
		if err != nil {
			t.Fatalf("%s->%s: %v", tt.from, tt.to, err)
		}
		if math.Abs(got-tt.want) > tt.tolerance {
			t.Fatalf("%s->%s = %f, want %f", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestDistanceHandlerRejectsInvalidCoordinates(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/distance?lat1=95&lon1=0&lat2=0&lon2=0", nil)
	rr := httptest.NewRecorder()

	distanceHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "latitude") {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestConvertHandlerRejectsUnsupportedPair(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/convert?value=1&from=nm&to=ft", nil)
	rr := httptest.NewRecorder()

	convertHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "unsupported conversion") {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}
