package main

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeAircraftImportFormat(t *testing.T) {
	if got := normalizeAircraftImportFormat("auto", "aircraft.csv.gz"); got != "tar1090" {
		t.Fatalf("auto gzip format = %q", got)
	}
	if got := normalizeAircraftImportFormat("auto", "source.csv"); got != "csv" {
		t.Fatalf("auto csv format = %q", got)
	}
	if got := normalizeAircraftImportFormat(" TAR1090 ", "source.csv"); got != "tar1090" {
		t.Fatalf("explicit format = %q", got)
	}
}

func TestImportTar1090CSV(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "aircraft.csv")
	output := filepath.Join(dir, "canonical.csv")
	data := strings.Join([]string{
		"4787a2;LN-NGM;B738;0;Boeing 737-800;2012;Norwegian;",
		"4ca8e6;EI-DCL;B738;0;Boeing 737-800;2005;Ryanair;",
		"4787a2;LN-DUP;B738;0;Duplicate;2013;Duplicate;",
		"BAD;ZZ-TEST;B738;0;Invalid;2020;Invalid;",
	}, "\n") + "\n"
	if err := os.WriteFile(input, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	stats, err := importTar1090CSV(input, output)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Rows != 4 || stats.Imported != 2 || stats.Duplicates != 1 || stats.Invalid != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	store, err := loadAircraftCSV(output)
	if err != nil {
		t.Fatalf("canonical output must load: %v", err)
	}
	item, ok := store.byICAO24["4787A2"]
	if !ok {
		t.Fatal("4787A2 missing after import")
	}
	if item.Model != "Boeing 737-800" || item.Operator != "Norwegian" || item.TypeCode != "B738" {
		t.Fatalf("unexpected aircraft: %+v", item)
	}
}

func TestImportTar1090Gzip(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "aircraft.csv.gz")
	output := filepath.Join(dir, "canonical.csv")
	file, err := os.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	if _, err := gz.Write([]byte("4787A2;LN-NGM;B738;0;Boeing 737-800;2012;Norwegian;\n")); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	stats, err := importAircraftByFormat(input, output, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Imported != 1 {
		t.Fatalf("imported = %d, want 1", stats.Imported)
	}
}

func TestAircraftFromTar1090RecordRejectsShortRow(t *testing.T) {
	if _, ok := aircraftFromTar1090Record([]string{"4787A2", "LN-NGM"}); ok {
		t.Fatal("short row accepted")
	}
}
