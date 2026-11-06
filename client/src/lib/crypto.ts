type Pending = { resolve: (v: any) => void; reject: (e: Error) => void };

let worker: Worker | null = null;
let nextId = 0;
const pending = new Map<number, Pending>();

function getWorker(): Worker {
  if (worker) return worker;

  const w = new Worker("/crypto.worker.js");
  w.onmessage = (e) => {
    const { id, ok, result, error } = e.data;
    const p = pending.get(id);
    if (!p) return;
    pending.delete(id);
    if (ok) p.resolve(result);
    else p.reject(new Error(error));
  };
  w.onerror = () => {
    for (const p of pending.values())
      p.reject(new Error("crypto worker crashed"));
    pending.clear();
    worker = null; // next call starts a fresh worker
  };

  worker = w;
  return w;
}

function call<T>(method: string, ...args: unknown[]): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const id = nextId++;
    pending.set(id, { resolve, reject });
    getWorker().postMessage({ id, method, args });
  });
}

export const vaultCrypto = {
  deriveKeys: (
    password: string,
    salt: Uint8Array,
    time: number,
    memory: number,
    parallelism: number,
  ) =>
    call<{ encKey: Uint8Array<ArrayBuffer>; authKey: Uint8Array<ArrayBuffer> }>(
      "deriveKeys",
      password,
      salt,
      time,
      memory,
      parallelism,
    ),
  newVaultKey: () => call<Uint8Array<ArrayBuffer>>("newVaultKey"),
  generateRandomSalt: () => call<Uint8Array<ArrayBuffer>>("generateRandomSalt"),
  sealItem: (key: Uint8Array, itemId: string, plaintext: Uint8Array) =>
    call<Uint8Array<ArrayBuffer>>("sealItem", key, itemId, plaintext),
  openItem: (key: Uint8Array, itemId: string, blob: Uint8Array) =>
    call<Uint8Array<ArrayBuffer>>("openItem", key, itemId, blob),
  wrapVaultKey: (encKey: Uint8Array, vaultKey: Uint8Array) =>
    call<Uint8Array<ArrayBuffer>>("wrapVaultKey", encKey, vaultKey),
  unwrapVaultKey: (encKey: Uint8Array, wrapped: Uint8Array) =>
    call<Uint8Array<ArrayBuffer>>("unwrapVaultKey", encKey, wrapped),
};
