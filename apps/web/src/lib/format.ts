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
