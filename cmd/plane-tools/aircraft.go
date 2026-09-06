package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type aircraft struct {
	ICAO24       string `json:"icao24"`
	Registration string `json:"registration"`
	TypeCode     string `json:"type_code"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Operator     string `json:"operator,omitempty"`
}

type aircraftStore struct {
	byICAO24       map[string]aircraft
	byRegistration map[string]aircraft
}

var seedAircraft = []aircraft{
	{ICAO24: "4787A2", Registration: "LN-NGM", TypeCode: "B738", Manufacturer: "Boeing", Model: "737-800", Operator: "Norwegian"},
	{ICAO24: "4CA8E6", Registration: "EI-DCL", TypeCode: "B738", Manufacturer: "Boeing", Model: "737-800", Operator: "Ryanair"},
	{ICAO24: "4851F5", Registration: "PH-BHA", TypeCode: "B789", Manufacturer: "Boeing", Model: "787-9", Operator: "KLM"},
}

var aircraftDB = newAircraftStore(seedAircraft)
var icao24RE = regexp.MustCompile(`^[0-9A-F]{6}$`)

func newAircraftStore(items []aircraft) *aircraftStore {
	store := &aircraftStore{
		byICAO24:       make(map[string]aircraft, len(items)),
		byRegistration: make(map[string]aircraft, len(items)),
	}
	for _, item := range items {
		item.ICAO24 = normalizeICAO24(item.ICAO24)
		item.Registration = normalizeRegistration(item.Registration)
		item.TypeCode = strings.ToUpper(strings.TrimSpace(item.TypeCode))
		store.byICAO24[item.ICAO24] = item
		store.byRegistration[item.Registration] = item
	}
	return store
}

func loadAircraftCSV(path string) (*aircraftStore, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read aircraft CSV header: %w", err)
	}

	columns := make(map[string]int, len(header))
	for i, name := range header {
		columns[strings.ToLower(strings.TrimSpace(name))] = i
	}
	for _, required := range []string{"icao24", "registration", "type_code", "manufacturer", "model"} {
		if _, ok := columns[required]; !ok {
			return nil, fmt.Errorf("aircraft CSV missing required column %q", required)
		}
	}

	var items []aircraft
	for rowNumber := 2; ; rowNumber++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read aircraft CSV row %d: %w", rowNumber, err)
		}

		value := func(name string) string {
			idx, ok := columns[name]
			if !ok || idx >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[idx])
		}

		item := aircraft{
			ICAO24:       normalizeICAO24(value("icao24")),
			Registration: normalizeRegistration(value("registration")),
			TypeCode:     strings.ToUpper(value("type_code")),
			Manufacturer: value("manufacturer"),
			Model:        value("model"),
			Operator:     value("operator"),
		}
		if !icao24RE.MatchString(item.ICAO24) {
			return nil, fmt.Errorf("aircraft CSV row %d has invalid icao24 %q", rowNumber, item.ICAO24)
		}
		if item.Registration == "" {
			return nil, fmt.Errorf("aircraft CSV row %d has empty registration", rowNumber)
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("aircraft CSV contains no data rows")
	}
	return newAircraftStore(items), nil
}

func configureAircraftStore(path string) error {
	store, err := loadAircraftCSV(path)
	if err != nil {
		if os.IsNotExist(err) {
			aircraftDB = newAircraftStore(seedAircraft)
			return nil
		}
		return err
	}
	aircraftDB = store
	return nil
}

func aircraftLookupHandler(w http.ResponseWriter, r *http.Request) {
	icao24 := normalizeICAO24(r.URL.Query().Get("icao24"))
	registration := normalizeRegistration(r.URL.Query().Get("registration"))

	if icao24 == "" && registration == "" {
		writeError(w, fmt.Errorf("provide icao24 or registration"))
		return
	}
	if icao24 != "" && !icao24RE.MatchString(icao24) {
		writeError(w, fmt.Errorf("icao24 must be exactly 6 hexadecimal characters"))
		return
	}

	if icao24 != "" {
		if item, ok := aircraftDB.byICAO24[icao24]; ok {
			writeJSON(w, http.StatusOK, item)
			return
		}
	}
	if registration != "" {
		if item, ok := aircraftDB.byRegistration[registration]; ok {
			writeJSON(w, http.StatusOK, item)
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "aircraft not found"})
}

func normalizeICAO24(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeRegistration(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
