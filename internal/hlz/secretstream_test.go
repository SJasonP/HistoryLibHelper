package hlz

import (
	"encoding/hex"
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestArgon2MatchesLibsodiumVector(t *testing.T) {
	password := []byte("pässword")
	salt := decodeHex(t, "000102030405060708090a0b0c0d0e0f")
	expected := "38fa25f5c7d37dd415adf0f9ba4838e59e8a086a4aa42f4a6f6c09e3d44309a4"
	actual := argon2.IDKey(password, salt, 2, 64*1024, 1, 32)
	if hex.EncodeToString(actual) != expected {
		t.Fatalf("key mismatch\nactual:   %x\nexpected: %s", actual, expected)
	}
}

func TestSecretStreamMatchesLibsodiumVector(t *testing.T) {
	key := decodeHex(t, "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	header := decodeHex(t, "49dd4399acb15cbbe710cdbede6df6a568d43832e696bf58")
	ad := decodeHex(t, "486973746f72794c696220686561646572")
	message := decodeHex(t, "63726f73732d6c616e67756167652073656372657473747265616d2074657374")
	expected := decodeHex(t, "b06749f8aec4121b603bc3c77c97f23120ab7c55739659018b300d957f2c635840f0bda0851cca7be4b4e031231c2bae9e")
	push, err := newSecretStream(key, header)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := push.push(message, ad, streamTagFinal)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(actual) != hex.EncodeToString(expected) {
		t.Fatalf("cipher mismatch\nactual:   %x\nexpected: %x", actual, expected)
	}
	pull, err := newSecretStream(key, header)
	if err != nil {
		t.Fatal(err)
	}
	plain, tag, err := pull.pull(expected, ad)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != string(message) || tag != streamTagFinal {
		t.Fatalf("plain=%q tag=%d", plain, tag)
	}
}

func TestSecretStreamMultipleFramesMatchLibsodiumVector(t *testing.T) {
	key := decodeHex(t, "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	header := decodeHex(t, "604dfc6bfe0b7d680b73541bab784cb7e4b8d2f397d36a21")
	ad := []byte("HistoryLib header")
	expected := [][]byte{
		decodeHex(t, "863b50ef80644b739cab95e6ac14b618043048cc627a5392d0d2e9cd496d"),
		decodeHex(t, "c01235d7a009adc077731c9a306cb59bcaa36699e96ee0a530a04e70f9cf002a96808b9a72"),
	}
	stream, err := newSecretStream(key, header)
	if err != nil {
		t.Fatal(err)
	}
	first, err := stream.push([]byte("first message"), ad, streamTagMessage)
	if err != nil {
		t.Fatal(err)
	}
	second, err := stream.push([]byte("second final message"), ad, streamTagFinal)
	if err != nil {
		t.Fatal(err)
	}
	for index, actual := range [][]byte{first, second} {
		if hex.EncodeToString(actual) != hex.EncodeToString(expected[index]) {
			t.Fatalf("frame %d mismatch\nactual:   %x\nexpected: %x", index, actual, expected[index])
		}
	}
}

func decodeHex(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}
