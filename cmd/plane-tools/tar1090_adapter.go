package main

import (
	"compress/gzip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func importAircraftByFormat(inputPath, outputPath, format string) (importStats, error) {
	switch normalizeAircraftImportFormat(format, inputPath) {
	case "csv":
		return importAircraftCSV(inputPath, outputPath)
	case "tar1090":
		return importTar1090CSV(inputPath, outputPath)
	default:
		return importStats{}, fmt.Errorf("unsupported aircraft import format %q", format)
	}
}

func normalizeAircraftImportFormat(format, inputPath string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" || format == "auto" {
		if strings.HasSuffix(strings.ToLower(inputPath), ".gz") {
			return "tar1090"
		}
		return "csv"
	}
	return format
}

func importTar1090CSV(inputPath, outputPath string) (importStats, error) {
	reader, closeReader, err := openTar1090Reader(inputPath)
	if err != nil {
		return importStats{}, err
	}
	defer closeReader()

	r := csv.NewReader(reader)
	r.Comma = ';'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	stats := importStats{}
	byICAO := make(map[string]aircraft)
	byRegistration := make(map[string]string)

	for {
		record, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return stats, fmt.Errorf("read tar1090 row %d: %w", stats.Rows+1, err)
		}
		if len(record) == 0 || strings.TrimSpace(record[0]) == "" {
			continue
		}
		if stats.Rows == 0 && strings.EqualFold(strings.TrimSpace(record[0]), "ICAO") {
			continue
		}
		stats.Rows++

		item, ok := aircraftFromTar1090Record(record)
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
		return stats, errors.New("no valid tar1090 aircraft rows to import")
	}
	if err := writeAircraftCSVAtomic(outputPath, items); err != nil {
		return stats, err
	}
	return stats, nil
}

func aircraftFromTar1090Record(record []string) (aircraft, bool) {
	if len(record) < 5 {
		return aircraft{}, false
	}
	get := func(index int) string {
		if index < 0 || index >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[index])
	}

	item := aircraft{
		ICAO24:       normalizeICAO24(get(0)),
		Registration: normalizeRegistration(get(1)),
		TypeCode:     strings.ToUpper(get(2)),
		Model:        get(4),
		Operator:     get(6),
	}
	if !icao24RE.MatchString(item.ICAO24) || item.Registration == "" || item.TypeCode == "" {
		return aircraft{}, false
	}
	return item, true
}

func openTar1090Reader(path string) (io.Reader, func(), error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, func() {}, fmt.Errorf("open tar1090 input: %w", err)
	}
	closeFile := func() { _ = file.Close() }

	if !strings.HasSuffix(strings.ToLower(path), ".gz") {
		return file, closeFile, nil
	}

	gz, err := gzip.NewReader(file)
	if err != nil {
		closeFile()
		return nil, func() {}, fmt.Errorf("open gzip tar1090 input: %w", err)
	}
	return gz, func() {
		_ = gz.Close()
		closeFile()
	}, nil
}
