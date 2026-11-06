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

const (
	blobV1     byte = 1
	nonceSize       = 12
	tagSize         = 16
	minBlobLen      = 1 + nonceSize + tagSize

	vaultKeyAAD   = "vaultkey/v1"
	itemAADPrefix = "item/v1:"

	keyLen = 32
)

var (
	ErrBlobTooShort       = errors.New("blob too short")
	ErrUnsupportedVersion = errors.New("unsupported blob version")
	ErrEmptyItemID        = errors.New("item id required")
)

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
func Seal(key, plaintext []byte, aad []byte) ([]byte, error) {
	gcm, err := newGCM(key)

	if err != nil {
		return nil, err
	}
	out := make([]byte, 1+gcm.NonceSize(), 1+gcm.NonceSize()+len(plaintext)+gcm.Overhead())
	out[0] = blobV1
	nonce := out[1:]
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(out, nonce, plaintext, aad), nil
}

// reverses Seal. GCM authenticates, so a wrong key or tampered data errors.
func Open(key, blob []byte, aad []byte) ([]byte, error) {

	if len(blob) < minBlobLen {
		return nil, ErrBlobTooShort
	}

	if blob[0] != blobV1 {
		return nil, ErrUnsupportedVersion
	}

	gcm, err := newGCM(key)

	if err != nil {
		return nil, err
	}

	if len(blob) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ct := blob[1:1+gcm.NonceSize()], blob[1+gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, aad)

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
func WrapVaultKey(encKey, vaultKey []byte) ([]byte, error) {
	return Seal(encKey, vaultKey, []byte(vaultKeyAAD))
}

// recovers the vault key on the client after login.
func UnwrapVaultKey(encKey, wrapped []byte) ([]byte, error) {
	return Open(encKey, wrapped, []byte(vaultKeyAAD))
}

func itemAAD(itemID string) []byte { return []byte(itemAADPrefix + itemID) }

func SealItem(vaultKey []byte, itemID string, plaintext []byte) ([]byte, error) {
	if itemID == "" {
		return nil, ErrEmptyItemID
	}
	return Seal(vaultKey, plaintext, itemAAD(itemID))
}

func OpenItem(vaultKey []byte, itemID string, blob []byte) ([]byte, error) {
	if itemID == "" {
		return nil, ErrEmptyItemID
	}
	return Open(vaultKey, blob, itemAAD(itemID))
}
