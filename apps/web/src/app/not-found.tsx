import Link from "next/link";

export default function NotFound() {
  return (
    <div className="mx-auto max-w-xl px-5 py-28 text-center">
      <p className="font-mono text-xs uppercase tracking-[0.22em] text-meeple-teal">
        404
      </p>
      <h1 className="mt-4 font-display font-800 text-4xl tracking-[-0.03em]">
        Not on this shelf.
      </h1>
      <p className="mt-3 text-chalk-dim">
        The page you asked for is not here. It may have been renamed, or never
        existed.
      </p>
      <Link
        href="/games"
        className="mt-7 inline-block bg-chalk text-felt-950 font-medium px-5 py-2 rounded-full hover:bg-meeple-amber transition-colors"
      >
        Browse games
      </Link>
    </div>
  );
}
