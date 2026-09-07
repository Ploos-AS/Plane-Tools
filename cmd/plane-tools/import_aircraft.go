package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type importStats struct {
	Rows       int `json:"rows"`
	Imported   int `json:"imported"`
	Duplicates int `json:"duplicates"`
	Invalid    int `json:"invalid"`
}

type sourceColumns struct {
	icao24       int
	registration int
	typeCode     int
	manufacturer int
	model        int
	operator     int
}

func runAircraftImport(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("import-aircraft", flag.ContinueOnError)
	fs.SetOutput(stdout)
	input := fs.String("input", "", "source aircraft database file")
	output := fs.String("output", getenv("PLANE_TOOLS_AIRCRAFT_CSV", "/data/aircraft.csv"), "canonical output CSV")
	format := fs.String("format", "auto", "input format: auto, csv, or tar1090")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*input) == "" {
		return errors.New("--input is required")
	}

	stats, err := importAircraftByFormat(*input, *output, *format)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "rows=%d imported=%d duplicates=%d invalid=%d format=%s output=%s\n", stats.Rows, stats.Imported, stats.Duplicates, stats.Invalid, normalizeAircraftImportFormat(*format, *input), *output)
	return err
}

func importAircraftCSV(inputPath, outputPath string) (importStats, error) {
	input, err := os.Open(inputPath)
	if err != nil {
		return importStats{}, fmt.Errorf("open input: %w", err)
	}
	defer input.Close()

	r := csv.NewReader(input)
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return importStats{}, fmt.Errorf("read header: %w", err)
	}
	cols, err := mapSourceColumns(header)
	if err != nil {
		return importStats{}, err
	}

	stats := importStats{}
	byICAO := make(map[string]aircraft)
	byRegistration := make(map[string]string)

	for {
		record, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return stats, fmt.Errorf("read CSV row %d: %w", stats.Rows+2, err)
		}
		stats.Rows++
		item, ok := aircraftFromSourceRecord(record, cols)
		if !ok {
			stats.Invalid++
			continue
		}
		if _, exists := byICAO[item.ICAO24]; exists {
			stats.Duplicates++
			continue
		}
		if _, exists := byRegistration[item.Registration]; exists {
			stats.Duplicates++
			continue
		}
		byICAO[item.ICAO24] = item
		byRegistration[item.Registration] = item.ICAO24
	}

	items := make([]aircraft, 0, len(byICAO))
	for _, item := range byICAO {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ICAO24 < items[j].ICAO24 })
	stats.Imported = len(items)
	if stats.Imported == 0 {
		return stats, errors.New("no valid aircraft rows to import")
	}

	if err := writeAircraftCSVAtomic(outputPath, items); err != nil {
		return stats, err
	}
	return stats, nil
}

func mapSourceColumns(header []string) (sourceColumns, error) {
	aliases := map[string]string{
		"icao24": "icao24", "hex": "icao24", "icao": "icao24", "modes": "icao24", "modeshex": "icao24",
		"registration": "registration", "reg": "registration", "tail": "registration", "tailnumber": "registration",
		"typecode": "type_code", "type": "type_code", "aircrafttype": "type_code", "aircrafttypecode": "type_code",
		"manufacturer": "manufacturer", "maker": "manufacturer",
		"model": "model", "aircraftmodel": "model",
		"operator": "operator", "operatorname": "operator", "owner": "operator",
	}

	positions := map[string]int{}
	for i, raw := range header {
		if canonical, ok := aliases[normalizeHeader(raw)]; ok {
			if _, exists := positions[canonical]; !exists {
				positions[canonical] = i
			}
		}

	for _, required := range []string{"icao24", "registration", "type_code", "manufacturer", "model"} {
		if _, ok := positions[required]; !ok {
			return sourceColumns{}, fmt.Errorf("source CSV missing required column %q", required)
		}
	}

	operator := -1
	if v, ok := positions["operator"]; ok {
		operator = v
	}
	return sourceColumns{
		icao24: positions["icao24"], registration: positions["registration"], typeCode: positions["type_code"],
		manufacturer: positions["manufacturer"], model: positions["model"], operator: operator,
	}, nil
}

func aircraftFromSourceRecord(record []string, cols sourceColumns) (aircraft, bool) {
	get := func(index int) string {
		if index < 0 || index >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[index])
	}
	item := aircraft{
		ICAO24:       normalizeICAO24(get(cols.icao24)),
		Registration: normalizeRegistration(get(cols.registration)),
		TypeCode:     strings.ToUpper(get(cols.typeCode)),
		Manufacturer: get(cols.manufacturer),
		Model:        get(cols.model),
		Operator:     get(cols.operator),
	}
	if !icao24RE.MatchString(item.ICAO24) || item.Registration == "" || item.TypeCode == "" || item.Manufacturer == "" || item.Model == "" {
		return aircraft{}, false
	}
	return item, true
}

func writeAircraftCSVAtomic(path string, items []aircraft) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	file, err := os.CreateTemp(dir, ".aircraft-*.csv")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	tmp := file.Name()
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(tmp)
	}

	w := csv.NewWriter(file)
	if err := w.Write([]string{"icao24", "registration", "type_code", "manufacturer", "model", "operator"}); err != nil {
		cleanup()
		return err
	}
	for _, item := range items {
		if err := w.Write([]string{item.ICAO24, item.Registration, item.TypeCode, item.Manufacturer, item.Model, item.Operator}); err != nil {
			cleanup()
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
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
		return fmt.Errorf("replace output: %w", err)
	}
	return nil
}

func normalizeHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "", "-", "", " ", "", ".", "")
	return replacer.Replace(value)
}
