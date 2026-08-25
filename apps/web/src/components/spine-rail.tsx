"use client";

import Link from "next/link";
import { motion } from "motion/react";
import { coverColor } from "@/lib/format";
import type { ShelfItem } from "@/lib/types";

/**
 * A collection, drawn the way you actually see one at home: box spines lined up
 * edge-on. Titles run vertically, colour comes from the same deterministic
 * player-colour used for covers, and hovering pulls a box halfway out.
 */
export function SpineRail({ items }: { items: ShelfItem[] }) {
  return (
    <div className="mt-3 relative">
      <div className="flex items-end gap-[3px] overflow-x-auto pb-3 pt-6">
        {items.map((item, i) => {
          const game = item.game;
          if (!game) return null;
          const color = coverColor(game.slug);
          // Vary the heights a little so the row reads as physical objects
          // rather than a chart.
          const height = 104 + ((game.slug.length * 7) % 34);

          return (
            <motion.div
              key={game.slug}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: Math.min(i * 0.025, 0.35), duration: 0.3 }}
              whileHover={{ y: -8 }}
            >
              <Link
                href={`/games/${game.slug}`}
                title={game.name}
                className="block rounded-t-[2px] shrink-0 relative overflow-hidden group"
                style={{
                  height,
                  width: 26,
                  background: `linear-gradient(180deg, ${color}, ${color}CC)`,
                }}
              >
                <span
                  className="absolute inset-0 flex items-center justify-center font-display font-700 text-[10px] tracking-tight whitespace-nowrap text-felt-950/85"
                  style={{ writingMode: "vertical-rl", textOrientation: "mixed" }}
                >
                  {game.name.length > 22 ? `${game.name.slice(0, 21)}…` : game.name}
                </span>
              </Link>
            </motion.div>
          );
        })}
      </div>
      {/* The shelf the boxes stand on. */}
      <div className="h-1 rounded-full bg-rule" />
    </div>
  );
}
