import { coverColor, initials, cn } from "@/lib/format";

type Props = {
  name: string;
  slug: string;
  src?: string | null;
  className?: string;
  sizes?: string;
  priority?: boolean;
  /** Render the lettering large, for game pages. */
  full?: boolean;
  /** Render it small, for search results and other inline uses. */
  compact?: boolean;
};

/** BGG's 64px crop — the only size the public snapshots publish. */
function isMicroThumb(src: string) {
  return src.includes("__micro") || src.includes("fit-in/64x64");
}

/**
 * A game's cover.
 *
 * Real artwork is shown whole, never cropped. Everything else gets a typographic
 * tile: the title's initials over a player colour derived from the slug, so a
 * game is always the same colour and a grid of them reads like a shelf of
 * spines.
 *
 * The 64x64 crop in the public snapshots is deliberately not used here. It is
 * the largest size available — the CDN signs each transform, so asking for
 * 300px returns 400 — and stretching it across a card only produced mush.
 * A tile that looks intentional beats a photograph that looks broken.
 *
 * Nothing needs to change when a BoardGameGeek token supplies real art: the
 * covers simply stop being micro thumbnails and start rendering as images.
 */
export function GameCover({ name, slug, src, className, priority, full, compact }: Props) {
  const usable = src && !isMicroThumb(src) ? src : null;

  if (usable) {
    return (
      <div className={cn("relative overflow-hidden bg-felt-800", className)}>
        {/*
          Box art is the game's identity, and it is not one shape: boxes come
          square, portrait and landscape. Cropping to fill the frame sliced the
          title off the artwork — Harmonies read as "MON" — so the art is
          contained and a blurred copy of it fills the space around it. The
          browser fetches the URL once and paints it twice.
        */}
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={usable}
          alt=""
          aria-hidden
          loading={priority ? "eager" : "lazy"}
          className="absolute inset-0 size-full scale-110 object-cover blur-2xl opacity-45"
        />
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={usable}
          alt=""
          loading={priority ? "eager" : "lazy"}
          className="absolute inset-0 size-full object-contain"
        />
      </div>
    );
  }

  const color = coverColor(slug);

  return (
    <div
      className={cn("relative overflow-hidden bg-felt-800", className)}
      style={{ background: `linear-gradient(155deg, ${color}26, ${color}0A 55%)` }}
      aria-hidden
    >
      <span
        className="absolute inset-0 flex items-center justify-center font-display font-800 leading-none select-none"
        style={{ color, fontSize: full ? "4.5rem" : compact ? "0.8rem" : "2.75rem" }}
      >
        {initials(name)}
      </span>
      <span className="absolute left-0 top-0 bottom-0 w-1" style={{ background: color }} />
    </div>
  );
}
