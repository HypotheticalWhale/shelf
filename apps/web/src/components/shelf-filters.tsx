"use client";

import { useMemo, useState } from "react";
import { ShelfRow, type ShelfEntry } from "./shelf-wall";
import { cn } from "@/lib/format";

const PLAYERS = [1, 2, 3, 4, 5, 6];

/**
 * The shelf, with its own filters.
 *
 * Filtering happens in the browser: a collection is at most a couple of hundred
 * games and they are already loaded, so narrowing it is instant and costs no
 * round trip. Mechanics are drawn from the shelf itself rather than the whole
 * catalogue, so every chip is one that will actually match something.
 *
 * Mechanics are OR'd — two of them means "either". Player counts are AND'd,
 * because picking 2 and 4 asks for games that seat both, not games that seat
 * one or the other.
 */
export function ShelfFilters({
  owned,
  wanted,
}: {
  owned: ShelfEntry[];
  wanted: ShelfEntry[];
}) {
  const [mechanics, setMechanics] = useState<string[]>([]);
  const [players, setPlayers] = useState<number[]>([]);

  const all = useMemo(() => [...owned, ...wanted], [owned, wanted]);

  // Only offer mechanics this collection actually contains, commonest first.
  const available = useMemo(() => {
    const counts = new Map<string, number>();
    for (const { game } of all) {
      for (const m of game.mechanics ?? []) {
        counts.set(m, (counts.get(m) ?? 0) + 1);
      }
    }
    return [...counts.entries()]
      .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
      .slice(0, 12)
      .map(([m]) => m);
  }, [all]);

  const matches = (entry: ShelfEntry) => {
    const { game } = entry;

    if (mechanics.length > 0) {
      const has = (game.mechanics ?? []).some((m) => mechanics.includes(m));
      if (!has) return false;
    }
    if (players.length > 0) {
      // Selecting counts describes the group sizes to cover, so the game has
      // to seat all of them — its range only needs to reach the lowest and the
      // highest asked for.
      if (game.minPlayers == null || game.maxPlayers == null) return false;
      const lo = Math.min(...players);
      const hi = Math.max(...players);
      if (game.minPlayers > lo || game.maxPlayers < hi) return false;
    }
    return true;
  };

  const filteredOwned = owned.filter(matches);
  const filteredWanted = wanted.filter(matches);
  const active = mechanics.length + players.length;

  const toggle = <T,>(list: T[], set: (v: T[]) => void, value: T) =>
    set(list.includes(value) ? list.filter((v) => v !== value) : [...list, value]);

  return (
    <>
      <div className="mt-8 flex flex-col gap-3 rounded-xl border border-rule-soft bg-felt-900/50 p-4">
        <Row label="Players">
          {PLAYERS.map((n) => (
            <Chip
              key={n}
              active={players.includes(n)}
              onClick={() => toggle(players, setPlayers, n)}
            >
              {n === 1 ? "Solo" : n === 6 ? "6+" : n}
            </Chip>
          ))}
        </Row>

        {available.length > 0 && (
          <Row label="Plays like">
            {available.map((m) => (
              <Chip
                key={m}
                active={mechanics.includes(m)}
                onClick={() => toggle(mechanics, setMechanics, m)}
              >
                {m}
              </Chip>
            ))}
          </Row>
        )}

        {active > 0 && (
          <button
            type="button"
            onClick={() => {
              setMechanics([]);
              setPlayers([]);
            }}
            className="self-start font-mono text-[10px] uppercase tracking-[0.14em] text-chalk-faint transition-colors hover:text-meeple-amber"
          >
            Clear {active} filter{active === 1 ? "" : "s"} ×
          </button>
        )}
      </div>

      <div className="mt-10">
        <ShelfRow
          label="Owned"
          hint={
            active > 0
              ? `${filteredOwned.length} of ${owned.length} match`
              : "Boxes actually on your shelf"
          }
          accent="var(--color-meeple-teal)"
          entries={filteredOwned}
          empty={
            active > 0 ? (
              <>Nothing you own matches these filters.</>
            ) : (
              <>Nothing marked as owned yet.</>
            )
          }
        />

        <ShelfRow
          label="Want"
          hint={
            active > 0
              ? `${filteredWanted.length} of ${wanted.length} match`
              : "The wishlist"
          }
          accent="var(--color-meeple-violet)"
          entries={filteredWanted}
          empty={
            active > 0 ? (
              <>Nothing on the wishlist matches these filters.</>
            ) : (
              <>Nothing on the wishlist.</>
            )
          }
        />
      </div>
    </>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-3">
      <span className="shrink-0 whitespace-nowrap font-mono text-[10px] uppercase tracking-[0.14em] text-chalk-faint sm:w-[86px] sm:self-start sm:pt-1.5">
        {label}
      </span>
      <div className="flex flex-wrap gap-1.5">{children}</div>
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
        "rounded-full border px-3 py-1 text-xs transition-colors",
        active
          ? "border-meeple-amber bg-meeple-amber font-medium text-felt-950"
          : "border-rule-soft text-chalk-dim hover:border-rule hover:text-chalk",
      )}
    >
      {children}
    </button>
  );
}
