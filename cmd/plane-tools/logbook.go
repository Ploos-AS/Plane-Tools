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
	"time"
)

type spottingLogEntry struct {
	ID                 string `json:"id"`
	ObservedAt         string `json:"observed_at"`
	ICAO24             string `json:"icao24,omitempty"`
	Registration       string `json:"registration,omitempty"`
	SpottingLocationID string `json:"spotting_location_id,omitempty"`
	AirportIdent       string `json:"airport_ident,omitempty"`
	Notes              string `json:"notes,omitempty"`
}

type spottingLogStore struct {
	mu    sync.RWMutex
	path  string
	items []spottingLogEntry
}

var spottingLogDB = &spottingLogStore{path: "/data/spotting-log.json"}
var errSpottingLogNotFound = fmt.Errorf("spotting log entry not found")

func configureSpottingLogStore(path string) error {
	store := &spottingLogStore{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			spottingLogDB = store
			return nil
		}
		return err
	}
	if len(data) != 0 {
		if err := json.Unmarshal(data, &store.items); err != nil {
			return fmt.Errorf("decode spotting log: %w", err)
		}
	}
	for i := range store.items {
		if err := normalizeAndValidateSpottingLogEntry(&store.items[i]); err != nil {
			return fmt.Errorf("spotting log entry %d: %w", i+1, err)
		}
	}
	spottingLogDB = store
	return nil
}

func spottingLogHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := filteredSpottingLog(r)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item spottingLogEntry
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			writeError(w, fmt.Errorf("invalid JSON body"))
			return
		}
		if item.ObservedAt == "" {
			item.ObservedAt = time.Now().UTC().Format(time.RFC3339)
		}
		if err := normalizeAndValidateSpottingLogEntry(&item); err != nil {
			writeError(w, err)
			return
		}
		if err := spottingLogDB.add(&item); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func spottingLogItemHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, fmt.Errorf("spotting log id is required"))
		return
	}

	switch r.Method {
	case http.MethodPut:
		var item spottingLogEntry
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			writeError(w, fmt.Errorf("invalid JSON body"))
			return
		}
		item.ID = id
		if err := normalizeAndValidateSpottingLogEntry(&item); err != nil {
			writeError(w, err)
			return
		}
		if err := spottingLogDB.update(id, item); err != nil {
			if err == errSpottingLogNotFound {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := spottingLogDB.delete(id); err != nil {
			if err == errSpottingLogNotFound {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
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

func filteredSpottingLog(r *http.Request) ([]spottingLogEntry, error) {
	icao24 := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("icao24")))
	registration := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("registration")))
	airportIdent := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("airport_ident")))
	locationID := strings.TrimSpace(r.URL.Query().Get("spotting_location_id"))
	if icao24 != "" && !icao24RE.MatchString(icao24) {
		return nil, fmt.Errorf("icao24 must be exactly six hexadecimal characters")
	}

	from, err := parseLogTimeFilter(strings.TrimSpace(r.URL.Query().Get("from")), false)
	if err != nil {
		return nil, fmt.Errorf("invalid from: %w", err)
	}
	to, err := parseLogTimeFilter(strings.TrimSpace(r.URL.Query().Get("to")), true)
	if err != nil {
		return nil, fmt.Errorf("invalid to: %w", err)
	}
	if from != nil && to != nil && from.After(*to) {
		return nil, fmt.Errorf("from must not be after to")
	}
	limit, err := queryLimit(r, 100, 1000)
	if err != nil {
		return nil, err
	}

	spottingLogDB.mu.RLock()
	items := append([]spottingLogEntry(nil), spottingLogDB.items...)
	spottingLogDB.mu.RUnlock()

	matches := make([]spottingLogEntry, 0, len(items))
	for _, item := range items {
		if icao24 != "" && item.ICAO24 != icao24 {
			continue
		}
		if registration != "" && item.Registration != registration {
			continue
		}
		if airportIdent != "" && item.AirportIdent != airportIdent {
			continue
		}
		if locationID != "" && item.SpottingLocationID != locationID {
			continue
		}
		observed, _ := time.Parse(time.RFC3339, item.ObservedAt)
		if from != nil && observed.Before(*from) {
			continue
		}
		if to != nil && observed.After(*to) {
			continue
		}
		matches = append(matches, item)
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].ObservedAt == matches[j].ObservedAt {
			return matches[i].ID > matches[j].ID
		}
		return matches[i].ObservedAt > matches[j].ObservedAt
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func parseLogTimeFilter(raw string, endOfDay bool) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		parsed = parsed.UTC()
		return &parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("use RFC3339 or YYYY-MM-DD")
	}
	parsed = parsed.UTC()
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}

func normalizeAndValidateSpottingLogEntry(item *spottingLogEntry) error {
	item.ID = strings.TrimSpace(item.ID)
	item.ObservedAt = strings.TrimSpace(item.ObservedAt)
	item.ICAO24 = strings.ToUpper(strings.TrimSpace(item.ICAO24))
	item.Registration = strings.ToUpper(strings.TrimSpace(item.Registration))
	item.SpottingLocationID = strings.TrimSpace(item.SpottingLocationID)
	item.AirportIdent = strings.ToUpper(strings.TrimSpace(item.AirportIdent))
	item.Notes = strings.TrimSpace(item.Notes)

	parsed, err := time.Parse(time.RFC3339, item.ObservedAt)
	if err != nil {
		return fmt.Errorf("observed_at must be RFC3339")
	}
	item.ObservedAt = parsed.UTC().Format(time.RFC3339)
	if item.ICAO24 == "" && item.Registration == "" {
		return fmt.Errorf("provide icao24 or registration")
	}
	if item.ICAO24 != "" && !icao24RE.MatchString(item.ICAO24) {
		return fmt.Errorf("icao24 must be exactly six hexadecimal characters")
	}
	if item.SpottingLocationID != "" {
		location, ok := spottingDB.get(item.SpottingLocationID)
		if !ok {
			return fmt.Errorf("unknown spotting_location_id %q", item.SpottingLocationID)
		}
		if item.AirportIdent == "" && location.AirportIdent != "" {
			item.AirportIdent = location.AirportIdent
		}
	}
	if item.AirportIdent != "" {
		if _, ok := airportDB.byIdent[item.AirportIdent]; !ok {
			return fmt.Errorf("unknown airport_ident %q", item.AirportIdent)
		}
	}
	return nil
}

func (s *spottingLogStore) add(item *spottingLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.ID == "" {
		item.ID = nextSpottingLogID(item.ObservedAt, s.items)
	}
	for _, existing := range s.items {
		if existing.ID == item.ID {
			return fmt.Errorf("spotting log id %q already exists", item.ID)
		}
	}
	updated := append(append([]spottingLogEntry(nil), s.items...), *item)
	if err := writeSpottingLogAtomic(s.path, updated); err != nil {
		return err
	}
	s.items = updated
	return nil
}

func (s *spottingLogStore) update(id string, item spottingLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := -1
	for i, existing := range s.items {
		if existing.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return errSpottingLogNotFound
	}
	updated := append([]spottingLogEntry(nil), s.items...)
	updated[index] = item
	if err := writeSpottingLogAtomic(s.path, updated); err != nil {
		return err
	}
	s.items = updated
	return nil
}

func (s *spottingLogStore) delete(id string) error {
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
		return errSpottingLogNotFound
	}
	updated := append([]spottingLogEntry(nil), s.items[:index]...)
	updated = append(updated, s.items[index+1:]...)
	if err := writeSpottingLogAtomic(s.path, updated); err != nil {
		return err
	}
	s.items = updated
	return nil
}

func nextSpottingLogID(observedAt string, items []spottingLogEntry) string {
	base := strings.NewReplacer(":", "", "-", "", "T", "-", "Z", "").Replace(observedAt)
	if base == "" {
		base = "observation"
	}
	seen := map[string]bool{}
	for _, item := range items {
		seen[item.ID] = true
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

func writeSpottingLogAtomic(path string, items []spottingLogEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create spotting log directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".spotting-log-*.json")
	if err != nil {
		return fmt.Errorf("create spotting log temp file: %w", err)
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
		return fmt.Errorf("replace spotting log: %w", err)
	}
	return nil
}
