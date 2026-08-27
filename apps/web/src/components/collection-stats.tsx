"use client";

import { useEffect, useRef } from "react";
import {
  motion,
  useInView,
  useMotionValue,
  useSpring,
  useTransform,
} from "motion/react";
import { cn } from "@/lib/format";

type Band = {
  label: string;
  value: number;
  color: string;
  /** Set when the viewer holds this game on that shelf. */
  mine?: boolean;
};

/**
 * How many people keep this game, and on which shelf.
 *
 * The numbers count up when the panel comes into view and the bar fills to
 * match, so the shape of a game's following reads before the figures do — a
 * game everyone wants but nobody owns looks different at a glance from one
 * that is on every shelf.
 *
 * With nothing recorded yet it says so plainly rather than showing three
 * zeroes, which would read as broken.
 */
export function CollectionStats({
  owners,
  players,
  wanters,
  viewerShelf = [],
  className,
}: {
  owners: number;
  players: number;
  wanters: number;
  viewerShelf?: string[];
  className?: string;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { once: true, margin: "-40px" });

  const bands: Band[] = [
    { label: "own it", value: owners, color: "var(--color-meeple-teal)", mine: viewerShelf.includes("owned") },
    { label: "played it", value: players, color: "var(--color-meeple-amber)", mine: viewerShelf.includes("played") },
    { label: "want it", value: wanters, color: "var(--color-meeple-violet)", mine: viewerShelf.includes("wishlist") },
  ];

  const total = owners + players + wanters;

  return (
    <div ref={ref} className={cn("rounded-xl border border-rule-soft bg-felt-900/60 p-5", className)}>
      <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-chalk-faint">
        On other shelves
      </p>

      {total === 0 ? (
        <p className="mt-3 text-sm text-chalk-faint">
          No one has this on a shelf yet.
        </p>
      ) : (
        <>
          <div className="mt-4 flex flex-wrap gap-x-12 gap-y-4">
            {bands.map((b) => (
              <div key={b.label}>
                <Counter value={b.value} play={inView} color={b.color} />
                <p className="mt-1 flex items-center gap-1.5 text-[11px] text-chalk-dim">
                  {b.label}
                  {b.mine && (
                    <span
                      title="You"
                      className="inline-block size-1.5 rounded-full"
                      style={{ background: b.color }}
                    />
                  )}
                </p>
              </div>
            ))}
          </div>

          {/* One bar, split by shelf, so the balance is visible at a glance. */}
          <div className="mt-4 flex h-1.5 gap-0.5 overflow-hidden rounded-full bg-felt-700">
            {bands.filter((b) => b.value > 0).map((b) => (
              <motion.span
                key={b.label}
                initial={{ width: 0 }}
                animate={inView ? { width: `${(b.value / total) * 100}%` } : { width: 0 }}
                transition={{ type: "spring", stiffness: 120, damping: 22, delay: 0.15 }}
                style={{ background: b.color }}
                className="block"
              />
            ))}
          </div>
        </>
      )}
    </div>
  );
}

/** A number that counts up once, when it is first seen. */
function Counter({
  value,
  play,
  color,
}: {
  value: number;
  play: boolean;
  color: string;
}) {
  const raw = useMotionValue(0);
  const spring = useSpring(raw, { stiffness: 90, damping: 20, mass: 0.7 });
  const text = useTransform(spring, (v) => Math.round(v).toLocaleString());

  useEffect(() => {
    raw.set(play ? value : 0);
  }, [play, value, raw]);

  return (
    <motion.span
      className="block font-mono text-2xl font-bold leading-none tabular-nums"
      style={{ color }}
    >
      {text}
    </motion.span>
  );
}
