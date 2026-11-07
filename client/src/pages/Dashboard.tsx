import { useCallback, useEffect, useState } from "react";
import { Link, Navigate } from "react-router";
import { useSession } from "../store/session";
import { logOut } from "../lib/auth";
import { createItem, deleteItem, fetchItems, updateItem } from "../lib/items";
import ItemDialog from "../components/ItemDialog";
import type { ItemData, VaultItem } from "../lib/types";

export default function Dashboard() {
  const email = useSession((s) => s.email);
  const vaultKey = useSession((s) => s.vaultKey);
  const items = useSession((s) => s.items);
  const refreshToken = useSession((s) => s.refreshToken);

  const [dialog, setDialog] = useState<{ item: VaultItem | null } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [failed, setFailed] = useState(0);

  const sync = useCallback(async () => {
    if (!vaultKey) return;
    try {
      const result = await fetchItems(vaultKey);
      useSession.getState().setItems(result.items);
      setFailed(result.failed);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Sync failed");
    } finally {
      setLoading(false);
    }
  }, [vaultKey]);

  useEffect(() => {
    void sync();
  }, [sync]);

  if (!refreshToken) return <Navigate to="/login" replace />;
  if (!vaultKey) return <Navigate to="/unlock" replace />;

  async function handleSave(data: ItemData) {
    const current = dialog?.item ?? null;
    const saved = current
      ? await updateItem(vaultKey!, current, data)
      : await createItem(vaultKey!, data);
    useSession.getState().upsertItem(saved);
    setDialog(null);
  }

  async function handleDelete(item: VaultItem) {
    await deleteItem(item.id);
    useSession.getState().removeItem(item.id);
    setDialog(null);
  }

  const sorted = [...items].sort((a, b) =>
    a.data.name.localeCompare(b.data.name),
  );

  return (
    <main className="mx-auto max-w-2xl p-4">
      <header className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-semibold">Vault</h1>
          <Link to="/">Home</Link>
        </div>
        <div className="flex items-center gap-3 text-sm">
          <span>{email}</span>
          <button
            onClick={() => {
              setLoading(true);
              void sync();
            }}
          >
            Sync
          </button>
          <button onClick={() => void logOut()}>Log out</button>
          <button onClick={() => useSession.getState().lock()}>Lock</button>
        </div>
      </header>

      {error && <p className="text-red-600">{error}</p>}
      {failed > 0 && (
        <p className="text-amber-700">
          {failed} item(s) could not be decrypted.
        </p>
      )}

      {loading ? (
        <p>Loading...</p>
      ) : sorted.length === 0 ? (
        <p>No items yet.</p>
      ) : (
        <ul className="divide-y rounded border">
          {sorted.map((item) => (
            <li key={item.id}>
              <button
                className="w-full px-3 py-2 text-left hover:bg-gray-100"
                onClick={() => setDialog({ item })}
              >
                <div className="font-medium">
                  {item.data.name || "(no name)"}
                </div>
                <div className="text-sm text-gray-600">
                  {item.data.username}
                </div>
              </button>
            </li>
          ))}
        </ul>
      )}

      <button
        aria-label="Add item"
        className="fixed bottom-6 right-6 h-12 w-12 rounded-full bg-black text-2xl text-white"
        onClick={() => setDialog({ item: null })}
      >
        +
      </button>

      {dialog && (
        <ItemDialog
          key={dialog.item?.id ?? "new"}
          item={dialog.item}
          onClose={() => setDialog(null)}
          onSave={handleSave}
          onDelete={handleDelete}
        />
      )}
    </main>
  );
}
