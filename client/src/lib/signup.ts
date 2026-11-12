import { vaultCrypto } from "./crypto";
import { errorMessage, signup as postSignup, throwIfRateLimited } from "./api";
import { toBase64 } from "./base64";
import { KDF_DEFAULTS, validateKdf } from "./kdf";
import type { AuthResponse, SignupRequest } from "./types";
import { getDeviceInfo } from "./device";

export async function signUp(
  email: string,
  password: string,
): Promise<{ auth: AuthResponse; vaultKey: Uint8Array }> {
  const kdf = { ...KDF_DEFAULTS };
  validateKdf(kdf);

  const salt = await vaultCrypto.generateRandomSalt();
  const pw = password.normalize("NFKC");
  const { encKey, authKey } = await vaultCrypto.deriveKeys(
    pw,
    salt,
    kdf.time,
    kdf.memoryKiB,
    kdf.parallelism,
  );
  const vaultKey = await vaultCrypto.newVaultKey();
  const wrappedVaultKey = await vaultCrypto.wrapVaultKey(encKey, vaultKey);
  const { deviceId, deviceName } = getDeviceInfo();

  const body: SignupRequest = {
    email: email.trim().toLowerCase(),
    deviceId,
    deviceName,
    authKey: toBase64(authKey),
    wrappedVaultKey: toBase64(wrappedVaultKey),
    salt: toBase64(salt),
    kdf,
  };

  const res = await postSignup(body);
  throwIfRateLimited(res);
  if (!res.ok)
    throw new Error(await errorMessage(res, "Signup failed, try again"));
  return { auth: (await res.json()) as AuthResponse, vaultKey };
}
