"use client";

import Link from "next/link";
import { useState } from "react";
import { motion } from "motion/react";
import { useLiveScore } from "./live-scores";
import { GameCover } from "./game-cover";
import { RatingPips } from "./rating-pips";
import { ScoreBadge } from "./score-badge";
import { formatYear, playerRange, playtime } from "@/lib/format";
import type { Game } from "@/lib/types";

/**
 * A game in the browse grid.
 *
 * The rating control sits on the card itself. That is the entire point: on BGG
 * rating something means opening its page, finding the widget and submitting.
 * Here it costs one click, from wherever you happen to be looking.
 */
export function GameCard({ game: initial, index = 0 }: { game: Game; index?: number }) {
  const [game, setGame] = useState(initial);

  // Somebody else rating this game updates the card in place. Only the
  // aggregates follow the stream — a viewer's own rating is theirs alone.
  const live = useLiveScore(game.slug);
  const shown = live ? { ...game, ...live } : game;

  const players = playerRange(game.minPlayers, game.maxPlayers);
  const time = playtime(game.minPlaytime, game.maxPlaytime);

  return (
    <motion.article
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, delay: Math.min(index * 0.03, 0.4), ease: [0.22, 1, 0.36, 1] }}
      className="group flex flex-col rounded-lg border border-rule-soft bg-felt-900/70 overflow-hidden hover:border-rule transition-colors"
    >
      <Link href={`/games/${game.slug}`} className="block">
        <GameCover
          name={game.name}
          slug={game.slug}
          src={game.thumbnailUrl}
          className="aspect-[4/3] w-full"
        />
      </Link>

      <div className="flex flex-col gap-3 p-3.5 flex-1">
        <div className="flex-1 min-w-0">
          <Link href={`/games/${game.slug}`} className="block">
            <h3 className="font-display font-700 text-[15px] leading-snug tracking-[-0.01em] line-clamp-2 group-hover:text-meeple-amber transition-colors">
              {game.name}
            </h3>
          </Link>
          <p className="mt-1 font-mono text-[11px] text-chalk-faint truncate">
            {[formatYear(game.yearPublished), players, time].filter(Boolean).join(" · ")}
          </p>
          {game.mechanics && game.mechanics.length > 0 && (
            <p className="mt-1.5 truncate font-mono text-[10px] uppercase tracking-wider text-meeple-teal">
              {game.mechanics.slice(0, 2).join(" · ")}
            </p>
          )}
        </div>

        <div className="flex items-center justify-between gap-2">
          <ScoreBadge score={shown.score} numRatings={shown.numRatings} />
        </div>

        <RatingPips
          slug={game.slug}
          viewerRating={game.viewerRating}
          onRated={setGame}
          className="pt-0.5"
        />
      </div>
    </motion.article>
  );
}
