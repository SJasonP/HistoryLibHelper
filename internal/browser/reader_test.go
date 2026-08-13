package browser

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"historylibhelper/internal/model"
	_ "modernc.org/sqlite"
)

func TestReadChromiumSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "History")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{`CREATE TABLE urls(id INTEGER PRIMARY KEY,url TEXT,title TEXT)`, `CREATE TABLE visits(id INTEGER PRIMARY KEY,url INTEGER,visit_time INTEGER)`, `INSERT INTO urls VALUES(1,'https://example.com','Example')`, `INSERT INTO visits VALUES(1,1,13335473600000000)`}
	for _, statement := range statements {
		if _, err = db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()
	records, err := Read(context.Background(), model.Profile{Browser: "Google Chrome", Database: path, Engine: "chromium"})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Timestamp != 1_691_000_000_000_000 || records[0].URL != "https://example.com" {
		t.Fatalf("unexpected records: %#v", records)
	}
}

func TestReadFirefoxSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "places.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{`CREATE TABLE moz_places(id INTEGER PRIMARY KEY,url TEXT,title TEXT)`, `CREATE TABLE moz_historyvisits(id INTEGER PRIMARY KEY,place_id INTEGER,visit_date INTEGER)`, `INSERT INTO moz_places VALUES(1,'https://mozilla.org','Mozilla')`, `INSERT INTO moz_historyvisits VALUES(1,1,1691000000000000)`}
	for _, statement := range statements {
		if _, err = db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()
	records, err := Read(context.Background(), model.Profile{Browser: "Firefox", Database: path, Engine: "firefox"})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Timestamp != 1_691_000_000_000_000 {
		t.Fatalf("unexpected records: %#v", records)
	}
}

func TestReadIncludesUncheckpointedWALRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "History")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	statements := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA wal_autocheckpoint=0`,
		`CREATE TABLE urls(id INTEGER PRIMARY KEY,url TEXT,title TEXT)`,
		`CREATE TABLE visits(id INTEGER PRIMARY KEY,url INTEGER,visit_time INTEGER)`,
		`INSERT INTO urls VALUES(1,'https://example.com/live','Live')`,
		`INSERT INTO visits VALUES(1,1,13335473600000000)`,
	}
	for _, statement := range statements {
		if _, err = db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	records, err := Read(context.Background(), model.Profile{
		Browser:  "Google Chrome",
		Database: path,
		Engine:   "chromium",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].URL != "https://example.com/live" {
		t.Fatalf("unexpected records: %#v", records)
	}
}
