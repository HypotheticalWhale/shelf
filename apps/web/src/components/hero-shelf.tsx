"use client";

import Link from "next/link";
import { motion, useReducedMotion } from "motion/react";
import { coverColor } from "@/lib/format";
import type { Game } from "@/lib/types";

/**
 * The signature moment on the home page: a shelf of games standing up.
 *
 * Every spine is a real game from the catalogue, coloured from its own slug and
 * sized from the length of its title, so the row is different every time the
 * top of the chart moves. On load they drop in and settle onto the shelf with a
 * spring, left to right, the way you would slide boxes into place.
 *
 * One orchestrated entrance rather than motion scattered across the page —
 * and skipped entirely when the visitor has asked for reduced motion.
 */
export function HeroShelf({ games }: { games: Game[] }) {
  const reduced = useReducedMotion();
  const spines = games.slice(0, 22);

  if (spines.length === 0) return null;

  return (
    <section aria-label="Games on the shelf" className="mt-14">
      <div className="relative">
        <div className="flex items-end gap-[3px] overflow-hidden pt-10">
          {spines.map((game, i) => {
            const color = coverColor(game.slug);
            // Vary the height from the title so the row reads as objects on a
            // shelf rather than a bar chart.
            const height = 96 + ((game.name.length * 11) % 46);

            return (
              <motion.div
                key={game.slug}
                initial={reduced ? false : { y: -140, opacity: 0, rotate: -6 }}
                animate={{ y: 0, opacity: 1, rotate: 0 }}
                transition={{
                  type: "spring",
                  stiffness: 260,
                  damping: 18,
                  delay: reduced ? 0 : 0.18 + i * 0.045,
                }}
                whileHover={reduced ? undefined : { y: -10 }}
                className="shrink-0"
              >
                <Link
                  href={`/games/${game.slug}`}
                  title={game.name}
                  aria-label={game.name}
                  className="relative block rounded-t-[2px] overflow-hidden group"
                  style={{
                    height,
                    width: 30,
                    background: `linear-gradient(180deg, ${color}, ${color}C0)`,
                  }}
                >
                  {game.thumbnailUrl && (
                    // The game's own colours bleeding through the spine.
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={game.thumbnailUrl}
                      alt=""
                      aria-hidden
                      className="absolute inset-0 size-full object-cover opacity-45 blur-[6px] scale-150"
                    />
                  )}
                  <span
                    className="absolute inset-0 flex items-center justify-center font-display font-700 text-[10px] tracking-tight whitespace-nowrap text-felt-950/85 px-1"
                    style={{ writingMode: "vertical-rl", textOrientation: "mixed" }}
                  >
                    {game.name.length > 24 ? `${game.name.slice(0, 23)}…` : game.name}
                  </span>
                  <span className="absolute inset-0 ring-1 ring-inset ring-felt-950/25 rounded-t-[2px]" />
                </Link>
              </motion.div>
            );
          })}
        </div>

        {/* The shelf they land on, drawn after them. */}
        <motion.div
          initial={reduced ? false : { scaleX: 0, opacity: 0 }}
          animate={{ scaleX: 1, opacity: 1 }}
          transition={{ duration: 0.7, delay: reduced ? 0 : 0.1, ease: [0.22, 1, 0.36, 1] }}
          style={{ originX: 0 }}
          className="h-1.5 rounded-full bg-gradient-to-r from-rule via-rule-soft to-transparent"
        />
      </div>
    </section>
  );
}
