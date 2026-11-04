package crypto

import (
	"bytes"
	"testing"
)

var salt []byte = []byte("a27e9b57053fa7851ed18249cc2fe8d0dca4d281a7071b0b96430a27c946bc83")

var k = KDFParams{
	Time:        3,
	MemoryKiB:   64 * 1024,
	Parallelism: 4,
}

func TestDeriveKeysSuccess(t *testing.T) {
	encKey, authKey, err := DeriveKeys("My strong password", salt, k)

	if err != nil {
		t.Fatal(err)
	}
	t.Log(encKey)
	t.Log(authKey)
}

func TestEncAndAuthKeysDiffer(t *testing.T) {

	enc, auth, _ := DeriveKeys("My strong password", salt, k)
	if bytes.Equal(enc, auth) {
		t.Fatal("enc and auth keys must differ")
	}
}

func TestShortLengthSalt(t *testing.T) {

	if _, _, err := DeriveKeys("My strong password", salt[:10], k); err == nil {
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

func TestValutKeyDecryptionSuccess(t *testing.T) {
	enc, _, _ := DeriveKeys("right", salt, k)
	vk, _ := NewVaultKey()
	wrapped, _ := WrapVaultKey(enc, vk)

	bad, _, _ := DeriveKeys("right", salt, k)
	if _, err := UnwrapVaultKey(bad, wrapped); err != nil {
		t.Fatal("right password must be able to decrypt the Valult key")
	}
}

func TestWrongPasswordValutKeyFails(t *testing.T) {
	enc, _, _ := DeriveKeys("right", salt, k)
	vk, _ := NewVaultKey()
	wrapped, _ := WrapVaultKey(enc, vk)

	bad, _, _ := DeriveKeys("wrong", salt, k)
	if _, err := UnwrapVaultKey(bad, wrapped); err == nil {
		t.Fatal("wrong password should not unwrap the vault key")
	}
}

func TestSealAndOpenSuccess(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	plaintext := []byte("vault data")

	sealed, err := Seal(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	unsealed, err := Open(key, sealed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(unsealed, plaintext) {
		t.Fatalf("unsealed data = %q, want %q", unsealed, plaintext)
	}
}

func TestSealAndOpenFailure(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	plaintext := []byte("vault data")

	if _, err := Seal(key[:15], plaintext); err == nil {
		t.Fatal("Seal with an invalid key length should fail")
	}

	sealed, err := Seal(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Open(key[:15], sealed); err == nil {
		t.Fatal("Open with an invalid key length should fail")
	}
	if _, err := Open(key, sealed[:8]); err == nil {
		t.Fatal("Open with a short ciphertext should fail")
	}

	sealed[len(sealed)-1] ^= 1
	if _, err := Open(key, sealed); err == nil {
		t.Fatal("Open with tampered ciphertext should fail")
	}
}

func TestPasswordLoginFullFlow(t *testing.T) {
	const password = "correct horse battery staple"
	plaintext := []byte("secret vault data")

	encKey, _, err := DeriveKeys(password, salt, k)
	if err != nil {
		t.Fatal(err)
	}

	vaultKey, err := NewVaultKey()
	if err != nil {
		t.Fatal(err)
	}
	wrappedVaultKey, err := WrapVaultKey(encKey, vaultKey)
	if err != nil {
		t.Fatal(err)
	}
	sealedData, err := Seal(vaultKey, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("correct password unseals data", func(t *testing.T) {
		loginEncKey, _, err := DeriveKeys(password, salt, k)
		if err != nil {
			t.Fatal(err)
		}

		loginVaultKey, err := UnwrapVaultKey(loginEncKey, wrappedVaultKey)
		if err != nil {
			t.Fatal(err)
		}

		got, err := Open(loginVaultKey, sealedData)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, plaintext) {
			t.Fatalf("unsealed data = %q, want %q", got, plaintext)
		}
	})

	t.Run("wrong password cannot unseal data", func(t *testing.T) {
		loginEncKey, _, err := DeriveKeys("incorrect password", salt, k)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := UnwrapVaultKey(loginEncKey, wrappedVaultKey); err == nil {
			t.Fatal("wrong password unexpectedly unwrapped the vault key")
		}
	})
}
