import { useState } from "react";
import type { FormEvent } from "react";
import { Navigate } from "react-router";
import { useAuthState, useSession } from "../store/session";
import { unlockVault } from "../lib/unlock";
import { logOut } from "../lib/auth";

export default function Unlock() {
  const auth = useAuthState();
  const email = useSession((s) => s.email);

  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  if (auth === "unlocked") return <Navigate to="/dashboard" replace />;
  if (auth === "out") return <Navigate to="/login" replace />;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!password) {
      setError("Enter your master password");
      return;
    }

    setLoading(true);
    setError("");
    try {
      await unlockVault(password);
      setPassword("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not unlock");
      setLoading(false);
    }
  }

  return (
    <main className="mx-auto max-w-sm p-4">
      <h1 className="text-xl font-semibold">Vault locked</h1>
      <p className="mb-4 text-gray-600">{email}</p>

      <form onSubmit={handleSubmit} className="space-y-3">
        <input
          type="password"
          placeholder="Master password"
          autoFocus
          className="w-full rounded border p-2"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <div className="flex items-center justify-between">
          <button type="submit" disabled={loading}>
            {loading ? "Unlocking..." : "Unlock"}
          </button>
          <button type="button" onClick={() => void logOut()}>
            Log out
          </button>
        </div>
      </form>

      {error && <p className="mt-3 text-red-600">{error}</p>}
    </main>
  );
}
