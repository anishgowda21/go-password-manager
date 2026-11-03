//Basically all the crytographic functions live here
// insted of doing the front end cryptography in TS I will use wasm

package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

const keyLen = 32

type KDFParams struct {
	Time        uint32
	MemoryKiB   uint32
	Parallelism uint8
}

func DeriveKeys(password string, salt []byte, p KDFParams) (encKey, authKey []byte, err error) {
	if len(salt) < 16 {
		return nil, nil, errors.New("salt too short")
	}
	master := argon2.IDKey([]byte(password), salt, p.Time, p.MemoryKiB, p.Parallelism, keyLen)

	if encKey, err = expand(master, "vault/enc/v1"); err != nil {
		return nil, nil, err
	}

	if authKey, err = expand(master, "vault/auth/v1"); err != nil {
		return nil, nil, err
	}

	return encKey, authKey, err

}

func expand(master []byte, label string) ([]byte, error) {
	out := make([]byte, keyLen)
	_, err := io.ReadFull(hkdf.Expand(sha256.New, master, []byte(label)), out)
	return out, err
}

func NewVaultKey() ([]byte, error) {
	k := make([]byte, keyLen)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}

	return k, nil
}
