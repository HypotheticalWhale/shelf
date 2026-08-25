"use client";

import { useState } from "react";
import { motion } from "motion/react";
import { useLiveScore } from "./live-scores";
import { RatingPips } from "./rating-pips";
import { ScoreBadge } from "./score-badge";
import { ShelfControls } from "./shelf-controls";
import { scoreColor } from "@/lib/format";
import type { GameDetail, Game } from "@/lib/types";

/**
 * The rating surface on a game page: your vote, the aggregate, and an honest
 * account of how the two relate.
 */
export function GamePanel({ game: initial }: { game: GameDetail }) {
  const [game, setGame] = useState<GameDetail>(initial);

  // Live aggregates from other people's ratings.
  const live = useLiveScore(game.slug);
  const shown = live ? { ...game, ...live } : game;

  const onRated = (updated: Game) => {
    // The API returns the recomputed aggregates; the histogram is only refreshed
    // on reload, so adjust the viewer's own bar locally to keep it honest.
    setGame((prev) => {
      const histogram = [...prev.histogram];
      if (prev.viewerRating != null) {
        const old = Math.min(9, Math.max(0, Math.round(prev.viewerRating) - 1));
        histogram[old] = Math.max(0, histogram[old] - 1);
      }
      if (updated.viewerRating != null) {
        const next = Math.min(9, Math.max(0, Math.round(updated.viewerRating) - 1));
        histogram[next] += 1;
      }
      return { ...prev, ...updated, histogram };
    });
  };

  const peak = Math.max(1, ...game.histogram);

  return (
    <div className="mt-6 rounded-xl border border-rule bg-felt-900/70 p-5 sm:p-6">
      <div className="flex flex-wrap items-start justify-between gap-6">
        <div>
          <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-chalk-faint">
            Shelf score
          </p>
          <div className="mt-2">
            <ScoreBadge score={shown.score} numRatings={shown.numRatings} size="lg" />
          </div>
          {shown.numRatings > 0 && (
            <p className="mt-1.5 font-mono text-[11px] text-chalk-faint">
              raw average {shown.mean.toFixed(2)}
            </p>
          )}
        </div>

        <div>
          <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-chalk-faint">
            Your rating
          </p>
          <RatingPips
            slug={game.slug}
            viewerRating={game.viewerRating}
            size="lg"
            onRated={onRated}
            className="mt-2.5"
          />
        </div>
      </div>

      {game.numRatings > 0 && (
        <div className="mt-7">
          <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-chalk-faint">
            Spread
          </p>
          <div className="mt-2.5 flex items-end gap-1 h-16 max-w-xs" aria-hidden>
            {game.histogram.map((count, i) => (
              <motion.div
                key={i}
                className="flex-1 rounded-cube min-h-[2px]"
                style={{ background: scoreColor(i + 1), opacity: count ? 1 : 0.15 }}
                initial={{ height: 0 }}
                animate={{ height: `${(count / peak) * 100}%` }}
                transition={{ type: "spring", stiffness: 200, damping: 22 }}
                title={`${count} rated it ${i + 1}`}
              />
            ))}
          </div>
          <div className="mt-1 flex justify-between font-mono text-[10px] text-chalk-faint max-w-xs">
            <span>1</span>
            <span>10</span>
          </div>
        </div>
      )}

      <div className="mt-7 pt-5 border-t border-rule-soft">
        <ShelfControls slug={game.slug} initial={game.viewerShelf ?? []} />
      </div>

      <p className="mt-5 text-xs text-chalk-faint leading-relaxed max-w-lg">
        The shelf score is a Bayesian average: every game starts weighted toward
        the site-wide mean and earns its way out as real ratings arrive. It is
        why a single perfect score will not put an unknown game at number one.
      </p>
    </div>
  );
}
