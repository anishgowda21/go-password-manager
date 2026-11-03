package main

import (
	"errors"
	"syscall/js"

	"gitlab.com/anishgowda21/go-password-manager/wasm/internal/crypto"
)

var params = crypto.KDFParams{Time: 3, MemoryKiB: 64 * 1024, Parallelism: 1}

func toBytes(v js.Value) []byte {
	b := make([]byte, v.Get("length").Int())
	js.CopyBytesToGo(b, v)
	return b
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
			return js.Undefined(), err
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
	salt := toBytes(args[1])
	p := crypto.KDFParams{
		Time:        uint32(args[2].Int()),
		MemoryKiB:   uint32(args[3].Int()),
		Parallelism: uint8(args[4].Int()),
	}

	return promise(func() (js.Value, error) {
		enc, auth, err := crypto.DeriveKeys(password, salt, p)
		if err != nil {
			return js.Undefined(), err
		}
		out := js.Global().Get("Object").New()
		out.Set("encKey", toJS(enc))
		out.Set("authKey", toJS(auth))
		return out, nil
	})
}

func main() {
	js.Global().Set("vaultCrypto", map[string]any{
		"deriveKeys":  js.FuncOf(deriveKeys),
		"newVaultKey": js.FuncOf(newVaultKey),
	})
	select {}
}
