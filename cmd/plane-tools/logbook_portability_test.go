package main

import (
	"strings"
	"testing"
)

func TestDecodeSpottingLogCSV(t *testing.T) {
	input := "id,observed_at,icao24,registration,spotting_location_id,airport_ident,notes\nobs-1,2026-09-07T08:00:00Z,4787A2,LN-NGM,,ENGM,arrival\n"
	items, err := decodeSpottingLogCSV(csvReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "obs-1" || items[0].ICAO24 != "4787A2" || items[0].AirportIdent != "ENGM" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestImportSpottingLogMergeAndReplace(t *testing.T) {
	store := &spottingLogStore{path: t.TempDir() + "/spotting-log.json", items: []spottingLogEntry{{ID: "old", ObservedAt: "2026-09-01T08:00:00Z", Registration: "LN-OLD"}}}
	incoming := []spottingLogEntry{{ID: "new", ObservedAt: "2026-09-07T08:00:00Z", Registration: "LN-NEW"}}
	result, err := store.importItems(incoming, "merge")
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || len(store.items) != 2 {
		t.Fatalf("merge result = %#v, items = %#v", result, store.items)
	}
	result, err = store.importItems(incoming, "replace")
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(store.items) != 1 || store.items[0].ID != "new" {
		t.Fatalf("replace result = %#v, items = %#v", result, store.items)
	}
}

func csvReader(input string) *csv.Reader {
	return csv.NewReader(strings.NewReader(input))
}
