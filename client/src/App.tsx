import { useEffect } from "react";
import { loadCrypto } from "./crypto/vaultCrypto";

export default function App() {
  useEffect(() => {
    (async () => {
      await loadCrypto();
      const salt = new Uint8Array(16).fill(7);
      const t0 = performance.now();
      const { encKey, authKey } = await window.vaultCrypto.deriveKeys(
        "test-password",
        salt,
      );
      console.log("derive took (ms):", performance.now() - t0);
      console.log("encKey", encKey);
      console.log("authKey", authKey);
      console.log("vaultKey", await window.vaultCrypto.newVaultKey());
    })();
  }, []);

  return <h1>Check the console</h1>;
}
