import { useState } from "react";
import type { SubmitEvent } from "react";
import { Link, Navigate, useNavigate } from "react-router";
import { logIn } from "../lib/login";
import { homeFor, startSession, useAuthState } from "../store/session";

export default function Login() {
  const auth = useAuthState();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [userId, setUserId] = useState("");
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
      setUserId(auth.userId);
      setPassword("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed, try again");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main>
      <h1>Log in</h1>
      <form onSubmit={handleSubmit}>
        <input
          type="email"
          placeholder="Email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <input
          type="password"
          placeholder="Master password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <button type="submit" disabled={loading}>
          {loading ? "Logging in..." : "Log in"}
        </button>
      </form>
      {error && <p style={{ color: "red" }}>{error}</p>}
      {userId && <p>Logged in as {userId}</p>}
      <Link to="/signup">Create an account</Link>
    </main>
  );
}
