package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type runway struct {
	LengthFT int    `json:"length_ft,omitempty"`
	WidthFT  int    `json:"width_ft,omitempty"`
	Surface  string `json:"surface,omitempty"`
	Lighted  bool   `json:"lighted"`
	Closed   bool   `json:"closed"`
	LowEnd   string `json:"low_end,omitempty"`
	HighEnd  string `json:"high_end,omitempty"`
}

type airport struct {
	Ident            string   `json:"ident"`
	ICAO             string   `json:"icao,omitempty"`
	IATA             string   `json:"iata,omitempty"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	LatitudeDeg      float64  `json:"latitude_deg"`
	LongitudeDeg     float64  `json:"longitude_deg"`
	ElevationFT      int      `json:"elevation_ft,omitempty"`
	Country          string   `json:"country,omitempty"`
	Region           string   `json:"region,omitempty"`
	Municipality     string   `json:"municipality,omitempty"`
	ScheduledService bool     `json:"scheduled_service"`
	Runways          []runway `json:"runways,omitempty"`
}

type airportStore struct {
	byIdent map[string]airport
	byICAO  map[string]airport
	byIATA  map[string]airport
}

var seedAirports = []airport{
	{Ident: "ENGM", ICAO: "ENGM", IATA: "OSL", Name: "Oslo Airport, Gardermoen", Type: "large_airport", LatitudeDeg: 60.1939, LongitudeDeg: 11.1004, ElevationFT: 681, Country: "NO", Region: "NO-32", Municipality: "Oslo", ScheduledService: true},
	{Ident: "ENCN", ICAO: "ENCN", IATA: "KRS", Name: "Kristiansand Airport, Kjevik", Type: "medium_airport", LatitudeDeg: 58.2042, LongitudeDeg: 8.0854, ElevationFT: 57, Country: "NO", Region: "NO-42", Municipality: "Kristiansand", ScheduledService: true},
}

var airportDB = newAirportStore(seedAirports)

func newAirportStore(items []airport) *airportStore {
	store := &airportStore{byIdent: map[string]airport{}, byICAO: map[string]airport{}, byIATA: map[string]airport{}}
	for _, item := range items {
		item.Ident = strings.ToUpper(strings.TrimSpace(item.Ident))
		item.ICAO = strings.ToUpper(strings.TrimSpace(item.ICAO))
		item.IATA = strings.ToUpper(strings.TrimSpace(item.IATA))
		store.byIdent[item.Ident] = item
		if item.ICAO != "" {
			store.byICAO[item.ICAO] = item
		}
		if item.IATA != "" {
			store.byIATA[item.IATA] = item
		}
	}
	return store
}

func configureAirportStore(airportsPath, runwaysPath string) error {
	store, err := loadOurAirports(airportsPath, runwaysPath)
	if err != nil {
		if os.IsNotExist(err) {
			airportDB = newAirportStore(seedAirports)
			return nil
		}
		return err
	}
	airportDB = store
	return nil
}

func loadOurAirports(airportsPath, runwaysPath string) (*airportStore, error) {
	file, err := os.Open(airportsPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r := csv.NewReader(file)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read airports header: %w", err)
	}
	cols := headerPositions(header)
	for _, required := range []string{"ident", "type", "name", "latitude_deg", "longitude_deg"} {
		if _, ok := cols[required]; !ok {
			return nil, fmt.Errorf("airports CSV missing required column %q", required)
		}
	}

	runwaysByAirport, err := loadRunwaysOptional(runwaysPath)
	if err != nil {
		return nil, err
	}

	var items []airport
	for row := 2; ; row++ {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read airports row %d: %w", row, err)
		}
		get := func(name string) string {
			idx, ok := cols[name]
			if !ok || idx >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[idx])
		}
		lat, err1 := strconv.ParseFloat(get("latitude_deg"), 64)
		lon, err2 := strconv.ParseFloat(get("longitude_deg"), 64)
		if err1 != nil || err2 != nil || !validLatLon(lat, lon) {
			return nil, fmt.Errorf("airports CSV row %d has invalid coordinates", row)
		}
		elev, _ := strconv.Atoi(get("elevation_ft"))
		ident := strings.ToUpper(get("ident"))
		icao := strings.ToUpper(get("icao_code"))
		if icao == "" {
			icao = strings.ToUpper(get("gps_code"))
		}
		item := airport{
			Ident: ident, ICAO: icao, IATA: strings.ToUpper(get("iata_code")), Name: get("name"), Type: get("type"),
			LatitudeDeg: lat, LongitudeDeg: lon, ElevationFT: elev, Country: get("iso_country"), Region: get("iso_region"),
			Municipality: get("municipality"), ScheduledService: strings.EqualFold(get("scheduled_service"), "yes"), Runways: runwaysByAirport[ident],
		}
		if item.Ident == "" || item.Name == "" {
			return nil, fmt.Errorf("airports CSV row %d missing ident or name", row)
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("airports CSV contains no data rows")
	}
	return newAirportStore(items), nil
}

func loadRunwaysOptional(path string) (map[string][]runway, error) {
	result := map[string][]runway{}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}
	defer file.Close()

	r := csv.NewReader(file)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read runways header: %w", err)
	}
	cols := headerPositions(header)
	if _, ok := cols["airport_ident"]; !ok {
		return nil, fmt.Errorf("runways CSV missing required column %q", "airport_ident")
	}
	for row := 2; ; row++ {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read runways row %d: %w", row, err)
		}
		get := func(name string) string {
			idx, ok := cols[name]
			if !ok || idx >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[idx])
		}
		length, _ := strconv.Atoi(get("length_ft"))
		width, _ := strconv.Atoi(get("width_ft"))
		ident := strings.ToUpper(get("airport_ident"))
		result[ident] = append(result[ident], runway{LengthFT: length, WidthFT: width, Surface: get("surface"), Lighted: get("lighted") == "1", Closed: get("closed") == "1", LowEnd: get("le_ident"), HighEnd: get("he_ident")})
	}
	return result, nil
}

func headerPositions(header []string) map[string]int {
	positions := make(map[string]int, len(header))
	for i, name := range header {
		positions[strings.ToLower(strings.TrimSpace(name))] = i
	}
	return positions
}

func airportLookupHandler(w http.ResponseWriter, r *http.Request) {
	icao := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("icao")))
	iata := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("iata")))
	ident := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("ident")))
	if icao == "" && iata == "" && ident == "" {
		writeError(w, fmt.Errorf("provide icao, iata or ident"))
		return
	}
	var item airport
	var ok bool
	if icao != "" {
		item, ok = airportDB.byICAO[icao]
	} else if iata != "" {
		item, ok = airportDB.byIATA[iata]
	} else {
		item, ok = airportDB.byIdent[ident]
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "airport not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}
