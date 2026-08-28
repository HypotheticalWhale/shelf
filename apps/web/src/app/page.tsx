import Link from "next/link";
import { apiGet } from "@/lib/api";
import { LandingExperience } from "@/components/landing-experience";
import { PostRow } from "@/components/post-row";
import type { GamePage, Post } from "@/lib/types";

export default async function HomePage() {
  const [top, feed] = await Promise.all([
    apiGet<GamePage>("/games?sort=score&limit=4", { authenticated: true }),
    apiGet<{ posts: Post[] }>("/posts?limit=4", { revalidate: 60 }),
  ]);

  return (
    <>
      <LandingExperience games={top.games} />

      {feed.posts.length > 0 && (
        <section className="shell pt-16">
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
    </>
  );
}
