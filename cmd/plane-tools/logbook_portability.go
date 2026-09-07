package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

var spottingLogCSVHeader = []string{"id", "observed_at", "icao24", "registration", "spotting_location_id", "airport_ident", "notes"}

func spottingLogExportHandler(w http.ResponseWriter, r *http.Request) {
	filter, err := parseSpottingLogFilter(r)
	if err != nil {
		writeError(w, err)
		return
	}
	spottingLogDB.mu.RLock()
	items := append([]spottingLogEntry(nil), spottingLogDB.items...)
	spottingLogDB.mu.RUnlock()
	items = filterSpottingLogEntries(items, filter)
	sort.Slice(items, func(i, j int) bool {
		if items[i].ObservedAt == items[j].ObservedAt {
			return items[i].ID < items[j].ID
		}
		return items[i].ObservedAt < items[j].ObservedAt
	})

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" || format == "json" {
		w.Header().Set("Content-Disposition", `attachment; filename="plane-tools-spotting-log.json"`)
		writeJSON(w, http.StatusOK, items)
		return
	}
	if format != "csv" {
		writeError(w, fmt.Errorf("format must be json or csv"))
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="plane-tools-spotting-log.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write(spottingLogCSVHeader)
	for _, item := range items {
		_ = writer.Write([]string{item.ID, item.ObservedAt, item.ICAO24, item.Registration, item.SpottingLocationID, item.AirportIdent, item.Notes})
	}
	writer.Flush()
}

func spottingLogImportHandler(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		contentType := strings.ToLower(r.Header.Get("Content-Type"))
		if strings.Contains(contentType, "text/csv") {
			format = "csv"
		} else {
			format = "json"
		}
	}

	var items []spottingLogEntry
	var err error
	switch format {
	case "json":
		err = json.NewDecoder(r.Body).Decode(&items)
	case "csv":
		items, err = decodeSpottingLogCSV(csv.NewReader(r.Body))
	default:
		writeError(w, fmt.Errorf("format must be json or csv"))
		return
	}
	if err != nil {
		writeError(w, fmt.Errorf("decode spotting log import: %w", err))
		return
	}
	if len(items) == 0 {
		writeError(w, fmt.Errorf("import contains no spotting log entries"))
		return
	}

	seen := map[string]bool{}
	for i := range items {
		if err := normalizeAndValidateSpottingLogEntry(&items[i]); err != nil {
			writeError(w, fmt.Errorf("spotting log entry %d: %w", i+1, err))
			return
		}
		if items[i].ID == "" {
			writeError(w, fmt.Errorf("spotting log entry %d: id is required for import", i+1))
			return
		}
		if seen[items[i].ID] {
			writeError(w, fmt.Errorf("duplicate spotting log id %q in import", items[i].ID))
			return
		}
		seen[items[i].ID] = true
	}

	mode := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mode")))
	if mode == "" {
		mode = "merge"
	}
	result, err := spottingLogDB.importItems(items, mode)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type spottingLogImportResult struct {
	Mode     string `json:"mode"`
	Imported int    `json:"imported"`
	Total    int    `json:"total"`
}

func (s *spottingLogStore) importItems(items []spottingLogEntry, mode string) (spottingLogImportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var updated []spottingLogEntry
	switch mode {
	case "replace":
		updated = append([]spottingLogEntry(nil), items...)
	case "merge":
		updated = append([]spottingLogEntry(nil), s.items...)
		index := make(map[string]int, len(updated))
		for i, item := range updated {
			index[item.ID] = i
		}
		for _, item := range items {
			if i, ok := index[item.ID]; ok {
				updated[i] = item
			} else {
				index[item.ID] = len(updated)
				updated = append(updated, item)
			}
		}
	default:
		return spottingLogImportResult{}, fmt.Errorf("mode must be merge or replace")
	}
	if err := writeSpottingLogAtomic(s.path, updated); err != nil {
		return spottingLogImportResult{}, err
	}
	s.items = updated
	return spottingLogImportResult{Mode: mode, Imported: len(items), Total: len(updated)}, nil
}

func decodeSpottingLogCSV(reader *csv.Reader) ([]spottingLogEntry, error) {
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	positions := map[string]int{}
	for i, name := range records[0] {
		positions[strings.ToLower(strings.TrimSpace(name))] = i
	}
	for _, required := range []string{"id", "observed_at"} {
		if _, ok := positions[required]; !ok {
			return nil, fmt.Errorf("CSV missing required column %q", required)
		}
	}
	value := func(record []string, name string) string {
		i, ok := positions[name]
		if !ok || i >= len(record) {
			return ""
		}
		return record[i]
	}
	items := make([]spottingLogEntry, 0, len(records)-1)
	for _, record := range records[1:] {
		if len(record) == 0 || strings.TrimSpace(strings.Join(record, "")) == "" {
			continue
		}
		items = append(items, spottingLogEntry{
			ID: value(record, "id"), ObservedAt: value(record, "observed_at"), ICAO24: value(record, "icao24"),
			Registration: value(record, "registration"), SpottingLocationID: value(record, "spotting_location_id"),
			AirportIdent: value(record, "airport_ident"), Notes: value(record, "notes"),
		})
	}
	return items, nil
}
