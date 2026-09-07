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
	"unicode"
)

type spottingLocation struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	LatitudeDeg  float64 `json:"latitude_deg"`
	LongitudeDeg float64 `json:"longitude_deg"`
	AirportIdent string  `json:"airport_ident,omitempty"`
}

type spottingAnalysis struct {
	Location   spottingLocation `json:"location"`
	Airport    airport          `json:"airport"`
	DistanceNM float64          `json:"distance_nm"`
	BearingDeg float64          `json:"bearing_deg"`
	Bound      bool             `json:"bound"`
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
	seenIDs := map[string]bool{}
	for i := range store.items {
		store.items[i].Name = strings.TrimSpace(store.items[i].Name)
		store.items[i].AirportIdent = strings.ToUpper(strings.TrimSpace(store.items[i].AirportIdent))
		if store.items[i].Name == "" || !validLatLon(store.items[i].LatitudeDeg, store.items[i].LongitudeDeg) {
			return fmt.Errorf("spotting locations contain invalid entry")
		}
		if store.items[i].ID == "" {
			store.items[i].ID = uniqueSpottingID(slugSpottingID(store.items[i].Name), seenIDs)
		}
		if seenIDs[store.items[i].ID] {
			return fmt.Errorf("spotting locations contain duplicate id %q", store.items[i].ID)
		}
		seenIDs[store.items[i].ID] = true
		if err := validateAirportBinding(store.items[i].AirportIdent); err != nil {
			return err
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
		if err := normalizeAndValidateSpottingLocation(&item); err != nil {
			writeError(w, err)
			return
		}
		if err := spottingDB.add(&item); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func spottingLocationItemHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, fmt.Errorf("spotting location id is required"))
		return
	}
	switch r.Method {
	case http.MethodPut:
		var item spottingLocation
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			writeError(w, fmt.Errorf("invalid JSON body"))
			return
		}
		item.ID = id
		if err := normalizeAndValidateSpottingLocation(&item); err != nil {
			writeError(w, err)
			return
		}
		if err := spottingDB.update(id, item); err != nil {
			if err == errSpottingNotFound {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "spotting location not found"})
				return
			}
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := spottingDB.delete(id); err != nil {
			if err == errSpottingNotFound {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "spotting location not found"})
				return
			}
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

var errSpottingNotFound = fmt.Errorf("spotting location not found")

func spottingAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	if id := strings.TrimSpace(r.URL.Query().Get("id")); id != "" {
		item, ok := spottingDB.get(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "spotting location not found"})
			return
		}
		writeSpottingAnalysis(w, item)
		return
	}

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
	location := spottingLocation{Name: strings.TrimSpace(r.URL.Query().Get("name")), LatitudeDeg: lat, LongitudeDeg: lon, AirportIdent: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("airport_ident")))}
	if err := validateAirportBinding(location.AirportIdent); err != nil {
		writeError(w, err)
		return
	}
	writeSpottingAnalysis(w, location)
}

func writeSpottingAnalysis(w http.ResponseWriter, location spottingLocation) {
	var target airport
	var ok bool
	bound := location.AirportIdent != ""
	if bound {
		target, ok = airportDB.byIdent[location.AirportIdent]
	} else {
		target, ok = nearestAirport(location.LatitudeDeg, location.LongitudeDeg)
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no airports available"})
		return
	}
	writeJSON(w, http.StatusOK, spottingAnalysis{
		Location:   location,
		Airport:    target,
		DistanceNM: round(greatCircleNM(location.LatitudeDeg, location.LongitudeDeg, target.LatitudeDeg, target.LongitudeDeg), 3),
		BearingDeg: round(initialBearing(location.LatitudeDeg, location.LongitudeDeg, target.LatitudeDeg, target.LongitudeDeg), 2),
		Bound:      bound,
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

func normalizeAndValidateSpottingLocation(item *spottingLocation) error {
	item.Name = strings.TrimSpace(item.Name)
	item.AirportIdent = strings.ToUpper(strings.TrimSpace(item.AirportIdent))
	if item.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !validLatLon(item.LatitudeDeg, item.LongitudeDeg) {
		return fmt.Errorf("coordinates must use latitude -90..90 and longitude -180..180")
	}
	return validateAirportBinding(item.AirportIdent)
}

func validateAirportBinding(ident string) error {
	if ident == "" {
		return nil
	}
	if _, ok := airportDB.byIdent[ident]; !ok {
		return fmt.Errorf("unknown airport_ident %q", ident)
	}
	return nil
}

func (s *spottingLocationStore) add(item *spottingLocation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	seenIDs := map[string]bool{}
	for _, existing := range s.items {
		seenIDs[existing.ID] = true
		if strings.EqualFold(existing.Name, item.Name) {
			return fmt.Errorf("spotting location %q already exists", item.Name)
		}
	}
	if item.ID == "" {
		item.ID = uniqueSpottingID(slugSpottingID(item.Name), seenIDs)
	}
	updated := append(append([]spottingLocation(nil), s.items...), *item)
	if err := writeSpottingLocationsAtomic(s.path, updated); err != nil {
		return err
	}
	s.items = updated
	return nil
}

func (s *spottingLocationStore) get(id string) (spottingLocation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.items {
		if item.ID == id {
			return item, true
		}
	}
	return spottingLocation{}, false
}

func (s *spottingLocationStore) update(id string, item spottingLocation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := -1
	for i, existing := range s.items {
		if existing.ID == id {
			index = i
			continue
		}
		if strings.EqualFold(existing.Name, item.Name) {
			return fmt.Errorf("spotting location %q already exists", item.Name)
		}
	}
	if index < 0 {
		return errSpottingNotFound
	}
	updated := append([]spottingLocation(nil), s.items...)
	updated[index] = item
	if err := writeSpottingLocationsAtomic(s.path, updated); err != nil {
		return err
	}
	s.items = updated
	return nil
}

func (s *spottingLocationStore) delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := -1
	for i, item := range s.items {
		if item.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return errSpottingNotFound
	}
	updated := append([]spottingLocation(nil), s.items[:index]...)
	updated = append(updated, s.items[index+1:]...)
	if err := writeSpottingLocationsAtomic(s.path, updated); err != nil {
		return err
	}
	s.items = updated
	return nil
}

func slugSpottingID(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func uniqueSpottingID(base string, seen map[string]bool) string {
	if base == "" {
		base = "location"
	}
	if !seen[base] {
		return base
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if !seen[candidate] {
			return candidate
		}
	}
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
