import { useState } from "react";
import { vaultCrypto } from "./lib/crypto";

const PASSWORD = "My Super storong password";
const SALT = new Uint8Array(16).fill(1);

const toHex = (b: Uint8Array) =>
  Array.from(b)
    .map((x) => x.toString(16).padStart(2, "0"))
    .join("");

export default function App() {
  const [output, setOutput] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function run() {
    setLoading(true);
    setError("");

    try {
      const t0 = performance.now();
      const { encKey, authKey } = await vaultCrypto.deriveKeys(
        PASSWORD,
        SALT,
        3,
        64 * 1024,
        1,
      );
      const ms = performance.now() - t0;

      const vaultKey = await vaultCrypto.newVaultKey();
      const wrapped = await vaultCrypto.wrapVaultKey(encKey, vaultKey);
      const unwrapped = await vaultCrypto.unwrapVaultKey(encKey, wrapped);

      const plaintext = JSON.stringify({
        url: "https://example.com",
        username: "me",
        password: "hunter2",
      });
      const sealed = await vaultCrypto.seal(
        vaultKey,
        new TextEncoder().encode(plaintext),
      );
      const opened = await vaultCrypto.open(vaultKey, sealed);

      setOutput({
        "Derivation time": `${ms.toFixed(0)} ms`,
        "Enc key": toHex(encKey),
        "Auth key": toHex(authKey),
        "Vault key": toHex(vaultKey),
        "Wrapped vault key": toHex(wrapped),
        "Unwrapped vault key": toHex(unwrapped),
        Plaintext: plaintext,
        Sealed: toHex(sealed),
        Opened: new TextDecoder().decode(opened),
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main style={{ maxWidth: 700, margin: "40px auto", padding: 20 }}>
      <h1>Go WASM crypto</h1>
      <p>Password: {PASSWORD}</p>
      <p>Salt: 16 bytes of 0x01</p>

      <button onClick={run} disabled={loading}>
        {loading ? "Running..." : "Run"}
      </button>

      {error && <p style={{ color: "red" }}>{error}</p>}

      {Object.entries(output).map(([label, value]) => (
        <section key={label}>
          <h3>{label}</h3>
          <code style={{ overflowWrap: "anywhere" }}>{value}</code>
        </section>
      ))}
    </main>
  );
}
