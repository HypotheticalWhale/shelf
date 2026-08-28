"use client";

import { useOptimistic, useTransition } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { cn } from "@/lib/format";

const PLAYERS = [1, 2, 3, 4, 5, 6];
const LENGTHS = [
  { label: "≤ 30 min", value: "30" },
  { label: "≤ 60 min", value: "60" },
  { label: "≤ 2 hrs", value: "120" },
];
const SORTS = [
  { label: "Top rated", value: "score" },
  { label: "Most rated", value: "popular" },
  { label: "Newest", value: "new" },
  { label: "A–Z", value: "name" },
];

/**
 * Filters are chips rather than dropdowns: every option is visible, one click
 * applies it, and a second click clears it. Nothing to open, nothing to submit.
 */
export function BrowseFilters({ mechanics }: { mechanics: string[] }) {
  const router = useRouter();
  const params = useSearchParams();
  const [pending, startTransition] = useTransition();

  // A chip used to stay unpressed until the server had answered, because its
  // appearance came from the URL and the URL only changes once the navigation
  // commits. That is a quarter of a second of a chip that looks broken on a
  // fast connection, and much longer on a slow one. The click now paints
  // immediately and the query catches up: React discards this value once the
  // real search params arrive, and by then they agree, so nothing flickers.
  const [optimistic, setOptimistic] = useOptimistic(
    params.toString(),
    (_current, next: string) => next,
  );
  const shown = new URLSearchParams(optimistic);

  const navigate = (next: URLSearchParams) => {
    startTransition(() => {
      setOptimistic(next.toString());
      // scroll: false — changing a filter should leave you where you are
      // rather than throwing you back to the top of the page.
      router.push(`/games?${next}`, { scroll: false });
    });
  };

  // Values within a facet accumulate — picking two mechanics means "either" —
  // and clicking a selected chip removes just that one.
  const selected = (key: string) => {
    const raw = shown.get(key);
    return raw ? raw.split(",").filter(Boolean) : [];
  };

  const toggle = (key: string, value: string) => {
    const next = new URLSearchParams(shown);
    const current = selected(key);
    const updated = current.includes(value)
      ? current.filter((v) => v !== value)
      : [...current, value];

    if (updated.length === 0) {
      next.delete(key);
    } else {
      next.set(key, updated.join(","));
    }
    next.delete("page");
    navigate(next);
  };

  // Sort stays single-select: a list has one order.
  const applySort = (value: string) => {
    const next = new URLSearchParams(shown);
    next.set("sort", value);
    next.delete("page");
    navigate(next);
  };

  const clearAll = () => {
    const next = new URLSearchParams(shown);
    for (const key of ["players", "maxTime", "mechanic", "detailed"]) next.delete(key);
    next.delete("page");
    navigate(next);
  };

  const activeCount =
    selected("players").length + selected("maxTime").length + selected("mechanic").length;

  // Only the commonest mechanics are offered, so a chosen one can fall outside
  // that list — arriving from a link, or after the counts shift. Showing it
  // anyway keeps every active filter visible and, more importantly, removable.
  const shownMechanics = [
    ...selected("mechanic").filter((m) => !mechanics.includes(m)),
    ...mechanics,
  ];

  return (
    <div className="mt-6 flex flex-col gap-3" data-browse-filters aria-busy={pending}>
      <FilterRow label="Players">
        {PLAYERS.map((n) => (
          <Chip
            key={n}
            active={selected("players").includes(String(n))}
            onClick={() => toggle("players", String(n))}
          >
            {n === 1 ? "Solo" : n === 6 ? "6+" : n}
          </Chip>
        ))}
      </FilterRow>

      <FilterRow label="Length">
        {LENGTHS.map((l) => (
          <Chip
            key={l.value}
            active={selected("maxTime").includes(l.value)}
            onClick={() => toggle("maxTime", l.value)}
          >
            {l.label}
          </Chip>
        ))}
      </FilterRow>

      {shownMechanics.length > 0 && (
        <FilterRow label="Plays like">
          {shownMechanics.map((m) => (
            <Chip
              key={m}
              active={selected("mechanic").includes(m)}
              onClick={() => toggle("mechanic", m)}
            >
              {m}
            </Chip>
          ))}
        </FilterRow>
      )}

      <FilterRow label="Sort">
        {SORTS.map((s) => (
          <Chip
            key={s.value}
            active={(shown.get("sort") ?? "score") === s.value}
            onClick={() => applySort(s.value)}
          >
            {s.label}
          </Chip>
        ))}
      </FilterRow>

      {activeCount > 0 && (
        <button
          type="button"
          onClick={clearAll}
          className="self-start font-mono text-[10px] uppercase tracking-[0.14em] text-chalk-faint transition-colors hover:text-meeple-amber"
        >
          Clear {activeCount} filter{activeCount === 1 ? "" : "s"} ×
        </button>
      )}
    </div>
  );
}

function FilterRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-3">
      <span className="shrink-0 whitespace-nowrap font-mono text-[10px] uppercase tracking-[0.14em] text-chalk-faint sm:w-[86px] sm:self-start sm:pt-1.5">
        {label}
      </span>
      <div className="flex gap-1.5 flex-wrap">{children}</div>
    </div>
  );
}

function Chip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        "px-3 py-1 rounded-full text-xs border transition-colors",
        active
          ? "bg-meeple-amber text-felt-950 border-meeple-amber font-medium"
          : "border-rule-soft text-chalk-dim hover:border-rule hover:text-chalk",
      )}
    >
      {children}
    </button>
  );
}
