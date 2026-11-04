import { Link } from "react-router";

export default function Home() {
  return (
    <main>
      <h1>Password Manager</h1>
      <Link to="/signup">Sign up</Link>
    </main>
  );
}
