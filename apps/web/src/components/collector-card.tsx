"use client";

import Link from "next/link";
import { motion, useReducedMotion } from "motion/react";
import { coverColor, scoreColor } from "@/lib/format";
import type { Collector } from "@/lib/types";

/**
 * One person in the directory, shown as a miniature of their shelf.
 *
 * The point of the page is browsing taste, so the card leads with the spines
 * they actually own rather than a row of numbers. The counts sit underneath as
 * context.
 */
export function CollectorCard({
  collector,
  index = 0,
}: {
  collector: Collector;
  index?: number;
}) {
  const reduced = useReducedMotion();
  const { user, ownedCount, ratedCount, postCount, avgRating, shelfPeek } = collector;
  const name = user.displayName || user.username;

  return (
    <motion.article
      initial={reduced ? false : { opacity: 0, y: 14 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, delay: Math.min(index * 0.05, 0.4) }}
      className="group rounded-xl border border-rule-soft bg-felt-900/70 p-5 transition-colors hover:border-rule"
    >
      <div className="flex items-center gap-3">
        {user.avatarUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={user.avatarUrl}
            alt=""
            className="size-10 rounded-full border border-rule object-cover"
          />
        ) : (
          <div className="grid size-10 place-items-center rounded-full border border-rule bg-felt-800 font-display text-sm font-800 text-meeple-amber">
            {name.slice(0, 1).toUpperCase()}
          </div>
        )}

        <div className="min-w-0 flex-1">
          <Link href={`/u/${user.username}`}>
            <h2 className="truncate font-display text-lg font-700 tracking-[-0.02em] transition-colors group-hover:text-meeple-amber">
              {name}
            </h2>
          </Link>
          <p className="truncate font-mono text-[11px] text-chalk-faint">
            @{user.username}
          </p>
        </div>

        {ratedCount > 0 && (
          <div className="shrink-0 text-right">
            <p
              className="font-mono text-lg font-bold leading-none"
              style={{ color: scoreColor(avgRating) }}
            >
              {avgRating.toFixed(1)}
            </p>
            <p className="mt-0.5 font-mono text-[10px] text-chalk-faint">avg</p>
          </div>
        )}
      </div>

      {shelfPeek.length > 0 && (
        <Link href={`/u/${user.username}`} className="mt-4 block">
          <div className="flex items-end gap-[2px]">
            {shelfPeek.slice(0, 12).map((game, i) => {
              const color = coverColor(game.slug);
              const height = 38 + ((game.name.length * 7) % 26);
              return (
                <motion.span
                  key={game.slug}
                  initial={reduced ? false : { scaleY: 0 }}
                  animate={{ scaleY: 1 }}
                  transition={{
                    delay: reduced ? 0 : Math.min(index * 0.05, 0.4) + i * 0.03,
                    type: "spring",
                    stiffness: 300,
                    damping: 22,
                  }}
                  title={game.name}
                  style={{
                    height,
                    originY: 1,
                    background: `linear-gradient(180deg, ${color}, ${color}AA)`,
                  }}
                  className="block w-2.5 shrink-0 rounded-t-[2px] ring-1 ring-inset ring-felt-950/30"
                />
              );
            })}
          </div>
          <div className="mt-[3px] h-1 rounded-full bg-rule" />
        </Link>
      )}

      <p className="mt-4 font-mono text-[11px] text-chalk-faint">
        {[
          ownedCount > 0 && `${ownedCount} owned`,
          ratedCount > 0 && `${ratedCount} rated`,
          postCount > 0 && `${postCount} ${postCount === 1 ? "post" : "posts"}`,
        ]
          .filter(Boolean)
          .join(" · ") || "just arrived"}
      </p>
    </motion.article>
  );
}
