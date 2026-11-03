import { useState } from "react";
import { loadCrypto } from "./crypto/vaultCrypto";

const salt = crypto.getRandomValues(new Uint8Array(16));

export default function App() {
  const [password, setPassword] = useState("");
  const [encKey, setEncKey] = useState("");
  const [authKey, setAuthKey] = useState("");
  const [vaultKey, setVaultKey] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function handleDeriveKeys() {
    if (!password) {
      setError("Please enter a password");
      return;
    }

    setLoading(true);
    setError("");

    try {
      await loadCrypto();

      const t0 = performance.now();

      const { encKey, authKey } = await window.vaultCrypto.deriveKeys(
        password,
        salt,
        3,
        65536,
        1,
      );

      console.log("Derivation took:", performance.now() - t0, "ms");

      const toHex = (bytes: Uint8Array) =>
        Array.from(bytes)
          .map((b) => b.toString(16).padStart(2, "0"))
          .join("");

      setEncKey(toHex(encKey));
      setAuthKey(toHex(authKey));

      const newKey = await window.vaultCrypto.newVaultKey();
      setVaultKey(toHex(newKey));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main style={{ maxWidth: 600, margin: "40px auto", padding: 20 }}>
      <h1>Go WASM Crypto Demo</h1>

      <label>Password</label>
      <input
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        placeholder="Enter your password"
        style={{
          display: "block",
          width: "100%",
          padding: 10,
          margin: "10px 0",
        }}
      />

      <button onClick={handleDeriveKeys} disabled={loading}>
        {loading ? "Deriving keys..." : "Derive Keys"}
      </button>

      {error && <p style={{ color: "red" }}>{error}</p>}

      {encKey && (
        <section>
          <h3>Encryption Key</h3>
          <code style={{ overflowWrap: "anywhere" }}>{encKey}</code>
        </section>
      )}

      {authKey && (
        <section>
          <h3>Authentication Key</h3>
          <code style={{ overflowWrap: "anywhere" }}>{authKey}</code>
        </section>
      )}

      {vaultKey && (
        <section>
          <h3>Vault Key</h3>
          <code style={{ overflowWrap: "anywhere" }}>{vaultKey}</code>
        </section>
      )}
    </main>
  );
}
