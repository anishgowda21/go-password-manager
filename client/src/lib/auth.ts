import { useSession } from "../store/session";
import { ApiError, errorMessage } from "./api";
import type { RefreshResponse } from "./types";

let inflight: Promise<string> | null = null;

async function doRefresh(): Promise<string> {
  const sent = useSession.getState().refreshToken;
  if (!sent) throw new ApiError(401, "Not logged in");

  const res = await fetch("/api/refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refreshToken: sent }),
  });

  if (res.status === 401) {
    if (useSession.getState().refreshToken === sent)
      useSession.getState().clear();
    throw new ApiError(401, "Session expired, please log in again");
  }
  if (!res.ok)
    throw new ApiError(
      res.status,
      await errorMessage(res, "Could not refresh session"),
    );

  const data = (await res.json()) as RefreshResponse;
  if (useSession.getState().refreshToken !== sent)
    throw new ApiError(401, "Logged out");

  useSession.getState().setTokens(data.accessToken, data.refreshToken);
  return data.accessToken;
}

function refreshTokens(): Promise<string> {
  inflight ??= doRefresh().finally(() => {
    inflight = null;
  });
  return inflight;
}

export async function authFetch(
  path: string,
  method: string,
  body?: unknown,
): Promise<Response> {
  const send = (token: string) =>
    fetch(path, {
      method,
      headers: {
        Authorization: `Bearer ${token}`,
        ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });

  let used = useSession.getState().accessToken;
  if (!used) used = await refreshTokens();
  const res = await send(used);
  if (res.status !== 401) return res;

  // another request may already have refreshed while this one was in flight
  const current = useSession.getState().accessToken;
  const token =
    current && current !== used
      ? current
      : await refreshTokens().catch(() => null);
  if (!token) return res;

  return send(token);
}

export async function logOut(): Promise<void> {
  const { refreshToken } = useSession.getState();
  useSession.getState().clear();

  if (!refreshToken) return;
  try {
    await fetch("/api/logout", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refreshToken }),
    });
  } catch {
    // offline
  }
}
