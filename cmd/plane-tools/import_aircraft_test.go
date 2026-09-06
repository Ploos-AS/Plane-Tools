package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapSourceColumnsAliases(t *testing.T) {
	cols, err := mapSourceColumns([]string{"hex", "reg", "type", "maker", "aircraft model", "owner"})
	if err != nil {
		t.Fatal(err)
	}
	if cols.icao24 != 0 || cols.registration != 1 || cols.typeCode != 2 || cols.manufacturer != 3 || cols.model != 4 || cols.operator != 5 {
		t.Fatalf("unexpected mapping: %+v", cols)
	}
}

func TestImportAircraftCSVNormalizesDeduplicatesAndSorts(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "source.csv")
	output := filepath.Join(dir, "aircraft.csv")
	data := strings.Join([]string{
		"hex,reg,type,maker,model,owner",
		"4ca8e6,ei-dcl,b738,Boeing,737-800,Ryanair",
		"4787a2,ln-ngm,b738,Boeing,737-800,Norwegian",
		"4787a2,ln-other,b738,Boeing,737-800,Duplicate Hex",
		"4851f5,ln-ngm,b789,Boeing,787-9,Duplicate Reg",
		"badhex,zz-test,b738,Boeing,737-800,Invalid",
	}, "\n") + "\n"
	if err := os.WriteFile(input, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	stats, err := importAircraftCSV(input, output)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Rows != 5 || stats.Imported != 2 || stats.Duplicates != 2 || stats.Invalid != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	wantPrefix := "icao24,registration,type_code,manufacturer,model,operator\n4787A2,LN-NGM,B738,Boeing,737-800,Norwegian\n4CA8E6,EI-DCL,B738,Boeing,737-800,Ryanair\n"
	if text != wantPrefix {
		t.Fatalf("unexpected output:\n%s", text)
	}

	store, err := loadAircraftStore(output)
	if err != nil {
		t.Fatalf("canonical output must be loadable: %v", err)
	}
	if got, ok := store.byRegistration["LN-NGM"]; !ok || got.ICAO24 != "4787A2" {
		t.Fatalf("lookup failed after import: %+v %v", got, ok)
	}
}

func TestImportAircraftCSVRejectsMissingRequiredColumn(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "source.csv")
	if err := os.WriteFile(input, []byte("hex,reg,type,maker\n4787A2,LN-NGM,B738,Boeing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := importAircraftCSV(input, filepath.Join(dir, "out.csv"))
	if err == nil || !strings.Contains(err.Error(), "model") {
		t.Fatalf("expected missing model error, got %v", err)
	}
}

func TestImportAircraftCSVDoesNotReplaceOutputWhenNoRowsValid(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "source.csv")
	output := filepath.Join(dir, "aircraft.csv")
	if err := os.WriteFile(input, []byte("icao24,registration,type_code,manufacturer,model\nBAD,X,B738,Boeing,737-800\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := importAircraftCSV(input, output)
	if err == nil {
		t.Fatal("expected import failure")
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "sentinel\n" {
		t.Fatalf("existing output was modified: %q", got)
	}
}
