import { ApiError, authFetch, errorMessage } from "./api";
import { fromBase64, toBase64 } from "./base64";
import { vaultCrypto } from "./crypto";
import type { ItemData, ServerItem, VaultItem } from "./types";

const encoder = new TextEncoder();
const decoder = new TextDecoder();

async function ensureOk(res: Response, fallback: string) {
  if (!res.ok)
    throw new ApiError(res.status, await errorMessage(res, fallback));
}

function parseItemData(v: unknown): ItemData {
  const o = (v ?? {}) as Record<string, unknown>;
  const s = (k: string) => (typeof o[k] === "string" ? (o[k] as string) : "");
  return {
    name: s("name"),
    url: s("url"),
    username: s("username"),
    password: s("password"),
    notes: s("notes"),
  };
}

async function seal(
  vaultKey: Uint8Array,
  id: string,
  data: ItemData,
): Promise<string> {
  const sealed = await vaultCrypto.sealItem(
    vaultKey,
    id,
    encoder.encode(JSON.stringify(data)),
  );
  return toBase64(sealed);
}

async function open(vaultKey: Uint8Array, s: ServerItem): Promise<VaultItem> {
  const plain = await vaultCrypto.openItem(vaultKey, s.id, fromBase64(s.blob));
  return {
    id: s.id,
    revision: s.revision,
    data: parseItemData(JSON.parse(decoder.decode(plain))),
  };
}

export async function fetchItems(token: string, vaultKey: Uint8Array) {
  const res = await authFetch("/api/sync", token, "GET");
  await ensureOk(res, "Sync Failed");

  const body = (await res.json()) as { items: ServerItem[] };
  const results = await Promise.allSettled(
    body.items.map((s) => open(vaultKey, s)),
  );
  const items = results.flatMap((r) =>
    r.status === "fulfilled" ? r.value : [],
  );

  return { items, failed: results.length - items.length };
}

export async function createItem(
  token: string,
  vaultKey: Uint8Array,
  data: ItemData,
): Promise<VaultItem> {
  const id = crypto.randomUUID();
  const res = await authFetch("/api/items", token, "POST", {
    id,
    blob: await seal(vaultKey, id, data),
  });
  await ensureOk(res, "Could not save item");
  const s = (await res.json()) as ServerItem;
  return { id: s.id, revision: s.revision, data };
}

export async function updateItem(
  token: string,
  vaultKey: Uint8Array,
  item: VaultItem,
  data: ItemData,
): Promise<VaultItem> {
  const res = await authFetch(`/api/items/${item.id}`, token, "PUT", {
    revision: item.revision,
    blob: await seal(vaultKey, item.id, data),
  });

  await ensureOk(res, "Could not update item");
  const s = (await res.json()) as ServerItem;
  return { id: item.id, revision: s.revision, data };
}

export async function deleteItem(token: string, id: string): Promise<void> {
  const res = await authFetch(`/api/items/${id}`, token, "DELETE");
  await ensureOk(res, "Could not delete item");
}
