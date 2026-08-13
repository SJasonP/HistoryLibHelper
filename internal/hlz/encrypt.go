package hlz

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/argon2"
	"golang.org/x/text/unicode/norm"
)

const encryptedChunkSize = 1_048_576

var encryptedMagic = []byte{'H', 'L', 'Z', 'E', 'N', 'C', 0, 1}

func EncryptFile(source, destination, password string) (resultErr error) {
	password = norm.NFC.String(password)
	if password == "" {
		return fmt.Errorf("password is required")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	key := argon2.IDKey([]byte(password), salt, 2, 64*1024, 1, 32)
	defer wipe(key)
	stream, streamHeader, err := newSecretStreamPush(key)
	if err != nil {
		return err
	}
	defer stream.clear()
	header := make([]byte, 0, 66)
	header = append(header, encryptedMagic...)
	header = append(header, 1, 1)
	var params [16]byte
	binary.BigEndian.PutUint32(params[:4], 2)
	binary.BigEndian.PutUint64(params[4:12], 64*1024*1024)
	binary.BigEndian.PutUint32(params[12:], encryptedChunkSize)
	header = append(header, params[:]...)
	header = append(header, salt...)
	header = append(header, streamHeader...)
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); resultErr == nil {
			resultErr = closeErr
		}
		if resultErr != nil {
			_ = os.Remove(destination)
		}
	}()
	if _, err = out.Write(header); err != nil {
		return err
	}
	current := make([]byte, encryptedChunkSize)
	n, err := io.ReadFull(in, current)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return err
	}
	current = current[:n]
	if len(current) == 0 {
		cipher, pushErr := stream.push(nil, header, streamTagFinal)
		if pushErr != nil {
			return pushErr
		}
		return writeFrame(out, cipher)
	}
	for {
		next := make([]byte, encryptedChunkSize)
		n, readErr := io.ReadFull(in, next)
		if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
			return readErr
		}
		next = next[:n]
		tag := streamTagMessage
		if len(next) == 0 {
			tag = streamTagFinal
		}
		cipher, err := stream.push(current, header, tag)
		if err != nil {
			return err
		}
		if err = writeFrame(out, cipher); err != nil {
			return err
		}
		if tag == streamTagFinal {
			return nil
		}
		current = next
	}
}
func writeFrame(w io.Writer, cipher []byte) error {
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(cipher)))
	if _, err := w.Write(size[:]); err != nil {
		return err
	}
	_, err := w.Write(cipher)
	return err
}
