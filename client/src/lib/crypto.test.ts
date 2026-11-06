import { beforeAll, describe, expect, it } from "vitest";
import { loadVaultCrypto } from "../test/loadWasm";

type VC = Awaited<ReturnType<typeof loadVaultCrypto>>;

const PASSWORD = "My second super strong password";
const SALT = new Uint8Array(16).fill(1);
const KDF = { time: 1, memory: 8 * 1024, parallelism: 1 };
const encoder = new TextEncoder();
const decoder = new TextDecoder();

const hex = (b: Uint8Array) =>
  Array.from(b)
    .map((x) => x.toString(16).padStart(2, "0"))
    .join("");

let vc: VC;

const derive = (password: string, salt = SALT) =>
  vc.deriveKeys(password, salt, KDF.time, KDF.memory, KDF.parallelism);

beforeAll(async () => {
  vc = await loadVaultCrypto();
});

describe("deriveKeys", () => {
  it("returns two 32-byte keys", async () => {
    const { encKey, authKey } = await derive(PASSWORD);
    expect(encKey).toHaveLength(32);
    expect(authKey).toHaveLength(32);
  });

  it("encKey and authKey differ", async () => {
    const { encKey, authKey } = await derive(PASSWORD);
    expect(encKey).not.toEqual(authKey);
  });

  it("is deterministic for the same inputs", async () => {
    const a = await derive(PASSWORD);
    const b = await derive(PASSWORD);
    expect(a.encKey).toEqual(b.encKey);
    expect(a.authKey).toEqual(b.authKey);
  });

  it("gives different keys for a different salt", async () => {
    const a = await derive(PASSWORD);
    const b = await derive(PASSWORD, new Uint8Array(16).fill(2));
    expect(a.encKey).not.toEqual(b.encKey);
  });

  it("rejects a salt shorter than 16 bytes", async () => {
    await expect(derive(PASSWORD, SALT.slice(0, 10))).rejects.toThrow();
  });

  it("rejects a non-Uint8Array salt", async () => {
    await expect(
      // @ts-expect-error wrong type on purpose
      vc.deriveKeys(PASSWORD, "salt", KDF.time, KDF.memory, KDF.parallelism),
    ).rejects.toThrow(/Uint8Array/);
  });

  it("rejects non-numeric params", async () => {
    await expect(
      // @ts-expect-error wrong type on purpose
      vc.deriveKeys(PASSWORD, SALT, "1", KDF.memory, KDF.parallelism),
    ).rejects.toThrow(/numbers/);
  });

  it("matches the Go known answer", async () => {
    const { encKey, authKey } = await derive(PASSWORD);
    expect(hex(encKey)).toBe(
      "83fde789899bb699438961c1e83ba09450256f74d4597d203ea98b6711bd63fc",
    );
    expect(hex(authKey)).toBe(
      "515b3f8fbc8ae37bd06a49ed2b960e29104b31475cb003dc9edd091d37561a6b",
    );
  });
});

describe("newVaultKey", () => {
  it("returns 32 random bytes", async () => {
    const a = await vc.newVaultKey();
    const b = await vc.newVaultKey();
    expect(a).toHaveLength(32);
    expect(a).not.toEqual(b);
  });
});

describe("wrapVaultKey / unwrapVaultKey", () => {
  it("round-trips with the right password", async () => {
    const { encKey } = await derive(PASSWORD);
    const vaultKey = await vc.newVaultKey();
    const wrapped = await vc.wrapVaultKey(encKey, vaultKey);
    const unwrapped = await vc.unwrapVaultKey(encKey, wrapped);
    expect(unwrapped).toEqual(vaultKey);
  });

  it("fails with the wrong password", async () => {
    const right = await derive(PASSWORD);
    const wrong = await derive("wrong password");
    const vaultKey = await vc.newVaultKey();
    const wrapped = await vc.wrapVaultKey(right.encKey, vaultKey);
    await expect(vc.unwrapVaultKey(wrong.encKey, wrapped)).rejects.toThrow();
  });
});

describe("sealItem / openItem", () => {
  const key = new Uint8Array(32).fill(1);
  const plaintext = encoder.encode(
    '{"username":"user1","password":"Super secure passwd"}',
  );

  it("round-trips", async () => {
    const sealed = await vc.sealItem(key, "item-1", plaintext);
    const opened = await vc.openItem(key, "item-1", sealed);
    expect(decoder.decode(opened)).toBe(decoder.decode(plaintext));
  });

  it("gives a different blob each time (random nonce)", async () => {
    const a = await vc.sealItem(key, "item-1", plaintext);
    const b = await vc.sealItem(key, "item-1", plaintext);
    expect(a).not.toEqual(b);
  });

  it("starts with version byte 1 and has the expected length", async () => {
    const sealed = await vc.sealItem(key, "item-1", plaintext);
    expect(sealed[0]).toBe(1);
    expect(sealed.length).toBe(1 + 12 + plaintext.length + 16);
  });

  it("rejects an invalid key length", async () => {
    await expect(
      vc.sealItem(key.slice(0, 15), "item-1", plaintext),
    ).rejects.toThrow();
    const sealed = await vc.sealItem(key, "item-1", plaintext);
    await expect(
      vc.openItem(key.slice(0, 15), "item-1", sealed),
    ).rejects.toThrow();
  });

  it("rejects a short ciphertext", async () => {
    const sealed = await vc.sealItem(key, "item-1", plaintext);
    await expect(
      vc.openItem(key, "item-1", sealed.slice(0, 8)),
    ).rejects.toThrow(/too short/);
  });

  it("rejects an unsupported version", async () => {
    const sealed = (await vc.sealItem(key, "item-1", plaintext)).slice();
    sealed[0] = 2;
    await expect(vc.openItem(key, "item-1", sealed)).rejects.toThrow(
      /unsupported/,
    );
  });

  it("rejects tampered ciphertext", async () => {
    const sealed = await vc.sealItem(key, "item-1", plaintext);
    const tampered = sealed.slice();
    tampered[tampered.length - 1] ^= 1;
    await expect(vc.openItem(key, "item-1", tampered)).rejects.toThrow();
  });

  it("rejects the wrong key", async () => {
    const sealed = await vc.sealItem(key, "item-1", plaintext);
    await expect(
      vc.openItem(new Uint8Array(32).fill(2), "item-1", sealed),
    ).rejects.toThrow();
  });

  it("rejects a blob swapped to another item id", async () => {
    const sealed = await vc.sealItem(key, "item-1", plaintext);
    await expect(vc.openItem(key, "item-2", sealed)).rejects.toThrow();
  });

  it("rejects an empty item id", async () => {
    await expect(vc.sealItem(key, "", plaintext)).rejects.toThrow(/item id/);
    const sealed = await vc.sealItem(key, "item-1", plaintext);
    await expect(vc.openItem(key, "", sealed)).rejects.toThrow(/item id/);
  });

  it("rejects bad argument types", async () => {
    await expect(
      // @ts-expect-error wrong type on purpose
      vc.sealItem("not bytes", "item-1", plaintext),
    ).rejects.toThrow(/Uint8Array/);
    await expect(
      // @ts-expect-error wrong type on purpose
      vc.sealItem(key, 123, plaintext),
    ).rejects.toThrow();
  });
});

describe("blob separation", () => {
  it("wrapped key length is 61 bytes", async () => {
    const { encKey } = await derive(PASSWORD);
    const wrapped = await vc.wrapVaultKey(encKey, await vc.newVaultKey());
    expect(wrapped).toHaveLength(61);
  });

  it("an item blob cannot be unwrapped as a vault key", async () => {
    const key = new Uint8Array(32).fill(1);
    const blob = await vc.sealItem(key, "item-1", new Uint8Array(32));
    await expect(vc.unwrapVaultKey(key, blob)).rejects.toThrow();
  });

  it("a wrapped key cannot be opened as an item", async () => {
    const key = new Uint8Array(32).fill(1);
    const wrapped = await vc.wrapVaultKey(key, await vc.newVaultKey());
    await expect(vc.openItem(key, "item-1", wrapped)).rejects.toThrow();
  });
});

describe("full login flow", () => {
  let wrapped: Uint8Array;
  let sealed: Uint8Array;

  beforeAll(async () => {
    const { encKey } = await derive(PASSWORD);
    const vaultKey = await vc.newVaultKey();
    wrapped = await vc.wrapVaultKey(encKey, vaultKey);
    sealed = await vc.sealItem(
      vaultKey,
      "item-1",
      encoder.encode("secret vault data"),
    );
  });

  it("correct password unseals the data", async () => {
    const { encKey } = await derive(PASSWORD);
    const vaultKey = await vc.unwrapVaultKey(encKey, wrapped);
    const opened = await vc.openItem(vaultKey, "item-1", sealed);
    expect(decoder.decode(opened)).toBe("secret vault data");
  });

  it("wrong password cannot unwrap the vault key", async () => {
    const { encKey } = await derive("incorrect password");
    await expect(vc.unwrapVaultKey(encKey, wrapped)).rejects.toThrow();
  });
});

describe("generateRandomSalt", () => {
  it("returns 16 bytes", async () => {
    const salt = await vc.generateRandomSalt();
    expect(salt).toBeInstanceOf(Uint8Array);
    expect(salt.length).toBe(16);
  });

  it("gives a different salt each time", async () => {
    const a = await vc.generateRandomSalt();
    const b = await vc.generateRandomSalt();
    expect(a).not.toEqual(b);
  });

  it("is not all zeros", async () => {
    const salt = await vc.generateRandomSalt();
    expect(salt.some((byte) => byte !== 0)).toBe(true);
  });

  it("is accepted by deriveKeys", async () => {
    const salt = await vc.generateRandomSalt();
    await expect(derive(PASSWORD, salt)).resolves.toBeDefined();
  });
});
