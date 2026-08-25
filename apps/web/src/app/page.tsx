import Link from "next/link";
import { apiGet } from "@/lib/api";
import { GameCard } from "@/components/game-card";
import { HeroRating } from "@/components/hero-rating";
import { HeroShelf } from "@/components/hero-shelf";
import { PostRow } from "@/components/post-row";
import type { GamePage, Post } from "@/lib/types";

export default async function HomePage() {
  // The shelf and the grid ask for different things: a wide row of spines, and
  // the ten games actually at the top of the chart.
  const [top, shelf, feed] = await Promise.all([
    apiGet<GamePage>("/games?sort=score&limit=10", { authenticated: true }),
    apiGet<GamePage>("/games?sort=score&limit=22&offset=10"),
    apiGet<{ posts: Post[] }>("/posts?limit=4"),
  ]);

  const featured = top.games[0];

  return (
    <div className="mx-auto max-w-6xl px-5">
      <section className="pt-16 pb-14 sm:pt-24 sm:pb-20">
        <p className="font-mono text-xs uppercase tracking-[0.22em] text-meeple-teal">
          Board games, minus the clutter
        </p>

        <h1 className="mt-5 font-display font-800 tracking-[-0.04em] leading-[0.92] text-[clamp(2.75rem,8vw,5.5rem)] max-w-4xl">
          Rating a game should take
          <span className="text-meeple-amber"> one tap</span>.
        </h1>

        <p className="mt-6 max-w-xl text-lg text-chalk-dim leading-relaxed">
          Keep the shelf you actually own, score what you have played, and write
          about the games worth arguing over. No forms, no submit buttons, no
          1998.
        </p>

        {featured && (
          <div className="mt-10">
            <HeroRating game={featured} />
          </div>
        )}

        <HeroShelf games={shelf.games} />
      </section>

      <section className="border-t border-rule-soft pt-12">
        <header className="flex items-baseline justify-between gap-4 flex-wrap">
          <h2 className="font-display font-700 text-2xl tracking-[-0.02em]">
            Highest rated
          </h2>
          <p className="text-sm text-chalk-faint max-w-md">
            Ranked with a Bayesian average, so a game needs more than one
            enthusiastic vote to reach the top. Games nobody here has rated yet
            fall back to BoardGameGeek&rsquo;s chart order.
          </p>
        </header>

        <div className="mt-7 grid gap-4 grid-cols-2 md:grid-cols-3 lg:grid-cols-5">
          {top.games.slice(0, 10).map((game, i) => (
            <GameCard key={game.slug} game={game} index={i} />
          ))}
        </div>

        <Link
          href="/games"
          className="mt-8 inline-flex items-center gap-2 text-sm text-chalk-dim hover:text-meeple-amber transition-colors"
        >
          Browse the whole catalogue
          <span aria-hidden>→</span>
        </Link>
      </section>

      {feed.posts.length > 0 && (
        <section className="border-t border-rule-soft mt-16 pt-12">
          <header className="flex items-baseline justify-between gap-4">
            <h2 className="font-display font-700 text-2xl tracking-[-0.02em]">
              From the shelves
            </h2>
            <Link
              href="/posts"
              className="text-sm text-chalk-dim hover:text-meeple-amber transition-colors"
            >
              All writing
            </Link>
          </header>

          <div className="mt-6 divide-y divide-rule-soft border-y border-rule-soft">
            {feed.posts.map((post) => (
              <PostRow key={post.id} post={post} />
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
