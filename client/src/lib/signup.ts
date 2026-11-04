import { vaultCrypto } from "./crypto";
import { signup as postSignup } from "./api";
import { toBase64 } from "./base64";
import { KDF_DEFAULTS, validateKdf } from "./kdf";
import type { SignupRequest } from "./types";

export async function signUp(email: string, password: string): Promise<void> {
  const kdf = { ...KDF_DEFAULTS };
  validateKdf(kdf);

  const salt = await vaultCrypto.generateRandomSalt();
  const { encKey, authKey } = await vaultCrypto.deriveKeys(
    password,
    salt,
    kdf.time,
    kdf.memoryKiB,
    kdf.parallelism,
  );
  const vaultKey = await vaultCrypto.newVaultKey();
  const wrappedVaultKey = await vaultCrypto.wrapVaultKey(encKey, vaultKey);

  const body: SignupRequest = {
    email: email.trim().toLowerCase(),
    authKey: toBase64(authKey),
    wrappedVaultKey: toBase64(wrappedVaultKey),
    salt: toBase64(salt),
    kdf,
  };

  const res = await postSignup(body);
  console.log("signup status:", res.status);
}
