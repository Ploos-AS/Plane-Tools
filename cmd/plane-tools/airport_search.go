package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type nearbyAirport struct {
	Airport    airport `json:"airport"`
	DistanceNM float64 `json:"distance_nm"`
}

func airportSearchHandler(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if q == "" {
		writeError(w, fmt.Errorf("missing query parameter %q", "q"))
		return
	}
	limit, err := queryLimit(r, 20, 100)
	if err != nil {
		writeError(w, err)
		return
	}

	matches := make([]airport, 0)
	for _, item := range airportDB.byIdent {
		haystack := strings.ToLower(strings.Join([]string{item.Ident, item.ICAO, item.IATA, item.Name, item.Municipality, item.Country, item.Region}, " "))
		if strings.Contains(haystack, q) {
			matches = append(matches, item)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Name == matches[j].Name {
			return matches[i].Ident < matches[j].Ident
		}
		return matches[i].Name < matches[j].Name
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	writeJSON(w, http.StatusOK, matches)
}

func airportNearbyHandler(w http.ResponseWriter, r *http.Request) {
	lat, err := queryFloat(r, "lat")
	if err != nil {
		writeError(w, err)
		return
	}
	lon, err := queryFloat(r, "lon")
	if err != nil {
		writeError(w, err)
		return
	}
	if !validLatLon(lat, lon) {
		writeError(w, fmt.Errorf("coordinates must use latitude -90..90 and longitude -180..180"))
		return
	}
	radius := 50.0
	if raw := strings.TrimSpace(r.URL.Query().Get("radius_nm")); raw != "" {
		radius, err = strconv.ParseFloat(raw, 64)
		if err != nil || radius <= 0 || radius > 1000 {
			writeError(w, fmt.Errorf("radius_nm must be greater than 0 and at most 1000"))
			return
		}
	}
	limit, err := queryLimit(r, 20, 100)
	if err != nil {
		writeError(w, err)
		return
	}

	matches := make([]nearbyAirport, 0)
	for _, item := range airportDB.byIdent {
		distance := greatCircleNM(lat, lon, item.LatitudeDeg, item.LongitudeDeg)
		if distance <= radius {
			matches = append(matches, nearbyAirport{Airport: item, DistanceNM: round(distance, 3)})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].DistanceNM == matches[j].DistanceNM {
			return matches[i].Airport.Ident < matches[j].Airport.Ident
		}
		return matches[i].DistanceNM < matches[j].DistanceNM
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	writeJSON(w, http.StatusOK, matches)
}

func queryLimit(r *http.Request, fallback, max int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > max {
		return 0, fmt.Errorf("limit must be between 1 and %d", max)
	}
	return value, nil
}
