import { apiGet } from "@/lib/api";
import { GameCard } from "@/components/game-card";
import { BrowseFilters } from "@/components/browse-filters";
import type { GamePage } from "@/lib/types";

export const metadata = { title: "Browse" };

// The grid runs 2, 3, 4, 5 and 6 columns across its breakpoints, and 60 is the
// lowest common multiple of those — so every page fills whole rows at every
// width. At 24 the five-column layout ended a page one card short.
const PAGE_SIZE = 60;

export default async function BrowsePage({ searchParams }: PageProps<"/games">) {
  const params = await searchParams;
  const get = (key: string) => {
    const v = params[key];
    return Array.isArray(v) ? v[0] : v;
  };

  const page = Number(get("page") ?? "1") || 1;
  const query = new URLSearchParams({
    limit: String(PAGE_SIZE),
    offset: String((page - 1) * PAGE_SIZE),
  });
  for (const key of ["q", "players", "maxTime", "sort", "mechanic"]) {
    const value = get(key);
    if (value) query.set(key, value);
  }

  const [result, mechanicList] = await Promise.all([
    apiGet<GamePage>(`/games?${query}`, { authenticated: true }),
    apiGet<{ mechanics: string[] }>("/mechanics?limit=14", { revalidate: 3600 }),
  ]);
  const lastPage = Math.max(1, Math.ceil(result.total / PAGE_SIZE));

  return (
    <div className="shell py-10">
      <header className="flex items-end justify-between gap-4 flex-wrap">
        <div>
          <h1 className="font-display font-800 text-4xl tracking-[-0.03em]">
            {get("q")
            ? `“${get("q")}”`
            : (get("mechanic")?.split(",").join(" · ") ?? "Every game")}
          </h1>
          <p className="mt-1.5 font-mono text-xs text-chalk-faint">
            {result.total} {result.total === 1 ? "game" : "games"}
          </p>
        </div>
      </header>

      <BrowseFilters mechanics={mechanicList.mechanics} />

      {result.games.length === 0 ? (
        <div className="mt-16 text-center">
          <p className="font-display font-700 text-xl">Nothing on this shelf.</p>
          <p className="mt-2 text-chalk-dim text-sm">
            Try a shorter search, or loosen the player and length filters.
          </p>
        </div>
      ) : (
        <div className="mt-7 grid gap-4 grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6">
          {result.games.map((game, i) => (
            <GameCard key={game.slug} game={game} index={i} />
          ))}
        </div>
      )}

      {lastPage > 1 && (
        <nav className="mt-12 flex items-center justify-center gap-3 font-mono text-sm">
          <PageLink params={params} page={page - 1} disabled={page <= 1}>
            ← Previous
          </PageLink>
          <span className="text-chalk-faint">
            {page} / {lastPage}
          </span>
          <PageLink params={params} page={page + 1} disabled={page >= lastPage}>
            Next →
          </PageLink>
        </nav>
      )}
    </div>
  );
}

function PageLink({
  params,
  page,
  disabled,
  children,
}: {
  params: Record<string, string | string[] | undefined>;
  page: number;
  disabled: boolean;
  children: React.ReactNode;
}) {
  if (disabled) {
    return <span className="text-chalk-faint/50 px-3 py-1.5">{children}</span>;
  }

  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (key === "page" || value === undefined) continue;
    query.set(key, Array.isArray(value) ? value[0] : value);
  }
  query.set("page", String(page));

  return (
    <a
      href={`/games?${query}`}
      className="px-3 py-1.5 rounded-full border border-rule-soft hover:border-rule hover:text-meeple-amber transition-colors"
    >
      {children}
    </a>
  );
}
