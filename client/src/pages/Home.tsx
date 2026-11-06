import { Link } from "react-router";

export default function Home() {
  return (
    <main>
      <h1>Password Manager</h1>
      <Link to="/login">Login</Link> <br />
      <Link to="/signup">Sign up</Link>
    </main>
  );
}
