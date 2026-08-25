import Image from "next/image";
import { coverColor, initials, cn } from "@/lib/format";

type Props = {
  name: string;
  slug: string;
  src?: string | null;
  className?: string;
  sizes?: string;
  priority?: boolean;
};

/**
 * A game's cover.
 *
 * Most of the catalogue has no artwork until a BoardGameGeek token is
 * configured, so the fallback has to look deliberate rather than broken. It
 * builds a typographic tile from the title's initials over a player colour
 * picked deterministically from the slug — the same game is always the same
 * colour, and a wall of them reads like a shelf of spines.
 */
export function GameCover({ name, slug, src, className, sizes, priority }: Props) {
  if (src) {
    return (
      <div className={cn("relative overflow-hidden bg-felt-800", className)}>
        <Image
          src={src}
          alt=""
          fill
          sizes={sizes ?? "(max-width: 768px) 45vw, 220px"}
          className="object-cover"
          priority={priority}
        />
      </div>
    );
  }

  const color = coverColor(slug);

  return (
    <div
      className={cn("relative overflow-hidden bg-felt-800", className)}
      style={{ background: `linear-gradient(155deg, ${color}22, ${color}0A 55%)` }}
      aria-hidden
    >
      <span
        className="absolute inset-0 flex items-center justify-center font-display font-800 leading-none select-none"
        style={{ color, fontSize: "clamp(2rem, 22cqw, 5rem)", containerType: "inline-size" }}
      >
        {initials(name)}
      </span>
      <span
        className="absolute left-0 top-0 bottom-0 w-1"
        style={{ background: color }}
      />
    </div>
  );
}
