import "../../public/wasm_exec.js";
import { readFileSync } from "node:fs";
import type { vaultCrypto } from "../lib/crypto.js";

type VaultCrypto = typeof vaultCrypto;
let ready: Promise<VaultCrypto> | null = null;

export function loadVaultCrypto(): Promise<VaultCrypto> {
  ready ??= (async () => {
    const g = globalThis as any;
    const go = new g.Go();
    const bytes = readFileSync(
      new URL("../../public/crypto.wasm", import.meta.url),
    );
    const { instance } = await WebAssembly.instantiate(bytes, go.importObject);

    go.run(instance);

    for (let i = 0; i < 500 && !g.vaultCrypto; i++) {
      await new Promise((r) => setTimeout(r, 10));
    }
    if (!g.vaultCrypto) throw new Error("vaultCrypto did not initialise");
    return g.vaultCrypto as VaultCrypto;
  })();
  return ready;
}
