/** Shared display helpers, so the same game reads identically on every screen. */

export function playerRange(min: number | null, max: number | null): string | null {
  if (!min && !max) return null;
  if (min && max && min !== max) return `${min}–${max} players`;
  const n = min ?? max;
  return n === 1 ? "solo" : `${n} players`;
}

export function playtime(min: number | null, max: number | null): string | null {
  if (!min && !max) return null;
  if (min && max && min !== max) return `${min}–${max} min`;
  return `${min ?? max} min`;
}

export function weightLabel(weight: number | null): string | null {
  if (!weight) return null;
  if (weight < 1.8) return "light";
  if (weight < 2.6) return "medium-light";
  if (weight < 3.4) return "medium";
  if (weight < 4.2) return "medium-heavy";
  return "heavy";
}

/**
 * Colour for a score on the 1–10 scale.
 *
 * Runs red → amber → teal. Green is deliberately avoided: red/green is the
 * pairing most often lost to colour vision deficiency, and teal keeps the
 * "good" end distinguishable for far more people.
 */
export function scoreColor(value: number): string {
  const v = Math.max(1, Math.min(10, value));
  if (v < 4) return "var(--color-meeple-red)";
  if (v < 5.5) return "#E8803F";
  if (v < 7) return "var(--color-meeple-amber)";
  if (v < 8.5) return "#8FBE5A";
  return "var(--color-meeple-teal)";
}

/** Deterministic player colour for a game, used for generated cover art. */
export function coverColor(seed: string): string {
  const palette = [
    "#E25642",
    "#F0A73E",
    "#35B8A0",
    "#4E93DC",
    "#A07DE6",
    "#E27BA8",
  ];
  let hash = 0;
  for (let i = 0; i < seed.length; i += 1) {
    hash = (hash * 31 + seed.charCodeAt(i)) >>> 0;
  }
  return palette[hash % palette.length];
}

/** Up to two initials from a game title, for generated covers. */
export function initials(name: string): string {
  const words = name
    .replace(/[^\p{L}\p{N} ]/gu, " ")
    .split(/\s+/)
    .filter(Boolean);
  if (words.length === 0) return "??";
  if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
  return (words[0][0] + words[1][0]).toUpperCase();
}

/**
 * Publication year, handling the ancient games in the catalogue.
 *
 * Senet, Backgammon and Go predate the common era, and BGG stores those as
 * negative years. Printing "-3000" looks like a bug, so render the era.
 */
export function formatYear(year: number | null): string | null {
  if (year === null || year === 0) return null;
  return year < 0 ? `${Math.abs(year)} BC` : String(year);
}

export function formatDate(iso: string | null): string {
  if (!iso) return "";
  return new Date(iso).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

export function cn(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(" ");
}

/** Joins a list the way a sentence does: "a", "a and b", "a, b and c". */
function sentenceList(items: string[], conjunction: "and" | "or"): string {
  if (items.length <= 1) return items[0] ?? "";
  if (items.length === 2) return `${items[0]} ${conjunction} ${items[1]}`;
  return `${items.slice(0, -1).join(", ")} ${conjunction} ${items[items.length - 1]}`;
}

/**
 * Describes the browse query in a sentence.
 *
 * The heading used to become whatever you had filtered for, so the page you
 * were on changed its name as you worked and never said what the filters
 * actually meant. Saying it here in plain words leaves the heading fixed and
 * makes the two different semantics legible: "for 2 and 5 players" is one game
 * that seats both, while "with X or Y" is any game carrying either.
 *
 * Clauses are comma-separated rather than strung together with "and", because
 * they describe different things — a length and a mechanic are not two halves
 * of one thought.
 */
export function describeQuery(opts: {
  total: number;
  query?: string;
  players?: string;
  maxTime?: string;
  mechanics?: string;
}): string {
  const count = opts.total.toLocaleString();
  const noun = opts.total === 1 ? "game" : "games";

  const clauses: string[] = [];

  if (opts.query) clauses.push(`matching \u201C${opts.query}\u201D`);

  const players = (opts.players ?? "")
    .split(",")
    .map(Number)
    .filter((n) => !Number.isNaN(n) && n > 0)
    .sort((a, b) => a - b);
  if (players.length === 1 && players[0] === 1) {
    clauses.push("for solo play");
  } else if (players.length > 0) {
    const labels = players.map((n) => (n >= 6 ? "6+" : String(n)));
    // "and", not "or": choosing 2 and 5 asks for one game that seats both.
    clauses.push(`for ${sentenceList(labels, "and")} players`);
  }

  const minutes = (opts.maxTime ?? "")
    .split(",")
    .map(Number)
    .filter((n) => !Number.isNaN(n) && n > 0);
  if (minutes.length > 0) {
    const shortest = Math.min(...minutes);
    clauses.push(shortest >= 120 ? "under 2 hours" : `under ${shortest} min`);
  }

  const mechanics = (opts.mechanics ?? "").split(",").filter(Boolean);
  if (mechanics.length > 0) {
    // "or": any one of them is a match, which is how discovery works.
    clauses.push(`with ${sentenceList(mechanics, "or")}`);
  }

  if (clauses.length === 0) return `${count} ${noun}`;
  return `${count} ${noun} ${clauses.join(", ")}`;
}
