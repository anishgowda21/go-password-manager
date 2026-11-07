import { useState } from "react";
import type { ChangeEvent, SubmitEvent } from "react";
import type { ItemData, VaultItem } from "../lib/types";

interface Props {
  item: VaultItem | null;
  onClose: () => void;
  onSave: (data: ItemData) => Promise<void>;
  onDelete: (item: VaultItem) => Promise<void>;
}

const blank: ItemData = {
  name: "",
  url: "",
  username: "",
  password: "",
  notes: "",
};

export default function ItemDialog({ item, onClose, onSave, onDelete }: Props) {
  const [data, setData] = useState<ItemData>(item?.data ?? blank);
  const [showPassword, setShowPassword] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  function field(key: keyof ItemData) {
    return {
      value: data[key],
      onChange: (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
        setData((d) => ({ ...d, [key]: e.target.value })),
    };
  }

  async function run(action: () => Promise<void>) {
    setBusy(true);
    setError("");
    try {
      await action();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong");
      setBusy(false);
    }
  }

  function handleSubmit(e: SubmitEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!data.name.trim()) {
      setError("Name is required");
      return;
    }
    void run(() => onSave({ ...data, name: data.name.trim() }));
  }

  function handleDelete() {
    if (!item || !window.confirm("Delete this item?")) return;
    void run(() => onDelete(item));
  }

  const input = "mt-1 w-full rounded border p-2";

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black/40 p-4">
      <form
        role="dialog"
        aria-modal="true"
        className="w-full max-w-md space-y-3 rounded bg-white p-4"
        onSubmit={handleSubmit}
      >
        <h2 className="text-lg font-semibold">
          {item ? "Edit item" : "New item"}
        </h2>

        <label className="block text-sm">
          Name
          <input className={input} {...field("name")} />
        </label>
        <label className="block text-sm">
          URL
          <input className={input} {...field("url")} />
        </label>
        <label className="block text-sm">
          Username
          <input className={input} autoComplete="off" {...field("username")} />
        </label>
        <label className="block text-sm">
          Password
          <div className="flex gap-2">
            <input
              className={input}
              type={showPassword ? "text" : "password"}
              autoComplete="new-password"
              {...field("password")}
            />
            <button type="button" onClick={() => setShowPassword((v) => !v)}>
              {showPassword ? "Hide" : "Show"}
            </button>
          </div>
        </label>
        <label className="block text-sm">
          Notes
          <textarea className={input} rows={3} {...field("notes")} />
        </label>

        {error && <p className="text-sm text-red-600">{error}</p>}

        <div className="flex items-center justify-between">
          <div>
            {item && (
              <button
                type="button"
                disabled={busy}
                className="text-red-600"
                onClick={handleDelete}
              >
                Delete
              </button>
            )}
          </div>
          <div className="flex gap-2">
            <button type="button" disabled={busy} onClick={onClose}>
              Cancel
            </button>
            <button type="submit" disabled={busy}>
              {busy ? "Saving..." : "Save"}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}
