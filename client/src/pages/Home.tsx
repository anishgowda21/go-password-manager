import { Link, Navigate } from "react-router";
import { homeFor, useAuthState } from "../store/session";

export default function Home() {
  const auth = useAuthState();
  if (auth !== "out") return <Navigate to={homeFor(auth)} replace />;

  return (
    <main>
      <h1>Password Manager</h1>
      <Link to="/signup">Sign up</Link> <Link to="/login">Log in</Link>
    </main>
  );
}
