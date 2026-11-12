import type { LoginRequest, SignupRequest } from "./types";

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export class RateLimitError extends Error {
  retryAfter: number;
  constructor(retryAfter: number) {
    super("Too many attempts, please wait a moment.");
    this.retryAfter = retryAfter;
  }
}

export function throwIfRateLimited(res: Response) {
  if (res.status !== 429) return;
  const secs = Number(res.headers.get("Retry-After"));
  throw new RateLimitError(secs > 0 ? secs : 10); // 10s if header missing
}

export function signup(body: SignupRequest): Promise<Response> {
  return fetch("/api/signup", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export function prelogin(email: string): Promise<Response> {
  return fetch("/api/prelogin", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  });
}

export function login(body: LoginRequest): Promise<Response> {
  return fetch("/api/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export async function errorMessage(
  res: Response,
  fallback: string,
): Promise<string> {
  if (res.status === 429) {
    const secs = Number(res.headers.get("Retry-After"));
    return secs > 0
      ? `Too many attempts. Try again in ${secs}s.`
      : "Too many attempts. Try again shortly.";
  }
  try {
    const data = await res.json();
    if (typeof data?.error === "string") return data.error;
  } catch {
    // something else.
  }
  return fallback;
}
