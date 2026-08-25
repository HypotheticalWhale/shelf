"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAuth } from "@clerk/nextjs";
import { SignInButton } from "@clerk/nextjs";
import { motion } from "motion/react";
import { apiRequest } from "@/lib/client";
import { scoreColor, cn } from "@/lib/format";
import type { Game } from "@/lib/types";

type Props = {
  slug: string;
  viewerRating: number | null;
  size?: "sm" | "lg";
  /** Called with the game the API returns, so the score can update in place. */
  onRated?: (game: Game) => void;
  className?: string;
};

const VALUES = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];

/**
 * Rating a game, in one tap.
 *
 * This is the product's whole argument. It appears on cards in the browse grid,
 * not only on game pages, so rating never costs a navigation. Clicking is
 * optimistic — the pips and the score move immediately and reconcile against
 * the API's response, which returns the recomputed aggregate in the same round
 * trip. If the write fails the rating snaps back and says why.
 */
export function RatingPips({ slug, viewerRating, size = "sm", onRated, className }: Props) {
  const { isSignedIn, getToken } = useAuth();
  const [rating, setRating] = useState<number | null>(viewerRating);
  const [preview, setPreview] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => setRating(viewerRating), [viewerRating]);

  const commit = useCallback(
    async (value: number | null) => {
      const previous = rating;
      setRating(value);
      // Drop the hover preview as soon as a value is committed. Without this
      // the strip keeps showing whichever pip the pointer last passed over,
      // so the control can display a number the user never chose.
      setPreview(null);
      setError(null);
      setBusy(true);

      try {
        const token = await getToken();
        const game = await apiRequest<Game>(`/games/${slug}/rating`, {
          method: value === null ? "DELETE" : "PUT",
          body: value === null ? undefined : JSON.stringify({ value }),
          token,
        });
        onRated?.(game);
      } catch (err) {
        setRating(previous);
        setError(err instanceof Error ? err.message : "Could not save that rating");
      } finally {
        setBusy(false);
      }
    },
    [getToken, onRated, rating, slug],
  );

  const onKeyDown = (event: React.KeyboardEvent) => {
    // Whole numbers on the number row, 0 for ten — faster than aiming at a pip.
    if (event.key >= "1" && event.key <= "9") {
      event.preventDefault();
      void commit(Number(event.key));
    } else if (event.key === "0") {
      event.preventDefault();
      void commit(10);
    } else if (event.key === "Backspace" || event.key === "Delete") {
      event.preventDefault();
      if (rating !== null) void commit(null);
    }
  };

  const shown = preview ?? rating;
  const tall = size === "lg";

  const pips = (
    <div
      ref={containerRef}
      role="radiogroup"
      aria-label="Your rating out of 10"
      tabIndex={0}
      onKeyDown={onKeyDown}
      onMouseLeave={() => setPreview(null)}
      onBlur={(event) => {
        // Focus moving out of the strip should also clear the preview, or
        // keyboard users are left looking at a stale value.
        if (!event.currentTarget.contains(event.relatedTarget as Node)) {
          setPreview(null);
        }
      }}
      className={cn(
        "flex items-end rounded-md",
        busy && "opacity-70",
      )}
    >
      {VALUES.map((value) => {
        const filled = shown !== null && value <= shown;
        const isChosen = rating === value;
        return (
          <button
            key={value}
            type="button"
            role="radio"
            aria-checked={isChosen}
            aria-label={`${value} out of 10`}
            disabled={busy}
            onMouseEnter={() => setPreview(value)}
            onFocus={() => setPreview(value)}
            onClick={() => void commit(isChosen ? null : value)}
            // The visible pip is deliberately slim, but a 10px target is far
            // too small to hit: a near miss lands on the card behind it and
            // opens the game instead of rating it, which reads as the control
            // being broken. The button carries transparent padding so the tap
            // area clears the 24px minimum while the bar stays thin.
            className={cn(
              "group/pip flex items-end justify-center bg-transparent px-[3px] py-2.5 -my-2.5 cursor-pointer",
              tall ? "w-[22px]" : "w-4",
            )}
          >
            <motion.span
              aria-hidden
              animate={{ scaleY: isChosen ? 1 : filled ? 0.88 : 0.66 }}
              whileTap={{ scaleY: 0.8 }}
              transition={{ type: "spring", stiffness: 520, damping: 24 }}
              style={{
                originY: 1,
                background: filled ? scoreColor(shown ?? value) : undefined,
              }}
              className={cn(
                "block w-full rounded-cube transition-colors",
                tall ? "h-9" : "h-6",
                !filled && "bg-felt-600 group-hover/pip:bg-rule",
              )}
            />
          </button>
        );
      })}

      <span
        className={cn(
          "ml-2 font-mono tabular-nums leading-none self-center",
          tall ? "text-lg" : "text-xs",
          shown === null ? "text-chalk-faint" : "text-chalk",
        )}
      >
        {shown === null ? (tall ? "rate it" : "—") : shown.toFixed(0)}
      </span>
    </div>
  );

  if (!isSignedIn) {
    return (
      <div className={className}>
        <SignInButton mode="modal">
          <button
            className="group/pips text-left"
            aria-label="Sign in to rate this game"
          >
            <div className="flex items-end gap-[3px] pointer-events-none">
              {VALUES.map((value) => (
                <span
                  key={value}
                  className={cn(
                    "rounded-cube bg-felt-600 group-hover/pips:bg-rule transition-colors",
                    tall ? "w-[16px] h-9 mx-[3px]" : "w-2.5 h-6 mx-[3px]",
                  )}
                />
              ))}
              <span
                className={cn(
                  "ml-2 self-center text-chalk-faint group-hover/pips:text-chalk transition-colors",
                  tall ? "text-sm" : "text-xs",
                )}
              >
                sign in to rate
              </span>
            </div>
          </button>
        </SignInButton>
      </div>
    );
  }

  return (
    <div className={className}>
      {pips}
      {error && (
        <p role="status" className="mt-1.5 text-xs text-meeple-red">
          {error}
        </p>
      )}
    </div>
  );
}
