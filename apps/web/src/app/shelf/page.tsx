import { auth } from "@clerk/nextjs/server";
import { redirect } from "next/navigation";
import Link from "next/link";
import { apiGet } from "@/lib/api";
import { ShelfRow, type ShelfEntry } from "@/components/shelf-wall";
import type { Rating, ShelfItem, User } from "@/lib/types";

export const metadata = { title: "My shelf" };

export default async function MyShelfPage() {
  const { userId } = await auth();
  if (!userId) redirect("/sign-in?redirect_url=/shelf");

  const me = await apiGet<User>("/me", { authenticated: true });

  // Every request here is marked authenticated so it is fetched fresh. These
  // are public endpoints, but apiGet caches unauthenticated reads for a minute
  // — which meant a game you had just added did not appear on your own shelf
  // until the cache expired.
  const [owned, wishlist, rated] = await Promise.all([
    apiGet<{ shelf: ShelfItem[] }>(`/users/${me.username}/shelf?status=owned&limit=200`, {
      authenticated: true,
    }),
    apiGet<{ shelf: ShelfItem[] }>(`/users/${me.username}/shelf?status=wishlist&limit=200`, {
      authenticated: true,
    }),
    apiGet<{ ratings: Rating[] }>(`/users/${me.username}/ratings?limit=200`, {
      authenticated: true,
    }),
  ]);

  const toEntries = (items: ShelfItem[]): ShelfEntry[] =>
    items.filter((i) => i.game).map((i) => ({ game: i.game! }));

  const ratedEntries: ShelfEntry[] = rated.ratings
    .filter((r) => r.game)
    .map((r) => ({ game: r.game!, rating: r.value }));

  const total =
    owned.shelf.length + wishlist.shelf.length + rated.ratings.length;

  return (
    <div className="mx-auto max-w-6xl px-5 py-12">
      <header className="flex items-end justify-between gap-4 flex-wrap">
        <div>
          <p className="font-mono text-[11px] uppercase tracking-[0.22em] text-meeple-teal">
            Your collection
          </p>
          <h1 className="mt-3 font-display font-800 text-4xl tracking-[-0.03em]">
            My shelf
          </h1>
        </div>
        <Link
          href={`/u/${me.username}`}
          className="text-sm text-chalk-dim transition-colors hover:text-meeple-amber"
        >
          See your public profile →
        </Link>
      </header>

      {total === 0 ? (
        <div className="mt-12 rounded-2xl border border-rule bg-felt-900/60 px-6 py-14 text-center">
          <p className="font-display font-700 text-xl">Nothing on the shelf yet.</p>
          <p className="mx-auto mt-2 max-w-sm text-sm text-chalk-dim">
            Rate a game or mark one as owned and it appears here, spine out,
            like the real thing.
          </p>
          <Link
            href="/games"
            className="mt-6 inline-block rounded-full bg-chalk px-5 py-2 font-medium text-felt-950 transition-colors hover:bg-meeple-amber"
          >
            Find games
          </Link>
        </div>
      ) : (
        <div className="mt-10">
          <ShelfRow
            label="Owned"
            hint="Boxes actually on your shelf"
            accent="var(--color-meeple-teal)"
            entries={toEntries(owned.shelf)}
            empty={
              <>
                Nothing marked as owned.{" "}
                <Link href="/games" className="text-meeple-amber hover:underline">
                  Add something you have.
                </Link>
              </>
            }
          />

          <ShelfRow
            label="Rated"
            hint="Your score on each spine"
            accent="var(--color-meeple-amber)"
            entries={ratedEntries}
            empty={
              <>
                You have not rated anything yet. It takes one tap from{" "}
                <Link href="/games" className="text-meeple-amber hover:underline">
                  the catalogue
                </Link>
                .
              </>
            }
          />

          <ShelfRow
            label="Want"
            hint="The wishlist"
            accent="var(--color-meeple-violet)"
            entries={toEntries(wishlist.shelf)}
            empty={<>Nothing on the wishlist.</>}
          />
        </div>
      )}
    </div>
  );
}
