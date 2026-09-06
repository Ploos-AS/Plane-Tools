package main

import (
	"fmt"
	"net/http"
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

var aircraftData = []aircraft{
	{ICAO24: "4787A2", Registration: "LN-NGM", TypeCode: "B738", Manufacturer: "Boeing", Model: "737-800", Operator: "Norwegian"},
	{ICAO24: "4CA8E6", Registration: "EI-DCL", TypeCode: "B738", Manufacturer: "Boeing", Model: "737-800", Operator: "Ryanair"},
	{ICAO24: "4851F5", Registration: "PH-BHA", TypeCode: "B789", Manufacturer: "Boeing", Model: "787-9", Operator: "KLM"},
}

var icao24RE = regexp.MustCompile(`^[0-9A-F]{6}$`)

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

	for _, item := range aircraftData {
		if (icao24 != "" && item.ICAO24 == icao24) || (registration != "" && item.Registration == registration) {
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
