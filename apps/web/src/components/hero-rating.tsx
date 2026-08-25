"use client";

import Link from "next/link";
import { useState } from "react";
import { GameCover } from "./game-cover";
import { RatingPips } from "./rating-pips";
import { ScoreBadge } from "./score-badge";
import { formatYear, playerRange, playtime } from "@/lib/format";
import type { Game } from "@/lib/types";

/**
 * The hero is the product demonstrating itself.
 *
 * Rather than describing one-tap rating, the page hands you a real game and a
 * live control: rate it here and the score moves, before you have navigated
 * anywhere or read a feature list.
 */
export function HeroRating({ game: initial }: { game: Game }) {
  const [game, setGame] = useState(initial);

  const meta = [formatYear(game.yearPublished), playerRange(game.minPlayers, game.maxPlayers), playtime(game.minPlaytime, game.maxPlaytime)]
    .filter(Boolean)
    .join(" · ");

  return (
    <div className="flex flex-col sm:flex-row sm:items-center gap-5 rounded-xl border border-rule bg-felt-900/80 p-4 sm:p-5 max-w-2xl backdrop-blur">
      <Link href={`/games/${game.slug}`} className="shrink-0">
        <GameCover
          name={game.name}
          slug={game.slug}
          src={game.thumbnailUrl}
          className="w-full sm:w-28 aspect-[4/3] sm:aspect-square rounded-lg"
          sizes="112px"
          priority
        />
      </Link>

      <div className="flex-1 min-w-0">
        <Link href={`/games/${game.slug}`}>
          <h2 className="font-display font-700 text-xl tracking-[-0.02em] hover:text-meeple-amber transition-colors">
            {game.name}
          </h2>
        </Link>
        <p className="mt-0.5 font-mono text-[11px] text-chalk-faint">{meta}</p>

        <div className="mt-3">
          <ScoreBadge score={game.score} numRatings={game.numRatings} />
        </div>

        <RatingPips
          slug={game.slug}
          viewerRating={game.viewerRating}
          size="lg"
          onRated={setGame}
          className="mt-3"
        />
      </div>
    </div>
  );
}
