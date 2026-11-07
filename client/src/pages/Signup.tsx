import { useState, type SubmitEvent } from "react";
import { signUp } from "../lib/signup";
import { useNavigate } from "react-router";
import { startSession } from "../store/session";

export default function Signup() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const navigate = useNavigate();

  async function handleSubmit(e: SubmitEvent<HTMLFormElement>) {
    e.preventDefault();

    if (!email || !password) {
      setError("Email and password are required");
      return;
    }
    if (password !== confirm) {
      setError("Passwords do not match");
      return;
    }

    setLoading(true);
    setError("");

    try {
      const { auth, vaultKey } = await signUp(email, password);
      startSession(auth, vaultKey, email);
      navigate("/dashboard");
      setPassword("");
      setConfirm("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Signup failed, try again");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main>
      <h1>Sign up</h1>
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
        <input
          type="password"
          placeholder="Confirm master password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
        />
        <button type="submit" disabled={loading}>
          {loading ? "Creating account..." : "Sign up"}
        </button>
      </form>
      {error && <p style={{ color: "red" }}>{error}</p>}
    </main>
  );
}
