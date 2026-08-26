"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
import { Search } from "lucide-react";
import { apiRequest } from "@/lib/client";
import { GameCover } from "./game-cover";
import { cn, formatYear, playerRange } from "@/lib/format";
import type { Game, GamePage } from "@/lib/types";

/**
 * Catalogue search, as a typeahead.
 *
 * With tens of thousands of games, search is the main way into the catalogue,
 * so it takes the middle of the bar rather than a corner of it — and it answers
 * in place. Results carry the cover, the year and how the game plays, which is
 * usually enough to recognise the right one without opening anything.
 *
 * Cmd-K, Ctrl-K and "/" all focus it; the arrow keys walk the results and Enter
 * opens the highlighted one, falling back to the full results page.
 */
export function SearchField() {
  // Narrow screens get a shorter prompt rather than a truncated one.
  const [compact, setCompact] = useState(false);
  useEffect(() => {
    const mq = window.matchMedia("(max-width: 640px)");
    const sync = () => setCompact(mq.matches);
    sync();
    mq.addEventListener("change", sync);
    return () => mq.removeEventListener("change", sync);
  }, []);

  const router = useRouter();
  const params = useSearchParams();

  // Depend on the query string, not on the params object: useSearchParams
  // hands back a fresh instance on every render, so an effect keyed to it
  // re-ran constantly — clearing the field and closing the results as fast as
  // they arrived.
  const urlQuery = params.get("q") ?? "";

  const [value, setValue] = useState(urlQuery);
  const [results, setResults] = useState<Game[]>([]);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const [loading, setLoading] = useState(false);

  const boxRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Keep the field in step when navigation changes the query behind it.
  useEffect(() => {
    setValue(urlQuery);
    setOpen(false);
  }, [urlQuery]);

  useEffect(() => {
    const q = value.trim();
    if (q.length < 2) {
      setResults([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    const timer = setTimeout(async () => {
      try {
        const page = await apiRequest<GamePage>(
          `/games?q=${encodeURIComponent(q)}&limit=6`,
        );
        setResults(page.games);
        setActive(-1);
        setOpen(true);
      } catch {
        setResults([]);
      } finally {
        setLoading(false);
      }
    }, 200);
    return () => clearTimeout(timer);
  }, [value]);

  useEffect(() => {
    const onDocDown = (e: MouseEvent) => {
      if (!boxRef.current?.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      const typingElsewhere =
        e.target instanceof HTMLElement &&
        /^(INPUT|TEXTAREA)$/.test(e.target.tagName);

      if ((e.key === "k" && (e.metaKey || e.ctrlKey)) || (e.key === "/" && !typingElsewhere)) {
        e.preventDefault();
        inputRef.current?.focus();
        inputRef.current?.select();
      }
    };
    document.addEventListener("mousedown", onDocDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDocDown);
      document.removeEventListener("keydown", onKey);
    };
  }, []);

  const go = useCallback(
    (href: string) => {
      setOpen(false);
      inputRef.current?.blur();
      router.push(href);
    },
    [router],
  );

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      setOpen(false);
      inputRef.current?.blur();
      return;
    }
    if (!open || results.length === 0) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((i) => (i + 1) % results.length);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((i) => (i <= 0 ? results.length - 1 : i - 1));
    } else if (e.key === "Enter" && active >= 0) {
      e.preventDefault();
      go(`/games/${results[active].slug}`);
    }
  };

  return (
    <div ref={boxRef} className="relative w-full">
      <form
        role="search"
        onSubmit={(e) => {
          e.preventDefault();
          const q = value.trim();
          go(q ? `/games?q=${encodeURIComponent(q)}` : "/games");
        }}
      >
        <Search
          className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-chalk-faint"
          aria-hidden
        />
        <input
          ref={inputRef}
          type="search"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onFocus={() => results.length > 0 && setOpen(true)}
          onKeyDown={onKeyDown}
          placeholder={compact ? "Search games" : "Search games, designers, mechanics…"}
          aria-label="Search games"
          role="combobox"
          aria-expanded={open}
          aria-controls="search-results"
          autoComplete="off"
          className="w-full rounded-full border border-rule-soft bg-felt-800/80 py-2 pl-10 pr-16 text-sm outline-none transition-colors placeholder:text-chalk-faint focus:border-rule focus:bg-felt-800"
        />

        {/* The shortcut, shown only while the field is idle. */}
        {!value && (
          <kbd className="pointer-events-none absolute right-3 top-1/2 hidden -translate-y-1/2 rounded border border-rule-soft px-1.5 py-0.5 font-mono text-[10px] text-chalk-faint sm:block">
            ⌘K
          </kbd>
        )}
        {loading && value.trim().length >= 2 && (
          <span className="absolute right-3 top-1/2 -translate-y-1/2 font-mono text-[10px] text-chalk-faint">
            …
          </span>
        )}
      </form>

      {open && results.length > 0 && (
        <div
          id="search-results"
          role="listbox"
          className="absolute left-0 right-0 top-full z-50 mt-2 overflow-hidden rounded-xl border border-rule bg-felt-900 shadow-2xl shadow-felt-950/70"
        >
          {results.map((game, i) => (
            <Link
              key={game.slug}
              href={`/games/${game.slug}`}
              role="option"
              aria-selected={i === active}
              onMouseEnter={() => setActive(i)}
              onClick={() => setOpen(false)}
              className={cn(
                "flex items-center gap-3 px-3 py-2.5 transition-colors",
                i === active ? "bg-felt-700" : "hover:bg-felt-800",
              )}
            >
              {/* The same typographic tile the cards use, so a result looks
                  like the game it will open. */}
              <GameCover
                name={game.name}
                slug={game.slug}
                src={game.thumbnailUrl}
                className="size-9 shrink-0 rounded"
                compact
              />

              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm">{game.name}</span>
                <span className="block truncate font-mono text-[10px] text-chalk-faint">
                  {[
                    formatYear(game.yearPublished),
                    playerRange(game.minPlayers, game.maxPlayers),
                    game.mechanics?.[0],
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </span>
              </span>

              {game.numRatings > 0 && (
                <span className="shrink-0 font-mono text-xs text-chalk-dim">
                  {game.score.toFixed(1)}
                </span>
              )}
            </Link>
          ))}

          <button
            type="button"
            onClick={() => go(`/games?q=${encodeURIComponent(value.trim())}`)}
            className="w-full border-t border-rule-soft px-3 py-2 text-left font-mono text-[11px] text-chalk-faint transition-colors hover:bg-felt-800 hover:text-chalk"
          >
            See all results for “{value.trim()}” →
          </button>
        </div>
      )}
    </div>
  );
}
