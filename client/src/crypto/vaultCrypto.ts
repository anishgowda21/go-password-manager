declare global {
  interface Window {
    Go: new () => {
      importObject: WebAssembly.Imports;
      run(instance: WebAssembly.Instance): Promise<void>;
    };
    vaultCrypto: {
      deriveKeys(
        password: string,
        salt: Uint8Array,
      ): Promise<{ encKey: Uint8Array; authKey: Uint8Array }>;
      newVaultKey(): Promise<Uint8Array>;
    };
  }
}

let ready: Promise<void> | null = null;

export function loadCrypto(): Promise<void> {
  if (!ready) {
    ready = (async () => {
      const go = new window.Go();
      const { instance } = await WebAssembly.instantiateStreaming(
        fetch("/crypto.wasm"),
        go.importObject,
      );
      void go.run(instance);
    })();
  }
  return ready;
}
