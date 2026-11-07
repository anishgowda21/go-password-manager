import { vaultCrypto } from "./crypto";
import { fromBase64 } from "./base64";
import { validateKdf } from "./kdf";
import { useSession } from "../store/session";

export async function unlockVault(password: string): Promise<void> {
  const { kdf, salt, wrappedVaultKey } = useSession.getState();
  if (!kdf || !salt || !wrappedVaultKey) {
    throw new Error("Session data is missing, please log out and log in again");
  }

  validateKdf(kdf);
  const saltBytes = fromBase64(salt);
  if (saltBytes.length < 16) throw new Error("Stored salt is invalid");

  const { encKey } = await vaultCrypto.deriveKeys(
    password,
    saltBytes,
    kdf.time,
    kdf.memoryKiB,
    kdf.parallelism,
  );

  let vaultKey: Uint8Array;
  try {
    vaultKey = await vaultCrypto.unwrapVaultKey(
      encKey,
      fromBase64(wrappedVaultKey),
    );
  } catch {
    throw new Error("Incorrect password");
  }

  useSession.getState().unlock(vaultKey);
}
