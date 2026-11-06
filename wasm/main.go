package main

import (
	"errors"
	"fmt"
	"syscall/js"

	"gitlab.com/anishgowda21/go-password-manager/wasm/internal/crypto"
)

var params = crypto.KDFParams{Time: 3, MemoryKiB: 64 * 1024, Parallelism: 1}

func toBytes(v js.Value) ([]byte, error) {
	if !v.InstanceOf(js.Global().Get("Uint8Array")) {
		return nil, errors.New("expected Uint8Array")
	}
	b := make([]byte, v.Get("length").Int())
	js.CopyBytesToGo(b, v)
	return b, nil
}

func toJS(b []byte) js.Value {
	a := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(a, b)
	return a
}

// promise runs fn and turns its result into a JS Promise.
func promise(fn func() (js.Value, error)) any {
	var handler js.Func

	handler = js.FuncOf(func(_ js.Value, args []js.Value) any {
		resolve, reject := args[0], args[1]
		go func() {
			defer handler.Release()
			v, err := fn()
			if err != nil {
				reject.Invoke(js.Global().Get("Error").New(err.Error()))
				return
			}
			resolve.Invoke(v)
		}()
		return nil
	})
	return js.Global().Get("Promise").New(handler)
}

func newVaultKey(_ js.Value, _ []js.Value) any {
	return promise(func() (js.Value, error) {
		k, err := crypto.NewVaultKey()
		if err != nil {
			return js.Undefined(), errors.New("cannot produce valut key, error: " + err.Error())
		}
		return toJS(k), nil
	})
}

func generateRandomSalt(_ js.Value, _ []js.Value) any {
	return promise(func() (js.Value, error) {
		k, err := crypto.GenerateRandomSalt()
		if err != nil {
			return js.Undefined(), errors.New("cannot produce salt, error: " + err.Error())
		}
		return toJS(k), nil
	})
}

func deriveKeys(_ js.Value, args []js.Value) any {
	if len(args) != 5 {
		return promise(func() (js.Value, error) {
			return js.Undefined(), errors.New("deriveKeys(password, salt, time, memory, parallelism) needs 5 args")
		})
	}
	password := args[0].String()
	salt, errToBytes := toBytes(args[1])
	p := crypto.KDFParams{
		Time:        uint32(args[2].Int()),
		MemoryKiB:   uint32(args[3].Int()),
		Parallelism: uint8(args[4].Int()),
	}

	return promise(func() (js.Value, error) {
		enc, auth, err := crypto.DeriveKeys(password, salt, p)
		if errToBytes != nil {
			return js.Undefined(), err
		}
		if err != nil {
			return js.Undefined(), errors.New("cannot derive keys, error: " + err.Error())
		}
		out := js.Global().Get("Object").New()
		out.Set("encKey", toJS(enc))
		out.Set("authKey", toJS(auth))
		return out, nil
	})
}

func twoBytesArgs(name string, fn func(a, b []byte) ([]byte, error)) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) != 2 {
			return promise(func() (js.Value, error) {
				return js.Undefined(), errors.New(name + " needs 2 args")
			})
		}
		a, b := toBytes(args[0]), toBytes(args[1])
		return promise(func() (js.Value, error) {
			out, err := fn(a, b)
			if err != nil {
				return js.Undefined(), fmt.Errorf("%s: %w", name, err)
			}
			return toJS(out), nil
		})
	})
}

func main() {
	js.Global().Set("vaultCrypto", map[string]any{
		"deriveKeys":         js.FuncOf(deriveKeys),
		"newVaultKey":        js.FuncOf(newVaultKey),
		"generateRandomSalt": js.FuncOf(generateRandomSalt),
		"seal":               twoBytesArgs("seal", crypto.Seal),
		"open":               twoBytesArgs("open", crypto.Open),
		"wrapVaultKey":       twoBytesArgs("wrapVaultKey", crypto.WrapVaultKey),
		"unwrapVaultKey":     twoBytesArgs("unwrapVaultKey", crypto.UnwrapVaultKey),
	})
	select {}
}
