package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSummarizeSpottingLog(t *testing.T) {
	items := []spottingLogEntry{
		{ID: "1", ObservedAt: "2026-09-01T10:00:00Z", ICAO24: "4787A2", Registration: "LN-NGM", AirportIdent: "ENCN", SpottingLocationID: "kjevik-west"},
		{ID: "2", ObservedAt: "2026-09-02T10:00:00Z", Registration: "LN-NGM", AirportIdent: "ENCN", SpottingLocationID: "kjevik-west"},
		{ID: "3", ObservedAt: "2026-09-03T10:00:00Z", ICAO24: "4CA8E6", Registration: "EI-DCL", AirportIdent: "ENGM"},
	}
	got := summarizeSpottingLog(items)
	if got.TotalSightings != 3 { t.Fatalf("total=%d", got.TotalSightings) }
	if got.UniqueAircraft != 2 { t.Fatalf("unique aircraft=%d", got.UniqueAircraft) }
	if got.UniqueICAO24 != 2 { t.Fatalf("unique icao=%d", got.UniqueICAO24) }
	if got.UniqueRegistrations != 2 { t.Fatalf("unique registrations=%d", got.UniqueRegistrations) }
	if got.FirstObservation != "2026-09-01T10:00:00Z" || got.LastObservation != "2026-09-03T10:00:00Z" { t.Fatalf("unexpected range: %+v", got) }
	if len(got.ByAirport) != 2 || got.ByAirport[0].Key != "ENCN" || got.ByAirport[0].Count != 2 { t.Fatalf("airports=%+v", got.ByAirport) }
	if len(got.BySpottingLocation) != 1 || got.BySpottingLocation[0].Count != 2 { t.Fatalf("locations=%+v", got.BySpottingLocation) }
	if len(got.Lifelist) != 2 || got.Lifelist[0].Sightings != 2 { t.Fatalf("lifelist=%+v", got.Lifelist) }
}

func TestSpottingLogSummaryHandlerFiltersWithoutListingLimit(t *testing.T) {
	old := spottingLogDB
	items := make([]spottingLogEntry, 0, 150)
	for i := 0; i < 150; i++ {
		items = append(items, spottingLogEntry{ID: "x", ObservedAt: "2026-09-01T10:00:00Z", ICAO24: "4787A2", Registration: "LN-NGM", AirportIdent: "ENCN"})
	}
	spottingLogDB = &spottingLogStore{items: items}
	defer func() { spottingLogDB = old }()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spotting-log/summary?airport_ident=encn", nil)
	rr := httptest.NewRecorder()
	spottingLogSummaryHandler(rr, req)
	if rr.Code != http.StatusOK { t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String()) }
	var got logbookSummary
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil { t.Fatal(err) }
	if got.TotalSightings != 150 { t.Fatalf("expected 150 sightings, got %d", got.TotalSightings) }
}

func TestSpottingLogSummaryHandlerValidatesFilters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/spotting-log/summary?icao24=bad", nil)
	rr := httptest.NewRecorder()
	spottingLogSummaryHandler(rr, req)
	if rr.Code != http.StatusBadRequest { t.Fatalf("status=%d", rr.Code) }
}
