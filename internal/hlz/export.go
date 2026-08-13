package hlz

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"historylibhelper/internal/model"
)

const chunkTarget = 50_000

type record struct {
	URL        string `json:"u"`
	Timestamp  int64  `json:"ts"`
	Title      string `json:"t,omitempty"`
	VisitCount int    `json:"vc,omitempty"`
	Browser    string `json:"sb,omitempty"`
	ImportedAt int64  `json:"ia,omitempty"`
	RawTime    int64  `json:"rt,omitempty"`
	Source     string `json:"sf,omitempty"`
}
type chunkEntry struct {
	ID    int    `json:"id"`
	Path  string `json:"path"`
	Count int    `json:"record_count"`
	Min   int64  `json:"min_ts"`
	Max   int64  `json:"max_ts"`
	SHA   string `json:"sha256"`
}
type bucket struct {
	Count    int
	Min, Max int64
}
type yearEntry struct {
	Year  int   `json:"year"`
	Count int   `json:"record_count"`
	Min   int64 `json:"min_ts"`
	Max   int64 `json:"max_ts"`
}

func Export(ctx context.Context, records []model.Record, output, version, password string, progress func(int, int)) (model.ExportResult, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return model.ExportResult{}, fmt.Errorf("output path is required")
	}
	if !strings.EqualFold(filepath.Ext(output), ".hlz") {
		output += ".hlz"
	}
	if err := ctx.Err(); err != nil {
		return model.ExportResult{}, err
	}

	sort.SliceStable(records, func(i, j int) bool { return records[i].Timestamp < records[j].Timestamp })
	temp, err := os.MkdirTemp("", "historylib-hlz-")
	if err != nil {
		return model.ExportResult{}, err
	}
	defer os.RemoveAll(temp)
	root := filepath.Join(temp, "archive")
	for _, dir := range []string{"chunks", "indexes"} {
		if err = os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			return model.ExportResult{}, err
		}
	}
	now := time.Now().UnixMicro()
	chunks := []chunkEntry{}
	years := map[int]*bucket{}
	months := map[string]*bucket{}
	days := map[string]*bucket{}
	for start, id := 0, 1; start < len(records); start, id = start+chunkTarget, id+1 {
		if err := ctx.Err(); err != nil {
			return model.ExportResult{}, err
		}
		end := min(start+chunkTarget, len(records))
		file := fmt.Sprintf("%08d.jsonl", id)
		path := filepath.Join(root, "chunks", file)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return model.ExportResult{}, err
		}
		h := sha256.New()
		w := io.MultiWriter(f, h)
		enc := json.NewEncoder(w)
		for _, item := range records[start:end] {
			r := record{item.URL, item.Timestamp, item.Title, 1, item.Browser, now, item.Timestamp, item.Source}
			if err = enc.Encode(r); err != nil {
				f.Close()
				return model.ExportResult{}, err
			}
			addBuckets(item.Timestamp, years, months, days)
		}
		if err = f.Close(); err != nil {
			return model.ExportResult{}, err
		}
		chunks = append(chunks, chunkEntry{id, "chunks/" + file, end - start, records[start].Timestamp, records[end-1].Timestamp, hex.EncodeToString(h.Sum(nil))})
		if progress != nil {
			progress(end, len(records))
		}
	}
	if err = writeJSON(filepath.Join(root, "indexes/chunks.json"), chunks); err != nil {
		return model.ExportResult{}, err
	}
	if err = writeIndexes(root, years, months, days); err != nil {
		return model.ExportResult{}, err
	}
	minTs, maxTs := int64(0), int64(0)
	if len(records) > 0 {
		minTs, maxTs = records[0].Timestamp, records[len(records)-1].Timestamp
	}
	manifest := map[string]any{"format": "historylib", "format_version": 1, "created_at_usec": now, "app_name": "HistoryLibHelper", "app_version": version, "record_schema": "hl_record_v1", "record_count": len(records), "chunk_count": len(chunks), "time_range_usec": map[string]int64{"min": minTs, "max": maxTs}, "chunk_encoding": "jsonl", "chunk_target_records": chunkTarget, "feature_flags": []string{"prebuilt_time_indexes_v1"}, "indexes": map[string]string{"chunks": "indexes/chunks.json", "years": "indexes/years.json", "months": "indexes/months.json", "days": "indexes/days.json"}}
	if err = writeJSON(filepath.Join(root, "manifest.json"), manifest); err != nil {
		return model.ExportResult{}, err
	}
	if err = ctx.Err(); err != nil {
		return model.ExportResult{}, err
	}
	stagedFile, err := os.CreateTemp(filepath.Dir(output), "."+filepath.Base(output)+".tmp-*")
	if err != nil {
		return model.ExportResult{}, err
	}
	stagedOutput := stagedFile.Name()
	if err = stagedFile.Close(); err != nil {
		return model.ExportResult{}, err
	}
	defer os.Remove(stagedOutput)

	plainOutput := stagedOutput
	if password != "" {
		plainOutput = filepath.Join(temp, "plain.hlz")
	}
	if err = zipDirectory(root, plainOutput); err != nil {
		return model.ExportResult{}, err
	}
	if password != "" {
		if err = EncryptFile(plainOutput, stagedOutput, password); err != nil {
			return model.ExportResult{}, err
		}
	}
	if err = syncFile(stagedOutput); err != nil {
		return model.ExportResult{}, err
	}
	if err = os.Rename(stagedOutput, output); err != nil {
		return model.ExportResult{}, err
	}
	return model.ExportResult{Output: output, RecordCount: len(records), MinTimeUsec: minTs, MaxTimeUsec: maxTs}, nil
}

func syncFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func addBuckets(ts int64, years map[int]*bucket, months, days map[string]*bucket) {
	t := time.UnixMicro(ts).In(time.Local)
	add(years, t.Year(), ts)
	add(months, t.Format("2006-01"), ts)
	add(days, t.Format("2006-01-02"), ts)
}
func add[K comparable](m map[K]*bucket, k K, ts int64) {
	b := m[k]
	if b == nil {
		m[k] = &bucket{1, ts, ts}
		return
	}
	b.Count++
	if ts < b.Min {
		b.Min = ts
	}
	if ts > b.Max {
		b.Max = ts
	}
}
func writeIndexes(root string, years map[int]*bucket, months, days map[string]*bucket) error {
	ys := make([]yearEntry, 0, len(years))
	for k, v := range years {
		ys = append(ys, yearEntry{k, v.Count, v.Min, v.Max})
	}
	sort.Slice(ys, func(i, j int) bool { return ys[i].Year < ys[j].Year })
	if err := writeJSON(filepath.Join(root, "indexes/years.json"), ys); err != nil {
		return err
	}
	if err := writeNamed(filepath.Join(root, "indexes/months.json"), "month", months, func(a, b string) bool { return a < b }); err != nil {
		return err
	}
	return writeNamed(filepath.Join(root, "indexes/days.json"), "day", days, func(a, b string) bool { return a < b })
}
func writeNamed(path, key string, m map[string]*bucket, less func(string, string) bool) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return less(keys[i], keys[j]) })
	items := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		v := m[k]
		items = append(items, map[string]any{key: k, "record_count": v.Count, "min_ts": v.Min, "max_ts": v.Max})
	}
	return writeJSON(path, items)
}
func writeJSON(path string, v any) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(v)
}
func zipDirectory(root, output string) error {
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	err = filepath.Walk(root, func(path string, info os.FileInfo, e error) error {
		if e != nil || info.IsDir() {
			return e
		}
		rel, _ := filepath.Rel(root, path)
		w, e := zw.Create(filepath.ToSlash(rel))
		if e != nil {
			return e
		}
		r, e := os.Open(path)
		if e != nil {
			return e
		}
		defer r.Close()
		_, e = io.Copy(w, r)
		return e
	})
	if closeErr := zw.Close(); err == nil {
		err = closeErr
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}
