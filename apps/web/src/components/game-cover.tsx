import { coverColor, initials, cn } from "@/lib/format";

type Props = {
  name: string;
  slug: string;
  src?: string | null;
  className?: string;
  sizes?: string;
  priority?: boolean;
  /** Render the art large and sharp, for game pages. */
  full?: boolean;
};

/**
 * BGG's 64px crop, which must not be stretched across a whole card.
 *
 * The public snapshots publish only this size, and the CDN signs each
 * transform — changing the dimensions, dropping the filters, or asking for an
 * unsigned path all return 400. Real artwork only arrives with a BGG API token.
 */
function isMicroThumb(src: string) {
  return src.includes("__micro") || src.includes("fit-in/64x64");
}

/**
 * A game's cover.
 *
 * Three cases, in order of how much the source can carry:
 *
 *   1. Real artwork — rendered edge to edge.
 *   2. BGG's 64x64 thumbnail, which is all the public snapshots publish and
 *      cannot be enlarged because the CDN signs its transforms. Stretching it
 *      over a 4:3 card turns it to mush, so it is shown at close to its own
 *      size over a blurred copy of itself. The blur supplies the game's real
 *      colours, the inset keeps the art readable, and the result looks
 *      deliberate rather than broken.
 *   3. No art at all — a typographic tile built from the title's initials over
 *      a player colour picked deterministically from the slug.
 */
export function GameCover({ name, slug, src, className, priority, full }: Props) {
  const color = coverColor(slug);

  if (src && !isMicroThumb(src)) {
    return (
      <div className={cn("relative overflow-hidden bg-felt-800", className)}>
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={src}
          alt=""
          loading={priority ? "eager" : "lazy"}
          className="absolute inset-0 size-full object-cover"
        />
      </div>
    );
  }

  if (src) {
    return (
      <div className={cn("relative overflow-hidden bg-felt-900", className)}>
        {/* A blurred copy fills any letterboxing with the game's own colours. */}
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={src}
          alt=""
          aria-hidden
          loading={priority ? "eager" : "lazy"}
          className="absolute inset-0 size-full object-cover scale-125 blur-2xl opacity-70 saturate-150"
        />
        {/* The cover itself, filling the frame. Only a 64px source is available
            until a BGG token supplies real artwork, so it is upscaled — a
            little extra contrast keeps the edges from going to mush. */}
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={src}
          alt=""
          loading={priority ? "eager" : "lazy"}
          style={{ filter: "contrast(1.1) saturate(1.08)" }}
          className="absolute inset-0 size-full object-cover"
        />
      </div>
    );
  }

  return (
    <div
      className={cn("relative overflow-hidden bg-felt-800", className)}
      style={{ background: `linear-gradient(155deg, ${color}22, ${color}0A 55%)` }}
      aria-hidden
    >
      <span
        className="absolute inset-0 flex items-center justify-center font-display font-800 leading-none select-none"
        style={{ color, fontSize: full ? "4rem" : "2.5rem" }}
      >
        {initials(name)}
      </span>
      <span className="absolute left-0 top-0 bottom-0 w-1" style={{ background: color }} />
    </div>
  );
}
