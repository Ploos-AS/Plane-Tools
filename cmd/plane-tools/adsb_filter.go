package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func adsbAircraftSearchHandler(w http.ResponseWriter, r *http.Request) {
	items, _, err := fetchADSBAircraft(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	q := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("q")))
	minAltitude, err := optionalIntQuery(r, "min_altitude_ft")
	if err != nil { writeError(w, err); return }
	maxAltitude, err := optionalIntQuery(r, "max_altitude_ft")
	if err != nil { writeError(w, err); return }
	maxDistance, err := optionalFloatQuery(r, "max_distance_nm")
	if err != nil { writeError(w, err); return }
	if minAltitude != nil && maxAltitude != nil && *minAltitude > *maxAltitude {
		writeError(w, fmt.Errorf("min_altitude_ft must not exceed max_altitude_ft"))
		return
	}
	if maxDistance != nil && *maxDistance < 0 {
		writeError(w, fmt.Errorf("max_distance_nm must be zero or greater"))
		return
	}

	filtered := make([]adsbAircraft, 0, len(items))
	for _, item := range items {
		if q != "" && !strings.Contains(strings.ToUpper(strings.Join([]string{item.Hex, item.Flight, item.Registration, item.TypeCode}, " ")), q) {
			continue
		}
		if minAltitude != nil && (item.Altitude == nil || *item.Altitude < *minAltitude) { continue }
		if maxAltitude != nil && (item.Altitude == nil || *item.Altitude > *maxAltitude) { continue }
		if maxDistance != nil && (item.DistanceNM == nil || *item.DistanceNM > *maxDistance) { continue }
		filtered = append(filtered, item)
	}

	sortBy := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort")))
	if sortBy == "" { sortBy = "distance" }
	if sortBy != "distance" && sortBy != "altitude" && sortBy != "registration" && sortBy != "callsign" && sortBy != "seen" {
		writeError(w, fmt.Errorf("sort must be distance, altitude, registration, callsign or seen"))
		return
	}
	order := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order")))
	if order == "" { order = "asc" }
	if order != "asc" && order != "desc" { writeError(w, fmt.Errorf("order must be asc or desc")); return }

	sort.SliceStable(filtered, func(i, j int) bool {
		less := adsbLess(filtered[i], filtered[j], sortBy)
		if order == "desc" { return !less && !adsbEqual(filtered[i], filtered[j], sortBy) }
		return less
	})
	writeJSON(w, http.StatusOK, filtered)
}

func adsbLess(a, b adsbAircraft, sortBy string) bool {
	switch sortBy {
	case "altitude":
		return pointerIntLess(a.Altitude, b.Altitude)
	case "registration":
		return strings.Compare(a.Registration, b.Registration) < 0
	case "callsign":
		return strings.Compare(a.Flight, b.Flight) < 0
	case "seen":
		return pointerFloatLess(a.SeenSeconds, b.SeenSeconds)
	default:
		return pointerFloatLess(a.DistanceNM, b.DistanceNM)
	}
}

func adsbEqual(a, b adsbAircraft, sortBy string) bool {
	switch sortBy {
	case "altitude": return pointerIntEqual(a.Altitude, b.Altitude)
	case "registration": return a.Registration == b.Registration
	case "callsign": return a.Flight == b.Flight
	case "seen": return pointerFloatEqual(a.SeenSeconds, b.SeenSeconds)
	default: return pointerFloatEqual(a.DistanceNM, b.DistanceNM)
	}
}

func pointerFloatLess(a, b *float64) bool {
	if a == nil { return false }
	if b == nil { return true }
	return *a < *b
}
func pointerFloatEqual(a, b *float64) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
func pointerIntLess(a, b *int) bool {
	if a == nil { return false }
	if b == nil { return true }
	return *a < *b
}
func pointerIntEqual(a, b *int) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }

func optionalIntQuery(r *http.Request, name string) (*int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" { return nil, nil }
	value, err := strconv.Atoi(raw)
	if err != nil { return nil, fmt.Errorf("%s must be an integer", name) }
	return &value, nil
}

func optionalFloatQuery(r *http.Request, name string) (*float64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" { return nil, nil }
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil { return nil, fmt.Errorf("%s must be a number", name) }
	return &value, nil
}
