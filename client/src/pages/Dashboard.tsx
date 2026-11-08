import { useCallback, useEffect, useState } from "react";
import { Link, Navigate } from "react-router";
import { useSession } from "../store/session";
import { logOut } from "../lib/auth";
import { createItem, deleteItem, fetchItems, updateItem } from "../lib/items";
import ItemDialog from "../components/ItemDialog";
import ItemCard from "../components/ItemCard";
import type { ItemData, VaultItem } from "../lib/types";

const btn =
  "rounded-md border border-gray-300 px-3 py-1.5 text-sm transition-colors hover:bg-gray-100";

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
    <main className="mx-auto max-w-2xl px-4 py-8">
      <header className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-baseline gap-3">
          <h1 className="text-2xl font-semibold tracking-tight">Vault</h1>
          <Link to="/" className="text-sm text-gray-500 hover:text-gray-900">
            Home
          </Link>
        </div>
        <div className="flex items-center gap-2">
          <span className="mr-1 hidden text-sm text-gray-500 sm:inline">
            {email}
          </span>
          <button
            className={btn}
            onClick={() => {
              setLoading(true);
              void sync();
            }}
          >
            Sync
          </button>
          <button className={btn} onClick={() => useSession.getState().lock()}>
            Lock
          </button>
          <button className={btn} onClick={() => void logOut()}>
            Log out
          </button>
        </div>
      </header>

      {error && <p className="mb-3 text-sm text-red-600">{error}</p>}
      {failed > 0 && (
        <p className="mb-3 text-sm text-amber-700">
          {failed} item(s) could not be decrypted.
        </p>
      )}

      {loading ? (
        <p className="py-12 text-center text-sm text-gray-500">Loading...</p>
      ) : sorted.length === 0 ? (
        <p className="py-12 text-center text-sm text-gray-500">
          No items yet. Add your first one with the + button.
        </p>
      ) : (
        <ul className="space-y-2">
          {sorted.map((item) => (
            <li key={item.id}>
              <ItemCard item={item} onClick={() => setDialog({ item })} />
            </li>
          ))}
        </ul>
      )}

      <button
        aria-label="Add item"
        className="fixed bottom-6 right-6 grid h-12 w-12 place-items-center rounded-full bg-gray-900 text-2xl text-white shadow-md transition-colors hover:bg-black"
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
