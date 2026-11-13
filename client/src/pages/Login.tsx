import { useState } from "react";
import type { SubmitEvent } from "react";
import { Link, Navigate, useNavigate } from "react-router";
import { logIn } from "../lib/login";
import { homeFor, startSession, useAuthState } from "../store/session";
import { RateLimitError } from "../lib/api";
import { formatSeconds, useCooldown } from "../lib/useCooldown";
import DemoBanner from "../components/DemoBanner";

const input =
  "w-full rounded-md border border-gray-300 px-3 py-2 text-sm outline-none transition-colors focus:border-gray-900";
const btnPrimary =
  "w-full rounded-md border border-gray-900 bg-gray-900 px-4 py-2 text-center text-sm font-medium text-white transition-colors hover:bg-black disabled:cursor-not-allowed disabled:opacity-60";

export default function Login() {
  const auth = useAuthState();
  const { secondsLeft, start } = useCooldown();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const navigate = useNavigate();

  if (auth !== "out") return <Navigate to={homeFor(auth)} replace />;

  async function handleSubmit(e: SubmitEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!email || !password) {
      setError("Email and password are required");
      return;
    }

    setLoading(true);
    setError("");

    try {
      const { auth, vaultKey } = await logIn(email, password);
      startSession(auth, vaultKey, email);
      navigate("/dashboard");
    } catch (err) {
      if (err instanceof RateLimitError) start(err.retryAfter);
      setError(err instanceof Error ? err.message : "Login failed, try again");
      setPassword("");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="grid min-h-screen place-items-center px-4">
      <div className="w-full max-w-sm">
        <DemoBanner />
        <h1 className="mb-6 text-center text-2xl font-semibold tracking-tight">
          Log in
        </h1>

        <form onSubmit={handleSubmit} className="space-y-3">
          <input
            type="email"
            placeholder="Email"
            autoComplete="email"
            className={input}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <input
            type="password"
            placeholder="Master password"
            autoComplete="current-password"
            className={input}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <button
            type="submit"
            disabled={loading || secondsLeft > 0}
            className={btnPrimary}
          >
            {secondsLeft > 0
              ? `Try again in ${formatSeconds(secondsLeft)}`
              : loading
                ? "Logging in..."
                : "Log in"}
          </button>
        </form>

        {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

        <p className="mt-6 text-center text-sm text-gray-500">
          <Link to="/signup" className="underline hover:text-gray-900">
            Create an account
          </Link>
        </p>
      </div>
    </main>
  );
}
