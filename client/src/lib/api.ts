import type { LoginRequest, SignupRequest } from "./types";

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export function authFetch(
  path: string,
  token: string,
  method: string,
  body?: unknown,
): Promise<Response> {
  return fetch(path, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
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
  try {
    const data = await res.json();
    if (typeof data?.error === "string") return data.error;
  } catch {
    // something else.
  }
  return fallback;
}
