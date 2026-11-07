import { create } from "zustand";
import type { AuthResponse, VaultItem } from "../lib/types";

const EMAIL_KEY = "email";

export function getRememberedEmail(): string {
  try {
    return sessionStorage.getItem(EMAIL_KEY) ?? "";
  } catch {
    return "";
  }
}

function rememberEmail(email: string) {
  try {
    sessionStorage.setItem(EMAIL_KEY, email);
  } catch {
    // throws error if storage is blocked so ignore it
  }
}

interface SessionState {
  email: string;
  userId: string;
  accessToken: string;
  refreshToken: string;
  vaultKey: Uint8Array | null;
  items: VaultItem[];
  setItems: (items: VaultItem[]) => void;
  upsertItem: (item: VaultItem) => void;
  removeItem: (id: string) => void;
  setTokens: (accessToken: string, refreshToken: string) => void;
  clear: () => void;
}

const empty = {
  email: "",
  userId: "",
  accessToken: "",
  refreshToken: "",
  vaultKey: null,
  items: [] as VaultItem[],
};

export const useSession = create<SessionState>()((set) => ({
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

  setTokens: (accessToken, refreshToken) => set({ accessToken, refreshToken }),

  clear: () => set(empty),
}));

export function startSession(
  auth: AuthResponse,
  vaultKey: Uint8Array,
  email: string,
) {
  const normalized = email.trim().toLocaleLowerCase();
  rememberEmail(email);
  useSession.setState({
    email: normalized,
    userId: auth.userId,
    accessToken: auth.accessToken,
    refreshToken: auth.refreshToken,
    vaultKey,
    items: [],
  });
}
