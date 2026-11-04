package crypto

import (
	"bytes"
	"slices"
	"testing"
)

var salt []byte = bytes.Repeat([]byte{1}, 16)
var password = "My strong password"
var k = KDFParams{
	Time:        1,
	MemoryKiB:   8 * 1024,
	Parallelism: 1,
}

func TestFullLoginFlow(t *testing.T) {
	secret := []byte("secret vault data")

	enc, _, err := DeriveKeys(password, salt, k)
	if err != nil {
		t.Fatalf("DeriveKeys() error: %v", err)
	}

	vaultKey, err := NewVaultKey()
	if err != nil {
		t.Fatalf("NewVaultKey() error: %v", err)
	}

	wrapped, err := WrapVaultKey(enc, vaultKey)
	if err != nil {
		t.Fatalf("WrapVaultKey() error: %v", err)
	}

	sealed, err := Seal(vaultKey, secret)
	if err != nil {
		t.Fatalf("Seal() error: %v", err)
	}

	t.Run("correct password unseals the data", func(t *testing.T) {
		enc, _, err := DeriveKeys(password, salt, k)
		if err != nil {
			t.Fatalf("DeriveKeys() error: %v", err)
		}

		unwrapped, err := UnwrapVaultKey(enc, wrapped)
		if err != nil {
			t.Fatalf("UnwrapVaultKey() error: %v", err)
		}

		opened, err := Open(unwrapped, sealed)
		if err != nil {
			t.Fatalf("Open() error: %v", err)
		}

		if !slices.Equal(opened, secret) {
			t.Fatal("opened data must match the original secret")
		}
	})

	t.Run("wrong password cannot unwrap the vault key", func(t *testing.T) {
		wrongEnc, _, err := DeriveKeys("incorrect password", salt, k)
		if err != nil {
			t.Fatalf("DeriveKeys() error: %v", err)
		}

		if _, err := UnwrapVaultKey(wrongEnc, wrapped); err == nil {
			t.Fatal("unwrap must fail with a key derived from the wrong password")
		}
	})
}

func TestDeriveKeys(t *testing.T) {
	t.Run("returns two 32-byte keys", func(t *testing.T) {
		enc, auth, err := DeriveKeys(password, salt, k)

		if err != nil {
			t.Fatalf("DeriveKeys() error: %v", err)
		}

		if len(enc) != 32 {
			t.Fatalf("encKey length must be 32")
		}

		if len(auth) != 32 {
			t.Fatalf("authKey length must be 32")
		}
	})

	t.Run("encKey and authKey differ", func(t *testing.T) {
		enc, auth, err := DeriveKeys(password, salt, k)

		if err != nil {
			t.Fatalf("DeriveKeys() error: %v", err)
		}

		if slices.Equal(enc, auth) {
			t.Fatal("encKey and authKey should be different")
		}
	})

	t.Run("is deterministic for the same inputs", func(t *testing.T) {
		enc, auth, err := DeriveKeys(password, salt, k)
		enc2, auth2, err2 := DeriveKeys(password, salt, k)

		if err != nil || err2 != nil {
			t.Fatalf("DeriveKeys() error: %v", err)
		}

		if !(slices.Equal(enc, enc2) && slices.Equal(auth, auth2)) {
			t.Fatal("encKey and authKey should be deterministic for same inputs")
		}
	})

	t.Run("gives different keys for a different salt", func(t *testing.T) {
		salt2 := bytes.Repeat([]byte{2}, 16)
		enc, auth, err := DeriveKeys(password, salt, k)
		enc2, auth2, err2 := DeriveKeys(password, salt2, k)

		if err != nil || err2 != nil {
			t.Fatalf("DeriveKeys() error: %v", err)
		}

		if slices.Equal(enc, enc2) || slices.Equal(auth, auth2) {
			t.Fatal("encKey and authKey should be different for different salt")
		}
	})

	t.Run("rejects a salt shorter than 16 bytes", func(t *testing.T) {
		if _, _, err := DeriveKeys(password, salt[:10], k); err == nil {
			t.Fatal("salt lenght less then 16 must be rejected")
		}
	})

}

func TestNewValutKey(t *testing.T) {
	t.Run("returns 32 random bytes", func(t *testing.T) {
		a, err := NewVaultKey()
		b, err2 := NewVaultKey()

		if err != nil || err2 != nil {
			t.Fatalf("NewVaultKey() error: %v", err)
		}

		if len(a) != 32 || len(b) != 32 {
			t.Fatal("valut key lenth must be 32")
		}

		if slices.Equal(a, b) {
			t.Fatal("valut keys must be random")
		}
	})
}

func TestWrapUnwrapValutKey(t *testing.T) {
	t.Run("round-trips with right password", func(t *testing.T) {
		enc, _, err := DeriveKeys(password, salt, k)
		if err != nil {
			t.Fatalf("DeriveKeys() error: %v", err)
		}

		val, err := NewVaultKey()

		if err != nil {
			t.Fatalf("NewVaultKey() error: %v", err)
		}

		wrapped, err := WrapVaultKey(enc, val)

		if err != nil {
			t.Fatalf("WrapVaultKey() error: %v", err)
		}

		unwrapped, err := UnwrapVaultKey(enc, wrapped)

		if err != nil {
			t.Fatalf("UnwrapVaultKey() error: %v", err)
		}

		if !slices.Equal(val, unwrapped) {
			t.Fatal("round trip with same password must result in same valut key")
		}
	})

	t.Run("fails with the wrong password", func(t *testing.T) {
		enc, _, err := DeriveKeys(password, salt, k)
		enc_wrong, _, err2 := DeriveKeys("Wrong passwd", salt, k)

		if err != nil || err2 != nil {
			t.Fatalf("DeriveKeys() error: %v", err)
		}

		val, err := NewVaultKey()
		if err != nil {
			t.Fatalf("NewVaultKey() error: %v", err)
		}

		wrapped, err := WrapVaultKey(enc, val)
		if err != nil {
			t.Fatalf("WrapVaultKey() error: %v", err)
		}

		_, err = UnwrapVaultKey(enc_wrong, wrapped)

		if err == nil {
			t.Fatal("must fail with wrong password")
		}

	})
}

func TestSealOpen(t *testing.T) {
	key := bytes.Repeat([]byte{6}, 32)
	plainText := []byte("{\"username\" : \"user1\",\"password\":\"Super secure passwd\"}")
	t.Run("round trips with same key", func(t *testing.T) {
		sealed, err := Seal(key, plainText)
		if err != nil {
			t.Fatalf("Seal() error: %v", err)
		}
		opened, err := Open(key, sealed)
		if err != nil {
			t.Fatalf("Open() error: %v", err)
		}

		if string(opened) != string(plainText) {
			t.Fatal("round trip with same key result in same plaintext")
		}
	})

	t.Run("gives a different blob each time (random nonce)", func(t *testing.T) {
		a, err := Seal(key, plainText)
		b, err2 := Seal(key, plainText)

		if err != nil || err2 != nil {
			t.Fatalf("Seal() error: %v", err)
		}

		if slices.Equal(a, b) {
			t.Fatal("must give random nounce to blob each time")
		}
	})

	t.Run("rejects an invalid key length", func(t *testing.T) {
		_, err := Seal(key[:15], plainText)

		if err == nil {
			t.Fatalf("key length less then 32 must be rejected by Seal()")
		}

		sealed, err := Seal(key, plainText)

		if err != nil {
			t.Fatalf("Seal() error: %v", err)
		}

		_, err = Open(key[:15], sealed)

		if err == nil {
			t.Fatalf("key length less then 32 must be rejected by Open()")
		}
	})

	t.Run("rejects short ciphertext", func(t *testing.T) {
		sealed, err := Seal(key, plainText)

		if err != nil {
			t.Fatalf("Seal() error: %v", err)
		}

		_, err = Open(key, sealed[:10])

		if err == nil {
			t.Fatalf("Must reject short ciphertext")
		}

	})

	t.Run("rejects tampered ciphertext", func(t *testing.T) {
		sealed, err := Seal(key, plainText)

		if err != nil {
			t.Fatalf("Seal() error: %v", err)
		}

		sealed[20] ^= 1
		_, err = Open(key, sealed)

		if err == nil {
			t.Fatalf("Must reject tampered ciphertext")
		}

	})

	t.Run("rejects the wrong key", func(t *testing.T) {
		sealed, err := Seal(key, plainText)

		if err != nil {
			t.Fatalf("Seal() error: %v", err)
		}

		key2 := bytes.Repeat([]byte{9}, 32)
		_, err = Open(key2, sealed)

		if err == nil {
			t.Fatalf("Must reject wrong key")
		}

	})

}

func TestGenerateRandomSalt(t *testing.T) {
	t.Run("returns 16 bytes", func(t *testing.T) {
		s, err := GenerateRandomSalt()
		if err != nil {
			t.Fatalf("generateRandomSalt() error: %v", err)
		}

		if len(s) != 16 {
			t.Fatalf("salt length must be 16, got %d", len(s))
		}
	})

	t.Run("gives a different salt each time", func(t *testing.T) {
		a, err := GenerateRandomSalt()
		b, err2 := GenerateRandomSalt()

		if err != nil || err2 != nil {
			t.Fatalf("generateRandomSalt() error: %v %v", err, err2)
		}

		if slices.Equal(a, b) {
			t.Fatal("salts must be random")
		}
	})

	t.Run("is not all zeros", func(t *testing.T) {
		s, err := GenerateRandomSalt()
		if err != nil {
			t.Fatalf("generateRandomSalt() error: %v", err)
		}

		if bytes.Equal(s, make([]byte, 16)) {
			t.Fatal("salt must not be all zero bytes")
		}
	})

	t.Run("is accepted by DeriveKeys", func(t *testing.T) {
		s, err := GenerateRandomSalt()
		if err != nil {
			t.Fatalf("generateRandomSalt() error: %v", err)
		}

		if _, _, err := DeriveKeys(password, s, k); err != nil {
			t.Fatalf("a generated salt must be valid for DeriveKeys: %v", err)
		}
	})
}

// This is to compare hash with the client test case
func TestKnownAnswer(t *testing.T) {
	salt := bytes.Repeat([]byte{1}, 16)
	p := KDFParams{Time: 1, MemoryKiB: 8 * 1024, Parallelism: 1}
	enc, auth, err := DeriveKeys("My second super strong password", salt, p)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("enc:  %x", enc)
	t.Logf("auth: %x", auth)
}
