"use client";

import { useEffect, useRef, useState } from "react";
import { apiRequest } from "@/lib/client";
import { cn, formatYear } from "@/lib/format";
import type { Game, GamePage } from "@/lib/types";

type Props = {
  /** Slug to preselect, e.g. when arriving from a game page. */
  defaultSlug?: string;
  onChange: (slug: string | null) => void;
};

/**
 * Attaches a post to a game by searching for it.
 *
 * This replaces a free-text field that asked for a "game-slug": nobody outside
 * the codebase knows what a slug is, and typing the actual title — "Brass:
 * Birmingham" — failed the whole publish with a bare "not found". Searching
 * makes the valid values discoverable, so the error cannot be reached.
 */
export function GamePicker({ defaultSlug, onChange }: Props) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<Game[]>([]);
  const [selected, setSelected] = useState<Game | null>(null);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const boxRef = useRef<HTMLDivElement>(null);

  // Resolve a preselected slug to a real game so the field shows a title.
  useEffect(() => {
    if (!defaultSlug) return;
    let cancelled = false;
    apiRequest<Game>(`/games/${defaultSlug}`)
      .then((game) => {
        if (cancelled) return;
        setSelected(game);
        onChange(game.slug);
      })
      .catch(() => {
        // A bad ?game= parameter simply leaves the field empty rather than
        // blocking the post.
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [defaultSlug]);

  // Debounced search.
  useEffect(() => {
    const q = query.trim();
    if (selected || q.length < 2) {
      setResults([]);
      return;
    }
    setLoading(true);
    const timer = setTimeout(async () => {
      try {
        const page = await apiRequest<GamePage>(
          `/games?q=${encodeURIComponent(q)}&limit=6`,
        );
        setResults(page.games);
        setOpen(true);
      } catch {
        setResults([]);
      } finally {
        setLoading(false);
      }
    }, 250);
    return () => clearTimeout(timer);
  }, [query, selected]);

  // Close the dropdown when focus or the pointer goes elsewhere.
  useEffect(() => {
    const onDocClick = (e: MouseEvent) => {
      if (!boxRef.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, []);

  const choose = (game: Game) => {
    setSelected(game);
    setQuery("");
    setResults([]);
    setOpen(false);
    onChange(game.slug);
  };

  const clear = () => {
    setSelected(null);
    setQuery("");
    onChange(null);
  };

  if (selected) {
    return (
      <div className="flex items-center gap-2 flex-wrap">
        <span className="inline-flex items-center gap-2 bg-felt-700 border border-rule rounded-full pl-3 pr-1.5 py-1 text-sm">
          {selected.name}
          {selected.yearPublished && (
            <span className="font-mono text-[11px] text-chalk-faint">
              {formatYear(selected.yearPublished)}
            </span>
          )}
          <button
            type="button"
            onClick={clear}
            aria-label={`Remove ${selected.name}`}
            className="size-5 rounded-full grid place-items-center text-chalk-faint hover:text-chalk hover:bg-felt-600 transition-colors"
          >
            ×
          </button>
        </span>
      </div>
    );
  }

  return (
    <div ref={boxRef} className="relative flex-1 min-w-0">
      <input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onFocus={() => results.length > 0 && setOpen(true)}
        placeholder="Search for a game (optional)"
        aria-label="Attach a game to this post"
        role="combobox"
        aria-expanded={open}
        aria-controls="game-picker-results"
        autoComplete="off"
        className="w-full bg-felt-800 border border-rule-soft rounded-md px-3 py-1.5 text-sm placeholder:text-chalk-faint outline-none focus:border-rule transition-colors"
      />

      {open && results.length > 0 && (
        <ul
          id="game-picker-results"
          role="listbox"
          className="absolute z-30 mt-1 w-full max-h-64 overflow-y-auto rounded-md border border-rule bg-felt-800 shadow-xl"
        >
          {results.map((game) => (
            <li key={game.slug} role="option" aria-selected={false}>
              <button
                type="button"
                onClick={() => choose(game)}
                className="w-full text-left px-3 py-2 text-sm hover:bg-felt-700 flex items-baseline justify-between gap-3 transition-colors"
              >
                <span className="truncate">{game.name}</span>
                <span className="font-mono text-[11px] text-chalk-faint shrink-0">
                  {formatYear(game.yearPublished) ?? ""}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}

      {loading && query.trim().length >= 2 && (
        <span className={cn("absolute right-3 top-1/2 -translate-y-1/2 text-[11px] text-chalk-faint")}>
          searching…
        </span>
      )}
    </div>
  );
}
