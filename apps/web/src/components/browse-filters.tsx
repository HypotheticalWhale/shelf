"use client";

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
export function BrowseFilters({ genres }: { genres: string[] }) {
  const router = useRouter();
  const params = useSearchParams();

  const apply = (key: string, value: string | null) => {
    const next = new URLSearchParams(params);
    if (value === null || next.get(key) === value) {
      next.delete(key);
    } else {
      next.set(key, value);
    }
    next.delete("page");
    router.push(`/games?${next}`);
  };

  return (
    <div className="mt-6 flex flex-col gap-3">
      <FilterRow label="Players">
        {PLAYERS.map((n) => (
          <Chip
            key={n}
            active={params.get("players") === String(n)}
            onClick={() => apply("players", String(n))}
          >
            {n === 1 ? "Solo" : n === 6 ? "6+" : n}
          </Chip>
        ))}
      </FilterRow>

      <FilterRow label="Length">
        {LENGTHS.map((l) => (
          <Chip
            key={l.value}
            active={params.get("maxTime") === l.value}
            onClick={() => apply("maxTime", l.value)}
          >
            {l.label}
          </Chip>
        ))}
      </FilterRow>

      {genres.length > 0 && (
        <FilterRow label="Genre">
          {genres.map((g) => (
            <Chip
              key={g}
              active={params.get("category") === g}
              onClick={() => apply("category", g)}
            >
              {g}
            </Chip>
          ))}
        </FilterRow>
      )}

      <FilterRow label="Sort">
        {SORTS.map((s) => (
          <Chip
            key={s.value}
            active={(params.get("sort") ?? "score") === s.value}
            onClick={() => apply("sort", s.value)}
          >
            {s.label}
          </Chip>
        ))}
      </FilterRow>
    </div>
  );
}

function FilterRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-3 flex-wrap">
      <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-chalk-faint w-14 shrink-0">
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
