import Link from "next/link";
import { apiGet } from "@/lib/api";
import { CollectorCard } from "@/components/collector-card";
import type { Collector } from "@/lib/types";

export const metadata = { title: "People" };

export default async function PeoplePage() {
  const { collectors } = await apiGet<{ collectors: Collector[] }>(
    "/collectors?limit=48",
    { revalidate: 60 },
  );

  const active = collectors.filter(
    (c) => c.ownedCount + c.ratedCount + c.postCount > 0,
  );

  return (
    <div className="mx-auto max-w-6xl px-5 py-12">
      <header>
        <p className="font-mono text-[11px] uppercase tracking-[0.22em] text-meeple-teal">
          The community
        </p>
        <h1 className="mt-3 font-display text-4xl font-800 tracking-[-0.03em]">
          Other people&rsquo;s shelves
        </h1>
        <p className="mt-2 max-w-lg text-chalk-dim">
          Who owns what, who rates hard, and who is writing about it. Every
          shelf is a link into somebody&rsquo;s taste.
        </p>
      </header>

      {active.length === 0 ? (
        <div className="mt-12 rounded-2xl border border-rule bg-felt-900/60 px-6 py-14 text-center">
          <p className="font-display text-xl font-700">Nobody else yet.</p>
          <p className="mx-auto mt-2 max-w-sm text-sm text-chalk-dim">
            Shelves appear here as soon as people start rating and collecting.
          </p>
          <Link
            href="/games"
            className="mt-6 inline-block rounded-full bg-chalk px-5 py-2 font-medium text-felt-950 transition-colors hover:bg-meeple-amber"
          >
            Be the first
          </Link>
        </div>
      ) : (
        <div className="mt-10 grid gap-4 sm:grid-cols-2">
          {active.map((c, i) => (
            <CollectorCard key={c.user.id} collector={c} index={i} />
          ))}
        </div>
      )}
    </div>
  );
}
