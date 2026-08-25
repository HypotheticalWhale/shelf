import { notFound } from "next/navigation";
import Link from "next/link";
import { apiGetOrNull } from "@/lib/api";
import { PostRow } from "@/components/post-row";
import { SpineRail } from "@/components/spine-rail";
import { formatDate, scoreColor } from "@/lib/format";
import type { Profile } from "@/lib/types";

export async function generateMetadata({ params }: PageProps<"/u/[username]">) {
  const { username } = await params;
  return { title: `${username}’s shelf` };
}

export default async function ProfilePage({ params }: PageProps<"/u/[username]">) {
  const { username } = await params;

  const profile = await apiGetOrNull<Profile>(`/users/${username}`, { authenticated: true });
  if (!profile) notFound();

  const { user, posts, recentRatings, shelf } = profile;
  const owned = shelf.filter((item) => item.status === "owned");

  return (
    <div className="shell max-w-5xl py-12">
      <header className="flex items-start gap-5">
        {user.avatarUrl ? (
          // Clerk avatars are already sized; a plain img avoids a loader round trip.
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={user.avatarUrl}
            alt=""
            className="size-16 rounded-full border border-rule object-cover"
          />
        ) : (
          <div className="size-16 rounded-full border border-rule bg-felt-800 flex items-center justify-center font-display font-800 text-xl text-meeple-amber">
            {(user.displayName || user.username).slice(0, 1).toUpperCase()}
          </div>
        )}

        <div className="min-w-0">
          <h1 className="font-display font-800 text-3xl tracking-[-0.03em]">
            {user.displayName || user.username}
          </h1>
          <p className="font-mono text-xs text-chalk-faint mt-1">
            @{user.username} · joined {formatDate(user.createdAt)}
          </p>
          {user.bio && <p className="mt-3 text-chalk-dim max-w-xl">{user.bio}</p>}
        </div>
      </header>

      {owned.length > 0 && (
        <section className="mt-12">
          <h2 className="font-mono text-[11px] uppercase tracking-[0.18em] text-chalk-faint">
            On the shelf
          </h2>
          <SpineRail items={owned} />
        </section>
      )}

      <section className="mt-12">
        <h2 className="font-display font-700 text-2xl tracking-[-0.02em]">Writing</h2>
        {posts.length === 0 ? (
          <p className="mt-3 text-sm text-chalk-faint">
            No posts yet.{" "}
            <Link href="/write" className="text-meeple-amber hover:underline">
              Write the first one.
            </Link>
          </p>
        ) : (
          <div className="mt-4 divide-y divide-rule-soft border-y border-rule-soft">
            {posts.map((post) => (
              <PostRow key={post.id} post={post} showAuthor={false} />
            ))}
          </div>
        )}
      </section>

      {recentRatings.length > 0 && (
        <section className="mt-12">
          <h2 className="font-display font-700 text-2xl tracking-[-0.02em]">
            Recently rated
          </h2>
          <ul className="mt-4 divide-y divide-rule-soft border-y border-rule-soft">
            {recentRatings.map((r) => (
              <li key={r.gameId} className="flex items-center gap-4 py-3">
                <span
                  className="font-mono font-bold text-lg tabular-nums w-9 shrink-0"
                  style={{ color: scoreColor(r.value) }}
                >
                  {r.value.toFixed(0)}
                </span>
                <Link
                  href={`/games/${r.game?.slug ?? ""}`}
                  className="flex-1 min-w-0 truncate hover:text-meeple-amber transition-colors"
                >
                  {r.game?.name}
                </Link>
                <span className="font-mono text-[11px] text-chalk-faint shrink-0">
                  {formatDate(r.updatedAt)}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}
