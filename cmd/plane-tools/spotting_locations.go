package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type spottingLocation struct {
	Name         string  `json:"name"`
	LatitudeDeg  float64 `json:"latitude_deg"`
	LongitudeDeg float64 `json:"longitude_deg"`
}

type spottingAnalysis struct {
	Location   spottingLocation `json:"location"`
	Airport    airport          `json:"airport"`
	DistanceNM float64          `json:"distance_nm"`
	BearingDeg float64          `json:"bearing_deg"`
}

type spottingLocationStore struct {
	mu    sync.RWMutex
	path  string
	items []spottingLocation
}

var spottingDB = &spottingLocationStore{path: "/data/spotting-locations.json"}

func configureSpottingLocationStore(path string) error {
	store := &spottingLocationStore{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			spottingDB = store
			return nil
		}
		return err
	}
	if len(data) != 0 {
		if err := json.Unmarshal(data, &store.items); err != nil {
			return fmt.Errorf("decode spotting locations: %w", err)
		}
	}
	for i := range store.items {
		store.items[i].Name = strings.TrimSpace(store.items[i].Name)
		if store.items[i].Name == "" || !validLatLon(store.items[i].LatitudeDeg, store.items[i].LongitudeDeg) {
			return fmt.Errorf("spotting locations contain invalid entry")
		}
	}
	spottingDB = store
	return nil
}

func spottingLocationsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		spottingDB.mu.RLock()
		items := append([]spottingLocation(nil), spottingDB.items...)
		spottingDB.mu.RUnlock()
		sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name) })
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item spottingLocation
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			writeError(w, fmt.Errorf("invalid JSON body"))
			return
		}
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			writeError(w, fmt.Errorf("name is required"))
			return
		}
		if !validLatLon(item.LatitudeDeg, item.LongitudeDeg) {
			writeError(w, fmt.Errorf("coordinates must use latitude -90..90 and longitude -180..180"))
			return
		}
		if err := spottingDB.add(item); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func spottingAnalysisHandler(w http.ResponseWriter, r *http.Request) {
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

	nearest, ok := nearestAirport(lat, lon)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no airports available"})
		return
	}
	location := spottingLocation{Name: strings.TrimSpace(r.URL.Query().Get("name")), LatitudeDeg: lat, LongitudeDeg: lon}
	writeJSON(w, http.StatusOK, spottingAnalysis{
		Location:   location,
		Airport:    nearest,
		DistanceNM: round(greatCircleNM(lat, lon, nearest.LatitudeDeg, nearest.LongitudeDeg), 3),
		BearingDeg: round(initialBearing(lat, lon, nearest.LatitudeDeg, nearest.LongitudeDeg), 2),
	})
}

func nearestAirport(lat, lon float64) (airport, bool) {
	var best airport
	bestDistance := 0.0
	found := false
	for _, item := range airportDB.byIdent {
		distance := greatCircleNM(lat, lon, item.LatitudeDeg, item.LongitudeDeg)
		if !found || distance < bestDistance || (distance == bestDistance && item.Ident < best.Ident) {
			best = item
			bestDistance = distance
			found = true
		}
	}
	return best, found
}

func (s *spottingLocationStore) add(item spottingLocation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.items {
		if strings.EqualFold(existing.Name, item.Name) {
			return fmt.Errorf("spotting location %q already exists", item.Name)
		}
	}
	updated := append(append([]spottingLocation(nil), s.items...), item)
	if err := writeSpottingLocationsAtomic(s.path, updated); err != nil {
		return err
	}
	s.items = updated
	return nil
}

func writeSpottingLocationsAtomic(path string, items []spottingLocation) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create spotting location directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".spotting-locations-*.json")
	if err != nil {
		return fmt.Errorf("create spotting location temp file: %w", err)
	}
	tmp := file.Name()
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(tmp)
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(items); err != nil {
		cleanup()
		return err
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace spotting locations: %w", err)
	}
	return nil
}
