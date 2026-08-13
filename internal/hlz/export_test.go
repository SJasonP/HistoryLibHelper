package hlz

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/argon2"
	"golang.org/x/text/unicode/norm"
	"historylibhelper/internal/model"
)

func TestExportProducesValidV1Archive(t *testing.T) {
	output := filepath.Join(t.TempDir(), "test.hlz")
	records := []model.Record{{URL: "https://b.example", Timestamp: 1_700_000_001_000_000, Browser: "Firefox"}, {URL: "https://a.example", Title: "A", Timestamp: 1_700_000_000_000_000, Browser: "Google Chrome"}}
	result, err := Export(context.Background(), records, output, "test", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.RecordCount != 2 {
		t.Fatalf("count=%d", result.RecordCount)
	}
	z, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	files := map[string]*zip.File{}
	for _, f := range z.File {
		files[f.Name] = f
	}
	var manifest map[string]any
	decodeZipJSON(t, files["manifest.json"], &manifest)
	if manifest["format"] != "historylib" || manifest["record_schema"] != "hl_record_v1" {
		t.Fatalf("manifest=%v", manifest)
	}
	var chunks []chunkEntry
	decodeZipJSON(t, files["indexes/chunks.json"], &chunks)
	if len(chunks) != 1 || chunks[0].Count != 2 {
		t.Fatalf("chunks=%v", chunks)
	}
	r, err := files[chunks[0].Path].Open()
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	r.Close()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if chunks[0].SHA != hex.EncodeToString(sum[:]) {
		t.Fatal("chunk checksum mismatch")
	}
}

func TestExportProducesDecryptablePasswordProtectedArchive(t *testing.T) {
	output := filepath.Join(t.TempDir(), "protected.hlz")
	records := []model.Record{{URL: "https://example.com", Timestamp: 1_700_000_000_000_000}}
	password := "pa\u0308ssword"
	if _, err := Export(context.Background(), records, output, "test", password, nil); err != nil {
		t.Fatal(err)
	}
	encrypted, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := decryptTestArchive(encrypted, "pässword")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(plain), int64(len(plain)))
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) == 0 {
		t.Fatal("decrypted archive is empty")
	}
	if _, err = decryptTestArchive(encrypted, "wrong password"); err == nil {
		t.Fatal("wrong password unexpectedly decrypted archive")
	}
}

func TestExportReplacesExistingArchiveOnlyAfterSuccess(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "existing.HLZ")
	if err := os.WriteFile(output, []byte("old archive"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Export(
		context.Background(),
		[]model.Record{{URL: "https://example.com", Timestamp: 1_700_000_000_000_000}},
		output,
		"test",
		"",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Output != output {
		t.Fatalf("output=%q", result.Output)
	}
	if _, err = zip.OpenReader(output); err != nil {
		t.Fatalf("replacement is not a valid archive: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(directory, ".existing.HLZ.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary outputs remain: %v", matches)
	}
}

func TestExportRejectsEmptyOutputPath(t *testing.T) {
	if _, err := Export(context.Background(), nil, "  ", "test", "", nil); err == nil {
		t.Fatal("empty output path was accepted")
	}
}

func TestCancelledExportPreservesExistingArchive(t *testing.T) {
	output := filepath.Join(t.TempDir(), "existing.hlz")
	original := []byte("existing archive")
	if err := os.WriteFile(output, original, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Export(ctx, nil, output, "test", "", nil); err == nil {
		t.Fatal("cancelled export succeeded")
	}
	actual, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, original) {
		t.Fatalf("existing output changed: %q", actual)
	}
}

func decryptTestArchive(data []byte, password string) ([]byte, error) {
	if len(data) < 66 || !bytes.Equal(data[:8], encryptedMagic) {
		return nil, io.ErrUnexpectedEOF
	}
	header := data[:66]
	salt := header[26:42]
	streamHeader := header[42:66]
	key := argon2.IDKey([]byte(norm.NFC.String(password)), salt, 2, 64*1024, 1, 32)
	defer wipe(key)
	stream, err := newSecretStream(key, streamHeader)
	if err != nil {
		return nil, err
	}
	var plain bytes.Buffer
	for offset := 66; offset < len(data); {
		if len(data)-offset < 4 {
			return nil, io.ErrUnexpectedEOF
		}
		size := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		if size < streamOverhead || len(data)-offset < size {
			return nil, io.ErrUnexpectedEOF
		}
		message, tag, pullErr := stream.pull(data[offset:offset+size], header)
		if pullErr != nil {
			return nil, pullErr
		}
		plain.Write(message)
		offset += size
		if tag == streamTagFinal {
			if offset != len(data) {
				return nil, fmt.Errorf("data follows final frame")
			}
			return plain.Bytes(), nil
		}
	}
	return nil, io.ErrUnexpectedEOF
}
func decodeZipJSON(t *testing.T, file *zip.File, target any) {
	t.Helper()
	if file == nil {
		t.Fatal("missing zip entry")
	}
	r, err := file.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err = json.NewDecoder(r).Decode(target); err != nil {
		t.Fatal(err)
	}
}
