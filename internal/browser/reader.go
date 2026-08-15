package browser

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"historylibhelper/internal/model"
	_ "modernc.org/sqlite"
)

const chromiumEpochOffset = int64(11_644_473_600_000_000)

func Read(ctx context.Context, profile model.Profile) ([]model.Record, error) {
	tempDir, err := os.MkdirTemp("", "historylib-helper-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)
	if err := os.Chmod(tempDir, 0o700); err != nil {
		return nil, err
	}
	snapshot := filepath.Join(tempDir, "history.sqlite")
	if err := createSnapshot(ctx, profile.Database, snapshot); err != nil {
		return nil, fmt.Errorf("snapshot %s: %w", profile.Browser, err)
	}
	db, err := sql.Open("sqlite", sqliteURI(snapshot, "ro"))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	query := `SELECT u.url, COALESCE(u.title,''), v.visit_time FROM visits v JOIN urls u ON u.id=v.url WHERE u.url<>'' ORDER BY v.visit_time,v.id`
	offset := chromiumEpochOffset
	if profile.Engine == "firefox" {
		query = `SELECT p.url, COALESCE(p.title,''), h.visit_date FROM moz_historyvisits h JOIN moz_places p ON p.id=h.place_id WHERE p.url<>'' AND h.visit_date IS NOT NULL ORDER BY h.visit_date,h.id`
		offset = 0
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]model.Record, 0, 4096)
	for rows.Next() {
		var rawURL, title string
		var rawTime int64
		if err := rows.Scan(&rawURL, &title, &rawTime); err != nil {
			return nil, err
		}
		ts := rawTime - offset
		if ts <= 0 || strings.TrimSpace(rawURL) == "" {
			continue
		}
		result = append(result, model.Record{URL: rawURL, Title: title, Timestamp: ts, Browser: profile.Browser, Source: filepath.Base(profile.Database)})
	}
	return result, rows.Err()
}

func createSnapshot(ctx context.Context, source, destination string) error {
	db, err := sql.Open("sqlite", sqliteURI(source, "ro"))
	if err != nil {
		return err
	}
	defer db.Close()
	quoted := strings.ReplaceAll(destination, "'", "''")
	_, err = db.ExecContext(ctx, "VACUUM INTO '"+quoted+"'")
	if err == nil {
		err = os.Chmod(destination, 0o600)
	}
	return err
}

func sqliteURI(path, mode string) string {
	// url.URL treats a Windows drive letter as a URI authority when the path
	// still contains backslashes (file:C:%5CUsers...). modernc SQLite rejects
	// that URI before it even attempts to open the database. Convert native
	// separators and make drive-letter paths absolute URI paths instead.
	uriPath := filepath.ToSlash(path)
	if volume := filepath.VolumeName(path); volume != "" && !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	u := url.URL{Scheme: "file", Path: uriPath}
	q := u.Query()
	q.Set("mode", mode)
	q.Set("_pragma", "busy_timeout(5000)")
	u.RawQuery = q.Encode()
	return u.String()
}
