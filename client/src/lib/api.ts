import type { SignupRequest } from "./types";

export function signup(body: SignupRequest): Promise<Response> {
  return fetch("/api/signup", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}
