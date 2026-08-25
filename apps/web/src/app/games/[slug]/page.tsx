import { notFound } from "next/navigation";
import Link from "next/link";
import { apiGetOrNull, apiGet } from "@/lib/api";
import { GameCover } from "@/components/game-cover";
import { GamePanel } from "@/components/game-panel";
import { PostRow } from "@/components/post-row";
import { formatYear, playerRange, playtime, weightLabel } from "@/lib/format";
import type { GameDetail, Post } from "@/lib/types";

export async function generateMetadata({ params }: PageProps<"/games/[slug]">) {
  const { slug } = await params;
  const game = await apiGetOrNull<GameDetail>(`/games/${slug}`);
  return { title: game?.name ?? "Game" };
}

export default async function GamePage({ params }: PageProps<"/games/[slug]">) {
  const { slug } = await params;

  const game = await apiGetOrNull<GameDetail>(`/games/${slug}`, { authenticated: true });
  if (!game) notFound();

  const { posts } = await apiGet<{ posts: Post[] }>(`/games/${slug}/posts?limit=8`);

  const facts = [
    ["Players", playerRange(game.minPlayers, game.maxPlayers)],
    ["Length", playtime(game.minPlaytime, game.maxPlaytime)],
    [
      "Complexity",
      game.weight ? `${game.weight.toFixed(1)} · ${weightLabel(game.weight)}` : null,
    ],
    ["Published", formatYear(game.yearPublished)],
    ["Designer", game.designers?.slice(0, 2).join(", ") || null],
  ].filter(([, value]) => Boolean(value)) as [string, string][];

  return (
    <div className="shell py-10">
      <div className="grid gap-10 lg:grid-cols-[300px_1fr]">
        <div className="lg:sticky lg:top-8 lg:self-start">
          <GameCover
            name={game.name}
            slug={game.slug}
            src={game.imageUrl ?? game.thumbnailUrl}
            className="w-full aspect-square rounded-xl border border-rule-soft"
            sizes="300px"
            priority
            full
          />

          <dl className="mt-5 divide-y divide-rule-soft border-y border-rule-soft">
            {facts.map(([label, value]) => (
              <div key={label} className="flex justify-between gap-4 py-2.5">
                <dt className="font-mono text-[11px] uppercase tracking-[0.14em] text-chalk-faint">
                  {label}
                </dt>
                <dd className="text-sm text-right">{value}</dd>
              </div>
            ))}
          </dl>

          <TagGroup label="Themes" tags={game.categories} accent="text-meeple-amber" />
          <TagGroup label="How it plays" tags={game.mechanics} accent="text-meeple-teal" />
        </div>

        <div className="min-w-0">
          <h1 className="font-display font-800 text-[clamp(2rem,5vw,3.25rem)] leading-[0.95] tracking-[-0.035em]">
            {game.name}
          </h1>

          <GamePanel game={game} />

          {game.description && (
            <section className="mt-10 max-w-2xl">
              <h2 className="font-mono text-[11px] uppercase tracking-[0.18em] text-chalk-faint">
                About
              </h2>
              <p className="mt-3 text-chalk-dim leading-relaxed whitespace-pre-line">
                {game.description}
              </p>
            </section>
          )}

          <section className="mt-12">
            <div className="flex items-baseline justify-between gap-4">
              <h2 className="font-display font-700 text-xl tracking-[-0.02em]">
                What people wrote
              </h2>
              <Link
                href={`/write?game=${game.slug}`}
                className="text-sm text-chalk-dim hover:text-meeple-amber transition-colors"
              >
                Write about {game.name}
              </Link>
            </div>

            {posts.length === 0 ? (
              <p className="mt-4 text-sm text-chalk-faint">
                Nobody has written about this one yet. Be first.
              </p>
            ) : (
              <div className="mt-3 divide-y divide-rule-soft border-y border-rule-soft">
                {posts.map((post) => (
                  <PostRow key={post.id} post={post} />
                ))}
              </div>
            )}
          </section>
        </div>
      </div>
    </div>
  );
}

function TagGroup({
  label,
  tags,
  accent,
}: {
  label: string;
  tags?: string[];
  accent: string;
}) {
  if (!tags || tags.length === 0) return null;

  return (
    <div className="mt-5">
      <p className="font-mono text-[10px] uppercase tracking-[0.18em] text-chalk-faint">
        {label}
      </p>
      <ul className="mt-2 flex flex-wrap gap-1.5">
        {tags.map((tag) => (
          <li
            key={tag}
            className={`font-mono text-[10px] uppercase tracking-wider ${accent} border border-rule-soft rounded-full px-2 py-0.5`}
          >
            {tag}
          </li>
        ))}
      </ul>
    </div>
  );
}
