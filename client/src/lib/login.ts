import { vaultCrypto } from "./crypto";
import { prelogin, login as postLogin, errorMessage } from "./api";
import { fromBase64, toBase64 } from "./base64";
import { validateKdf } from "./kdf";
import { getDeviceInfo } from "./device";
import type { AuthResponse, PreloginResponse } from "./types";

export async function logIn(
  rawEmail: string,
  password: string,
): Promise<{ auth: AuthResponse; vaultKey: Uint8Array }> {
  const email = rawEmail.trim().toLowerCase();

  const preRes = await prelogin(email);
  if (!preRes.ok)
    throw new Error(await errorMessage(preRes, "Could not start login"));

  const pre = (await preRes.json()) as PreloginResponse;
  if (
    typeof pre?.salt !== "string" ||
    typeof pre?.kdf !== "object" ||
    pre.kdf === null
  ) {
    throw new Error("Unexpected prelogin response");
  }
  validateKdf(pre.kdf);

  const salt = fromBase64(pre.salt);
  if (salt.length < 16) throw new Error("Invalid salt from server");

  const pw = password.normalize("NFKC");
  const { encKey, authKey } = await vaultCrypto.deriveKeys(
    pw,
    salt,
    pre.kdf.time,
    pre.kdf.memoryKiB,
    pre.kdf.parallelism,
  );

  const { deviceId, deviceName } = getDeviceInfo();
  const res = await postLogin({
    email,
    authKey: toBase64(authKey),
    deviceId,
    deviceName,
  });
  if (!res.ok) throw new Error(await errorMessage(res, "Login failed"));

  const auth = (await res.json()) as AuthResponse;
  const vaultKey = await vaultCrypto.unwrapVaultKey(
    encKey,
    fromBase64(auth.wrappedVaultKey),
  );

  return { auth, vaultKey };
}
