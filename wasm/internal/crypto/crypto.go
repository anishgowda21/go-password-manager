//Basically all the crytographic functions live here
// insted of doing the front end cryptography in TS I will use wasm

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
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

// encrypts plaintext with AES-256-GCM. Output is nonce || ciphertext.
func Seal(key, plaintext []byte) ([]byte, error) {
	gcm, err := newGCM(key)

	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// reverses Seal. GCM authenticates, so a wrong key or tampered data errors.
func Open(key, blob []byte) ([]byte, error) {
	gcm, err := newGCM(key)

	if err != nil {
		return nil, err
	}

	if len(blob) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ct := blob[:gcm.NonceSize()], blob[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)

}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)

	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}

func GenerateRandomSalt() ([]byte, error) {
	s := make([]byte, 16)
	if _, err := rand.Read(s); err != nil {
		return nil, err
	}

	return s, nil
}

// encrypts the vault key with the password-derived encKey.
func WrapVaultKey(encKey, vaultKey []byte) ([]byte, error) { return Seal(encKey, vaultKey) }

// recovers the vault key on the client after login.
func UnwrapVaultKey(encKey, wrapped []byte) ([]byte, error) { return Open(encKey, wrapped) }
