"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { motion, useReducedMotion } from "motion/react";
import { coverColor, scoreColor } from "@/lib/format";
import type { Game } from "@/lib/types";

export type ShelfEntry = {
  game: Game;
  /** Shown on the spine's end cap when the row is a rating list. */
  rating?: number;
};

const SPINE_W = 32;
const GAP = 3;

/** Roughly how many characters fit down a spine of this height. */
function capacity(height: number, fontPx: number, hasCap: boolean) {
  const usable = height - (hasCap ? 30 : 14);
  return Math.floor(usable / (fontPx * 0.62));
}

/**
 * Sizes a spine to its title instead of at random.
 *
 * A board game box is as tall as it needs to be, and a long title deserves a
 * taller spine — so height grows with the name, and the type steps down a
 * point before anything is cut. Truncation is the last resort rather than the
 * first, which is what made "Dune: Imperium – Uprising" read as clipped.
 */
function fitSpine(name: string, hasCap: boolean) {
  const maxH = 190;
  let fontPx = 10;
  let height = Math.min(maxH, Math.round(96 + Math.min(name.length, 44) * 2.1));

  if (capacity(height, fontPx, hasCap) < name.length) {
    height = maxH;
  }
  if (capacity(height, fontPx, hasCap) < name.length) {
    fontPx = 9;
  }

  const room = capacity(height, fontPx, hasCap);
  const label =
    name.length > room ? `${name.slice(0, Math.max(4, room - 1))}…` : name;

  return { height, fontPx, label, truncated: label !== name };
}

/**
 * One labelled shelf, wrapping into a bookcase when it fills up.
 *
 * A single scrolling row hides everything past the edge once a collection
 * grows, so the spines wrap onto as many boards as they need. The number per
 * board is measured from the container rather than assumed, so it stays right
 * across breakpoints.
 */
export function ShelfRow({
  label,
  hint,
  entries,
  empty,
  accent,
}: {
  label: string;
  hint?: string;
  entries: ShelfEntry[];
  empty: React.ReactNode;
  accent: string;
}) {
  const reduced = useReducedMotion();
  const boxRef = useRef<HTMLDivElement>(null);
  const [perShelf, setPerShelf] = useState(0);

  useEffect(() => {
    const el = boxRef.current;
    if (!el) return;
    const measure = () => {
      const fit = Math.floor((el.clientWidth + GAP) / (SPINE_W + GAP));
      setPerShelf(Math.max(1, fit));
    };
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const shelves: ShelfEntry[][] = [];
  if (perShelf > 0) {
    for (let i = 0; i < entries.length; i += perShelf) {
      shelves.push(entries.slice(i, i + perShelf));
    }
  }

  return (
    <section className="mt-14 first:mt-0">
      <header className="flex flex-wrap items-baseline justify-between gap-4">
        <h2 className="font-display text-xl font-700 tracking-[-0.02em]">
          {label}
          <span className="ml-2.5 font-mono text-xs text-chalk-faint">
            {entries.length}
          </span>
        </h2>
        {hint && <p className="text-xs text-chalk-faint">{hint}</p>}
      </header>

      <div ref={boxRef} className="mt-4">
        {entries.length === 0 ? (
          <div className="rounded-xl border border-dashed border-rule-soft px-5 py-8 text-center text-sm text-chalk-faint">
            {empty}
          </div>
        ) : (
          shelves.map((shelf, shelfIndex) => (
            <div key={shelfIndex} className={shelfIndex > 0 ? "mt-7" : undefined}>
              <div className="flex items-end gap-[3px] pt-10">
                {shelf.map(({ game, rating }, i) => {
                  const color = coverColor(game.slug);
                  const { height, fontPx, label: spineLabel, truncated } = fitSpine(
                    game.name,
                    rating !== undefined,
                  );
                  const order = shelfIndex * perShelf + i;

                  return (
                    <motion.div
                      key={game.slug}
                      initial={reduced ? false : { y: -110, opacity: 0, rotate: -5 }}
                      animate={{ y: 0, opacity: 1, rotate: 0 }}
                      transition={{
                        type: "spring",
                        stiffness: 240,
                        damping: 19,
                        delay: reduced ? 0 : Math.min(order * 0.03, 1.1),
                      }}
                      whileHover={reduced ? undefined : { y: -12 }}
                      className="shrink-0"
                    >
                      <Link
                        href={`/games/${game.slug}`}
                        // The full name is always reachable, even when the
                        // spine could not hold every character.
                        title={truncated ? game.name : undefined}
                        aria-label={game.name}
                        className="relative block overflow-hidden rounded-t-[3px]"
                        style={{
                          height,
                          width: SPINE_W,
                          background: `linear-gradient(180deg, ${color}, ${color}BB)`,
                        }}
                      >
                        {game.imageUrl && (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img
                            src={game.imageUrl}
                            alt=""
                            aria-hidden
                            className="absolute inset-0 size-full scale-125 object-cover opacity-45 blur-[6px]"
                          />
                        )}

                        <span
                          className="absolute inset-x-0 top-1.5 flex items-center justify-center whitespace-nowrap px-1 font-display font-700 tracking-tight text-felt-950/85"
                          style={{
                            bottom: rating !== undefined ? 24 : 8,
                            fontSize: fontPx,
                            writingMode: "vertical-rl",
                            textOrientation: "mixed",
                          }}
                        >
                          {spineLabel}
                        </span>

                        {rating !== undefined && (
                          <span
                            className="absolute inset-x-0 bottom-0 grid h-5 place-items-center font-mono text-[10px] font-bold text-felt-950"
                            style={{ background: scoreColor(rating) }}
                          >
                            {rating.toFixed(0)}
                          </span>
                        )}

                        <span className="absolute inset-0 rounded-t-[3px] ring-1 ring-inset ring-felt-950/25" />
                      </Link>
                    </motion.div>
                  );
                })}
              </div>

              <motion.div
                initial={reduced ? false : { scaleX: 0 }}
                animate={{ scaleX: 1 }}
                transition={{
                  duration: 0.6,
                  delay: reduced ? 0 : shelfIndex * 0.08,
                  ease: [0.22, 1, 0.36, 1],
                }}
                style={{ originX: 0, background: accent }}
                className="h-1.5 rounded-full"
              />
            </div>
          ))
        )}
      </div>
    </section>
  );
}
