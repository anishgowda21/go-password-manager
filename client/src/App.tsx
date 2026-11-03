import { useEffect, useState } from "react";
import { loadCrypto } from "./crypto/vaultCrypto";

interface PreloginResponse {
  kdfSettings: {
    kdfType: number;
    iterations: number;
    memory: number;
    parallelism: number;
  };
  salt: string;
}

export default function App() {
  const [password, setPassword] = useState("");
  const [prelogin, setPrelogin] = useState<{
    salt: string;
    saltBytes: Uint8Array;
    kdfSettings: PreloginResponse["kdfSettings"];
  } | null>(null);
  const [encKey, setEncKey] = useState("");
  const [authKey, setAuthKey] = useState("");
  const [vaultKey, setVaultKey] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const controller = new AbortController();

    async function fetchPrelogin() {
      try {
        const response = await fetch("http://localhost:8090/api/prelogin", {
          signal: controller.signal,
        });
        if (!response.ok) {
          throw new Error(`Pre-login request failed (${response.status})`);
        }

        const data = (await response.json()) as PreloginResponse;
        const binarySalt = atob(data.salt);
        const saltBytes = Uint8Array.from(binarySalt, (character) =>
          character.charCodeAt(0),
        );

        setPrelogin({
          salt: data.salt,
          saltBytes,
          kdfSettings: data.kdfSettings,
        });
      } catch (err) {
        if (!controller.signal.aborted) {
          setError(
            err instanceof Error
              ? err.message
              : "Could not load pre-login settings",
          );
        }
      }
    }

    void fetchPrelogin();
    return () => controller.abort();
  }, []);

  async function handleDeriveKeys() {
    if (!password) {
      setError("Please enter a password");
      return;
    }

    if (!prelogin) {
      setError("Pre-login settings are not available");
      return;
    }

    setLoading(true);
    setError("");

    try {
      await loadCrypto();

      const t0 = performance.now();

      const { encKey, authKey } = await window.vaultCrypto.deriveKeys(
        password,
        prelogin.saltBytes,
        3,
        prelogin.kdfSettings.memory,
        prelogin.kdfSettings.parallelism,
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

      {prelogin ? (
        <p>
          Salt: <code>{prelogin.salt}</code>
        </p>
      ) : (
        <p>{error ? "Unable to load salt." : "Loading salt..."}</p>
      )}

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

      <button onClick={handleDeriveKeys} disabled={loading || !prelogin}>
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
