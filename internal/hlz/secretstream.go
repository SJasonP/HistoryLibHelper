package hlz

//lint:file-ignore SA1019 Interoperability with libsodium secretstream requires
// its specified Poly1305 construction; compatibility vectors cover this code.

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/poly1305"
)

const (
	streamHeaderSize      = 24
	streamOverhead        = 17
	streamTagMessage byte = 0
	streamTagFinal   byte = 3
)

type secretStream struct {
	key   [32]byte
	nonce [12]byte
}

func newSecretStreamPush(key []byte) (*secretStream, []byte, error) {
	header := make([]byte, streamHeaderSize)
	if _, err := rand.Read(header); err != nil {
		return nil, nil, err
	}
	s, err := newSecretStream(key, header)
	return s, header, err
}
func newSecretStream(key, header []byte) (*secretStream, error) {
	if len(key) != 32 || len(header) != 24 {
		return nil, fmt.Errorf("invalid secretstream key or header")
	}
	derived, err := chacha20.HChaCha20(key, header[:16])
	if err != nil {
		return nil, err
	}
	s := &secretStream{}
	copy(s.key[:], derived)
	copy(s.nonce[4:], header[16:])
	s.resetCounter()
	return s, nil
}

func (s *secretStream) push(message, ad []byte, tag byte) ([]byte, error) {
	block, cipher, mac, err := s.encryptParts(message, ad, tag)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 1+len(cipher)+16)
	out[0] = block[0]
	copy(out[1:], cipher)
	copy(out[1+len(cipher):], mac[:])
	s.advance(mac[:], tag)
	return out, nil
}

func (s *secretStream) pull(ciphertext, ad []byte) ([]byte, byte, error) {
	if len(ciphertext) < streamOverhead {
		return nil, 0, fmt.Errorf("ciphertext too short")
	}
	mlen := len(ciphertext) - streamOverhead
	encryptedTag := ciphertext[0]
	cipherMessage := ciphertext[1 : 1+mlen]
	stored := ciphertext[1+mlen:]
	stream, block, key, err := s.start()
	if err != nil {
		return nil, 0, err
	}
	defer wipe(block)
	defer wipe(key[:])
	block[0] = encryptedTag
	stream.XORKeyStream(block, block)
	tag := block[0]
	block[0] = encryptedTag
	mac := computeMAC(key, ad, block, cipherMessage)
	if subtle.ConstantTimeCompare(mac[:], stored) != 1 {
		return nil, 0, fmt.Errorf("authentication failed")
	}
	plain := make([]byte, mlen)
	stream.XORKeyStream(plain, cipherMessage)
	s.advance(stored, tag)
	return plain, tag, nil
}

func (s *secretStream) encryptParts(message, ad []byte, tag byte) ([]byte, []byte, [16]byte, error) {
	stream, block, key, err := s.start()
	if err != nil {
		return nil, nil, [16]byte{}, err
	}
	defer wipe(key[:])
	block[0] = tag
	stream.XORKeyStream(block, block)
	cipher := make([]byte, len(message))
	stream.XORKeyStream(cipher, message)
	mac := computeMAC(key, ad, block, cipher)
	return block, cipher, mac, nil
}

func (s *secretStream) start() (*chacha20.Cipher, []byte, *[32]byte, error) {
	stream, err := chacha20.NewUnauthenticatedCipher(s.key[:], s.nonce[:])
	if err != nil {
		return nil, nil, nil, err
	}
	block := make([]byte, 64)
	stream.XORKeyStream(block, block)
	key := new([32]byte)
	copy(key[:], block[:32])
	wipe(block)
	return stream, block, key, nil
}

func computeMAC(key *[32]byte, ad, tagBlock, cipher []byte) [16]byte {
	mac := poly1305.New(key)
	mac.Write(ad)
	writePadding(mac, len(ad))
	mac.Write(tagBlock)
	mac.Write(cipher)
	// Secretstream intentionally preserves libsodium's historical padding
	// expression. It writes mlen mod 16 zero bytes, not AEAD-style padding.
	if padding := len(cipher) & 0xf; padding != 0 {
		_, _ = mac.Write(make([]byte, padding))
	}
	var lengths [16]byte
	binary.LittleEndian.PutUint64(lengths[:8], uint64(len(ad)))
	binary.LittleEndian.PutUint64(lengths[8:], uint64(64+len(cipher)))
	mac.Write(lengths[:])
	var out [16]byte
	copy(out[:], mac.Sum(nil))
	return out
}
func writePadding(mac interface{ Write([]byte) (int, error) }, size int) {
	if remainder := size % 16; remainder != 0 {
		_, _ = mac.Write(make([]byte, 16-remainder))
	}
}
func (s *secretStream) advance(mac []byte, tag byte) {
	for i := 0; i < 8; i++ {
		s.nonce[4+i] ^= mac[i]
	}
	counter := binary.LittleEndian.Uint32(s.nonce[:4]) + 1
	binary.LittleEndian.PutUint32(s.nonce[:4], counter)
	if tag&2 != 0 || counter == 0 {
		s.rekey()
	}
}
func (s *secretStream) rekey() {
	stream, err := chacha20.NewUnauthenticatedCipher(s.key[:], s.nonce[:])
	if err != nil {
		panic(err)
	}
	material := make([]byte, 40)
	copy(material, s.key[:])
	copy(material[32:], s.nonce[4:])
	stream.XORKeyStream(material, material)
	copy(s.key[:], material[:32])
	copy(s.nonce[4:], material[32:])
	s.resetCounter()
	wipe(material)
}
func (s *secretStream) resetCounter() {
	for i := 0; i < 4; i++ {
		s.nonce[i] = 0
	}
	s.nonce[0] = 1
}
func (s *secretStream) clear() {
	wipe(s.key[:])
	wipe(s.nonce[:])
}
func wipe(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
