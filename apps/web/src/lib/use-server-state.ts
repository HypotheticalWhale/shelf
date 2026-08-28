"use client";

import { useState, type Dispatch, type SetStateAction } from "react";

/**
 * Client state seeded from server data, which re-seeds when that data changes.
 *
 * useState reads its argument only on the first render. A component that seeds
 * itself from a server prop therefore keeps showing the first version it ever
 * saw: router.refresh() re-renders the server component and hands down fresh
 * data, and the component ignores it. That is what left the "on other shelves"
 * counts showing the old totals right next to the buttons that had just
 * changed them.
 *
 * Local updates still apply immediately — an optimistic rating, a toggled
 * shelf — but whenever the server sends a new object it wins, because it is
 * the authority.
 */
export function useServerState<T>(fromServer: T): [T, Dispatch<SetStateAction<T>>] {
  const [value, setValue] = useState<T>(fromServer);
  const [seen, setSeen] = useState<T>(fromServer);

  // Adjusting state during render, rather than in an effect: React re-runs the
  // component immediately with the new value instead of painting the stale one
  // first and correcting it a frame later.
  if (seen !== fromServer) {
    setSeen(fromServer);
    setValue(fromServer);
  }

  return [value, setValue];
}
