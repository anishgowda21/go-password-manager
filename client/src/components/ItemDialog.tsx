import { useState } from "react";
import type { SubmitEvent } from "react";
import type { ItemData, VaultItem } from "../lib/types";
import Button from "./Button";
import Field from "./Field";

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

const textFields = [
  { key: "name", label: "Name" },
  { key: "url", label: "URL" },
  { key: "username", label: "Username", autoComplete: "off" },
] as const;

export default function ItemDialog({ item, onClose, onSave, onDelete }: Props) {
  const [data, setData] = useState<ItemData>(item?.data ?? blank);
  const [editing, setEditing] = useState(item === null);
  const [showPassword, setShowPassword] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const set = (key: keyof ItemData) => (value: string) =>
    setData((d) => ({ ...d, [key]: value }));

  const original = item?.data ?? blank;
  const dirty = (Object.keys(original) as (keyof ItemData)[]).some(
    (key) => data[key] !== original[key],
  );

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
    if (!editing || !dirty) return;
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

  function handleCancelEdit() {
    if (!item) return onClose();
    setData(item.data);
    setError("");
    setShowPassword(false);
    setEditing(false);
  }

  return (
    <div className="fixed inset-0 z-10 grid place-items-center bg-black/40 p-4">
      <form
        role="dialog"
        aria-modal="true"
        className="w-full max-w-md space-y-4 rounded-lg border border-gray-200 bg-white p-6 shadow-lg"
        onSubmit={handleSubmit}
      >
        <h2 className="text-xl font-semibold tracking-tight">
          {!item ? "New item" : editing ? "Edit item" : "Item details"}
        </h2>

        {textFields.map(({ key, label, ...rest }) => (
          <Field
            key={key}
            label={label}
            value={data[key]}
            onChange={set(key)}
            readOnly={!editing}
            {...rest}
          />
        ))}

        <Field
          label="Password"
          type={showPassword ? "text" : "password"}
          autoComplete="new-password"
          value={data.password}
          onChange={set("password")}
          readOnly={!editing}
          action={
            <Button onClick={() => setShowPassword((v) => !v)}>
              {showPassword ? "Hide" : "Show"}
            </Button>
          }
        />

        <Field
          multiline
          label="Notes"
          value={data.notes}
          onChange={set("notes")}
          readOnly={!editing}
        />

        {error && <p className="text-sm text-red-600">{error}</p>}

        <div className="flex items-center justify-between pt-2">
          <div>
            {item && editing && (
              <Button variant="danger" disabled={busy} onClick={handleDelete}>
                Delete
              </Button>
            )}
          </div>
          <div className="flex gap-2">
            {editing ? (
              <>
                <Button key="cancel" disabled={busy} onClick={handleCancelEdit}>
                  Cancel
                </Button>
                <Button
                  key="save"
                  type="submit"
                  variant="primary"
                  disabled={busy || !dirty}
                >
                  {busy ? "Saving..." : "Save"}
                </Button>
              </>
            ) : (
              <>
                <Button key="close" onClick={onClose}>
                  Close
                </Button>
                <Button
                  key="edit"
                  variant="primary"
                  onClick={() => setEditing(true)}
                >
                  Edit
                </Button>
              </>
            )}
          </div>
        </div>
      </form>
    </div>
  );
}
