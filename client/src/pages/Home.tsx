import { Link } from "react-router";
import { SiGo, SiReact } from "react-icons/si";
import { homeFor, useAuthState } from "../store/session";

const btn =
  "w-32 rounded-md border border-gray-300 px-4 py-2 text-center text-sm font-medium transition-colors hover:bg-gray-100";
const btnPrimary =
  "w-32 rounded-md border border-gray-900 bg-gray-900 px-4 py-2 text-center text-sm font-medium text-white transition-colors hover:bg-black";
const pill =
  "inline-flex items-center gap-1.5 rounded-full border border-gray-200 px-3 py-1 text-sm text-gray-600";

export default function Home() {
  const auth = useAuthState();
  const signedOut = auth === "out";

  return (
    <main className="grid min-h-screen place-items-center px-4">
      <div className="w-full max-w-md text-center">
        <h1 className="mb-2 text-3xl font-semibold tracking-tight">
          Go Password Manager
        </h1>
        <p className="mb-5 text-gray-500">A zero trust password manager.</p>

        <div className="mb-8 flex items-center justify-center gap-2 text-gray-400">
          <span className={pill}>
            <SiGo className="text-xl text-[#00ADD8]" aria-hidden />
            Go
          </span>
          <span aria-hidden>+</span>
          <span className={pill}>
            <SiReact className="text-base text-[#61DAFB]" aria-hidden />
            React
          </span>
        </div>

        <div className="flex justify-center gap-3">
          {signedOut ? (
            <>
              <Link to="/signup" className={btnPrimary}>
                Sign up
              </Link>
              <Link to="/login" className={btn}>
                Log in
              </Link>
            </>
          ) : (
            <Link to={homeFor(auth)} className={btnPrimary}>
              Continue
            </Link>
          )}
        </div>
      </div>
    </main>
  );
}
