package main

import (
	"fmt"
	"net/http"
	"sort"
)

type logbookCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type lifelistEntry struct {
	AircraftKey  string `json:"aircraft_key"`
	ICAO24       string `json:"icao24,omitempty"`
	Registration string `json:"registration,omitempty"`
	FirstSeen    string `json:"first_seen"`
	LastSeen     string `json:"last_seen"`
	Sightings    int    `json:"sightings"`
}

type logbookSummary struct {
	TotalSightings      int             `json:"total_sightings"`
	UniqueAircraft      int             `json:"unique_aircraft"`
	UniqueICAO24        int             `json:"unique_icao24"`
	UniqueRegistrations int             `json:"unique_registrations"`
	FirstObservation    string          `json:"first_observation,omitempty"`
	LastObservation     string          `json:"last_observation,omitempty"`
	ByAirport           []logbookCount  `json:"by_airport"`
	BySpottingLocation  []logbookCount  `json:"by_spotting_location"`
	Lifelist            []lifelistEntry `json:"lifelist"`
}

func spottingLogSummaryHandler(w http.ResponseWriter, r *http.Request) {
	filter, err := parseSpottingLogFilter(r)
	if err != nil {
		writeError(w, err)
		return
	}

	spottingLogDB.mu.RLock()
	items := append([]spottingLogEntry(nil), spottingLogDB.items...)
	spottingLogDB.mu.RUnlock()

	items = filterSpottingLogEntries(items, filter)
	writeJSON(w, http.StatusOK, summarizeSpottingLog(items))
}

func summarizeSpottingLog(items []spottingLogEntry) logbookSummary {
	summary := logbookSummary{
		TotalSightings:     len(items),
		ByAirport:          []logbookCount{},
		BySpottingLocation: []logbookCount{},
		Lifelist:           []lifelistEntry{},
	}
	if len(items) == 0 {
		return summary
	}

	icaoSet := map[string]bool{}
	registrationSet := map[string]bool{}
	airportCounts := map[string]int{}
	locationCounts := map[string]int{}
	life := map[string]*lifelistEntry{}

	for _, item := range items {
		if summary.FirstObservation == "" || item.ObservedAt < summary.FirstObservation {
			summary.FirstObservation = item.ObservedAt
		}
		if summary.LastObservation == "" || item.ObservedAt > summary.LastObservation {
			summary.LastObservation = item.ObservedAt
		}
		if item.ICAO24 != "" {
			icaoSet[item.ICAO24] = true
		}
		if item.Registration != "" {
			registrationSet[item.Registration] = true
		}
		if item.AirportIdent != "" {
			airportCounts[item.AirportIdent]++
		}
		if item.SpottingLocationID != "" {
			locationCounts[item.SpottingLocationID]++
		}

		key := spottingAircraftKey(item)
		entry, ok := life[key]
		if !ok {
			life[key] = &lifelistEntry{
				AircraftKey:  key,
				ICAO24:       item.ICAO24,
				Registration: item.Registration,
				FirstSeen:    item.ObservedAt,
				LastSeen:     item.ObservedAt,
				Sightings:    1,
			}
			continue
		}
		entry.Sightings++
		if item.ObservedAt < entry.FirstSeen {
			entry.FirstSeen = item.ObservedAt
		}
		if item.ObservedAt > entry.LastSeen {
			entry.LastSeen = item.ObservedAt
		}
		if entry.ICAO24 == "" && item.ICAO24 != "" {
			entry.ICAO24 = item.ICAO24
		}
		if entry.Registration == "" && item.Registration != "" {
			entry.Registration = item.Registration
		}
	}

	summary.UniqueICAO24 = len(icaoSet)
	summary.UniqueRegistrations = len(registrationSet)
	summary.UniqueAircraft = len(life)
	summary.ByAirport = sortedLogbookCounts(airportCounts)
	summary.BySpottingLocation = sortedLogbookCounts(locationCounts)
	for _, entry := range life {
		summary.Lifelist = append(summary.Lifelist, *entry)
	}
	sort.Slice(summary.Lifelist, func(i, j int) bool {
		if summary.Lifelist[i].FirstSeen == summary.Lifelist[j].FirstSeen {
			return summary.Lifelist[i].AircraftKey < summary.Lifelist[j].AircraftKey
		}
		return summary.Lifelist[i].FirstSeen < summary.Lifelist[j].FirstSeen
	})
	return summary
}

func spottingAircraftKey(item spottingLogEntry) string {
	if item.ICAO24 != "" {
		return "icao24:" + item.ICAO24
	}
	if item.Registration != "" {
		return "registration:" + item.Registration
	}
	return fmt.Sprintf("observation:%s", item.ID)
}

func sortedLogbookCounts(counts map[string]int) []logbookCount {
	result := make([]logbookCount, 0, len(counts))
	for key, count := range counts {
		result = append(result, logbookCount{Key: key, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].Key < result[j].Key
		}
		return result[i].Count > result[j].Count
	})
	return result
}
