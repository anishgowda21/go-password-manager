import { useState } from "react";
import type { SubmitEvent } from "react";
import { Navigate } from "react-router";
import { useAuthState, useSession } from "../store/session";
import { unlockVault } from "../lib/unlock";
import { logOut } from "../lib/auth";

const input =
  "w-full rounded-md border border-gray-300 px-3 py-2 text-sm outline-none transition-colors focus:border-gray-900";
const btnPrimary =
  "w-full rounded-md border border-gray-900 bg-gray-900 px-4 py-2 text-center text-sm font-medium text-white transition-colors hover:bg-black disabled:cursor-not-allowed disabled:opacity-60";

export default function Unlock() {
  const auth = useAuthState();
  const email = useSession((s) => s.email);

  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  if (auth === "unlocked") return <Navigate to="/dashboard" replace />;
  if (auth === "out") return <Navigate to="/login" replace />;

  async function handleSubmit(e: SubmitEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!password) {
      setError("Enter your master password");
      return;
    }

    setLoading(true);
    setError("");
    try {
      await unlockVault(password);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not unlock");
      setLoading(false);
    }
  }

  return (
    <main className="grid min-h-screen place-items-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="mb-1 text-center text-2xl font-semibold tracking-tight">
          Vault locked
        </h1>
        <p className="mb-6 text-center text-sm text-gray-500">{email}</p>

        <form onSubmit={handleSubmit} className="space-y-3">
          <input
            type="password"
            placeholder="Master password"
            autoComplete="current-password"
            autoFocus
            className={input}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <button type="submit" disabled={loading} className={btnPrimary}>
            {loading ? "Unlocking..." : "Unlock"}
          </button>
        </form>

        {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

        <p className="mt-6 text-center text-sm text-gray-500">
          <button
            type="button"
            onClick={() => void logOut()}
            className="underline hover:text-gray-900"
          >
            Log out
          </button>
        </p>
      </div>
    </main>
  );
}
