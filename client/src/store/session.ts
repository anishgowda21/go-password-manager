import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { AuthResponse, KdfPayload, VaultItem } from "../lib/types";

interface SessionState {
  email: string;
  userId: string;
  accessToken: string;
  refreshToken: string;
  wrappedVaultKey: string;
  salt: string;
  kdf: KdfPayload | null;
  vaultKey: Uint8Array | null;
  items: VaultItem[];
  setItems: (items: VaultItem[]) => void;
  upsertItem: (item: VaultItem) => void;
  removeItem: (id: string) => void;
  setTokens: (accessToken: string, refreshToken: string) => void;
  unlock: (vaultKey: Uint8Array) => void;
  lock: () => void;
  clear: () => void;
}

const empty = {
  email: "",
  userId: "",
  accessToken: "",
  refreshToken: "",
  wrappedVaultKey: "",
  salt: "",
  kdf: null as KdfPayload | null,
  vaultKey: null as Uint8Array | null,
  items: [] as VaultItem[],
};

export const useSession = create<SessionState>()(
  persist(
    (set) => ({
      ...empty,
      setItems: (items) => set({ items }),
      upsertItem: (item) =>
        set((s) => ({
          items: s.items.some((i) => i.id === item.id)
            ? s.items.map((i) => (i.id === item.id ? item : i))
            : [...s.items, item],
        })),
      removeItem: (id) =>
        set((s) => ({ items: s.items.filter((i) => i.id !== id) })),
      setTokens: (accessToken, refreshToken) =>
        set({ accessToken, refreshToken }),
      unlock: (vaultKey) => set({ vaultKey }),
      lock: () => set({ accessToken: "", vaultKey: null, items: [] }),
      clear: () => set(empty),
    }),
    {
      name: "session",
      storage: createJSONStorage(() => sessionStorage),
      partialize: (s) => ({
        email: s.email,
        userId: s.userId,
        refreshToken: s.refreshToken,
        wrappedVaultKey: s.wrappedVaultKey,
        salt: s.salt,
        kdf: s.kdf,
      }),
    },
  ),
);

export function startSession(
  auth: AuthResponse,
  vaultKey: Uint8Array,
  email: string,
) {
  useSession.setState({
    email: email.trim().toLowerCase(),
    userId: auth.userId,
    accessToken: auth.accessToken,
    refreshToken: auth.refreshToken,
    wrappedVaultKey: auth.wrappedVaultKey,
    salt: auth.salt,
    kdf: auth.kdf,
    vaultKey,
    items: [],
  });
}

export type AuthState = "out" | "locked" | "unlocked";

export function useAuthState(): AuthState {
  return useSession((s) =>
    s.vaultKey ? "unlocked" : s.refreshToken ? "locked" : "out",
  );
}

export function homeFor(state: AuthState): string {
  return state === "unlocked"
    ? "/dashboard"
    : state === "locked"
      ? "/unlock"
      : "/login";
}
