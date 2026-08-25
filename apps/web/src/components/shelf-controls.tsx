"use client";

import { useState } from "react";
import { useAuth, SignInButton } from "@clerk/nextjs";
import { apiRequest } from "@/lib/client";
import { cn } from "@/lib/format";

const STATUSES = [
  { value: "owned", label: "I own it" },
  { value: "played", label: "Played it" },
  { value: "wishlist", label: "Want it" },
] as const;

/** Shelf membership: three toggles, saved the moment they are pressed. */
export function ShelfControls({ slug, initial }: { slug: string; initial: string[] }) {
  const { isSignedIn, getToken } = useAuth();
  const [active, setActive] = useState<string[]>(initial);
  const [error, setError] = useState<string | null>(null);

  const toggle = async (status: string) => {
    const on = active.includes(status);
    const previous = active;
    setActive(on ? active.filter((s) => s !== status) : [...active, status]);
    setError(null);

    try {
      const token = await getToken();
      await apiRequest(`/shelf/${slug}${on ? `?status=${status}` : ""}`, {
        method: on ? "DELETE" : "PUT",
        body: on ? undefined : JSON.stringify({ status }),
        token,
      });
    } catch (err) {
      setActive(previous);
      setError(err instanceof Error ? err.message : "Could not update your shelf");
    }
  };

  if (!isSignedIn) {
    return (
      <SignInButton mode="modal">
        <button className="font-mono text-[11px] uppercase tracking-[0.18em] text-chalk-faint hover:text-chalk transition-colors">
          Sign in to add this to your shelf
        </button>
      </SignInButton>
    );
  }

  return (
    <div>
      <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-chalk-faint">
        Your shelf
      </p>
      <div className="mt-2.5 flex gap-2 flex-wrap">
        {STATUSES.map((s) => {
          const on = active.includes(s.value);
          return (
            <button
              key={s.value}
              type="button"
              onClick={() => void toggle(s.value)}
              aria-pressed={on}
              className={cn(
                "px-3 py-1.5 rounded-full text-xs border transition-colors",
                on
                  ? "bg-meeple-teal text-felt-950 border-meeple-teal font-medium"
                  : "border-rule-soft text-chalk-dim hover:border-rule hover:text-chalk",
              )}
            >
              {s.label}
            </button>
          );
        })}
      </div>
      {error && <p className="mt-2 text-xs text-meeple-red">{error}</p>}
    </div>
  );
}
