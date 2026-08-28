import { auth } from "@clerk/nextjs/server";

/**
 * Server-side API access.
 *
 * Server components call the Go service directly rather than looping back
 * through this app's own rewrite — one less hop, and it works during build and
 * prerender where a relative URL has no host to resolve against.
 */
const API_URL = process.env.SHELF_API_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

type FetchOptions = {
  /** Attach the caller's Clerk token so the API can personalise the response. */
  authenticated?: boolean;
  /**
   * Seconds to cache. Omit for always-fresh.
   *
   * This has to be opt-in. When the default was a silent 60 seconds, anything
   * that reflects what people are doing right now went stale: a game added to
   * your shelf was missing from your own shelf page, and your change was
   * missing from everyone else's view of it, until the minute happened to
   * elapse. Pages that genuinely want caching now say so.
   */
  revalidate?: number;
};

export async function apiGet<T>(path: string, options: FetchOptions = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: "application/json" };

  if (options.authenticated) {
    const { getToken } = await auth();
    const token = await getToken();
    if (token) headers.Authorization = `Bearer ${token}`;
  }

  // A personalised response must never be cached and served to someone else,
  // and an unauthenticated read is only cached when the caller asks for it.
  const cached = !options.authenticated && options.revalidate !== undefined;

  const res = await fetch(`${API_URL}${path}`, {
    headers,
    cache: cached ? undefined : "no-store",
    next: cached ? { revalidate: options.revalidate } : undefined,
  });

  if (!res.ok) {
    throw new ApiError(res.status, `${path} returned ${res.status}`);
  }
  return res.json() as Promise<T>;
}

/** Returns null on 404 instead of throwing, for pages that render notFound(). */
export async function apiGetOrNull<T>(path: string, options: FetchOptions = {}): Promise<T | null> {
  try {
    return await apiGet<T>(path, options);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null;
    throw err;
  }
}
