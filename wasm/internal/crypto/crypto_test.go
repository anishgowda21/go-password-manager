package crypto

import (
	"bytes"
	"testing"
)

func TestDeriveKeysSuccess(t *testing.T) {
	salt := "a27e9b57053fa7851ed18249cc2fe8d0dca4d281a7071b0b96430a27c946bc83"
	k := KDFParams{
		Time:        3,
		MemoryKiB:   64 * 1024,
		Parallelism: 4,
	}
	encKey, authKey, err := DeriveKeys("My strong password", []byte(salt), k)

	if err != nil {
		t.Fatal(err)
	}
	t.Log(encKey)
	t.Log(authKey)
}

func TestEncAndAuthKeysDiffer(t *testing.T) {
	salt := "a27e9b57053fa7851ed18249cc2fe8d0dca4d281a7071b0b96430a27c946bc83"
	k := KDFParams{
		Time:        3,
		MemoryKiB:   64 * 1024,
		Parallelism: 4,
	}
	enc, auth, _ := DeriveKeys("My strong password", []byte(salt), k)
	if bytes.Equal(enc, auth) {
		t.Fatal("enc and auth keys must differ")
	}
}

func TestShortLengthSalt(t *testing.T) {
	salt := "a27e9b5705"
	k := KDFParams{
		Time:        3,
		MemoryKiB:   64 * 1024,
		Parallelism: 4,
	}
	if _, _, err := DeriveKeys("My strong password", []byte(salt), k); err == nil {
		t.Fatal("salt length short then 16 must fail")
	}
}

func TestValutKeyGen(t *testing.T) {
	key, err := NewVaultKey()
	if err != nil {
		t.Fatal(nil)
	}
	t.Log(key)
}
