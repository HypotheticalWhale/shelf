"use client";

/**
 * Browser-side API access.
 *
 * Requests go to this app's own /api prefix, which Next rewrites to the Go
 * service. Same origin, so no preflight.
 */
export async function apiRequest<T>(
  path: string,
  init: RequestInit & { token?: string | null } = {},
): Promise<T> {
  const { token, ...rest } = init;

  const headers = new Headers(rest.headers);
  headers.set("Accept", "application/json");
  if (rest.body) headers.set("Content-Type", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);

  const res = await fetch(`/api${path}`, { ...rest, headers });

  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch {
      // Response had no JSON body; the status-based message stands.
    }
    throw new Error(message);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}
