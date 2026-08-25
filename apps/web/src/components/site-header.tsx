import Link from "next/link";
import { Suspense } from "react";
import { SignInButton, SignUpButton, Show, UserButton } from "@clerk/nextjs";
import { SearchField } from "./search-field";

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-40 border-b border-white/5 bg-felt-950/80 backdrop-blur-sm">
      <div className="mx-auto max-w-6xl px-5 h-16 flex items-center gap-6">
        <Link href="/" className="group flex items-center gap-2.5 shrink-0">
          <ShelfMark />
          <span className="font-display text-xl font-800 tracking-[-0.03em]">
            Shelf
          </span>
        </Link>

        <nav className="hidden sm:flex items-center gap-5 text-sm text-chalk-dim">
          <Link href="/games" className="hover:text-chalk transition-colors">
            Browse
          </Link>
          <Link href="/posts" className="hover:text-chalk transition-colors">
            Reading
          </Link>
        </nav>

        <div className="flex-1 min-w-0">
          {/* SearchField reads useSearchParams, which opts any page containing
              it out of static rendering. The header sits in the root layout, so
              without this boundary every static page — including 404 — fails to
              prerender. */}
          <Suspense fallback={<div className="h-8" />}>
            <SearchField />
          </Suspense>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          <Show when="signed-in">
            <Link
              href="/write"
              className="hidden sm:inline-flex text-sm text-chalk-dim hover:text-chalk transition-colors"
            >
              Write
            </Link>
            <UserButton
              appearance={{ elements: { avatarBox: "size-8" } }}
              userProfileProps={{ appearance: undefined }}
            />
          </Show>
          <Show when="signed-out">
            <SignInButton mode="modal">
              <button className="text-sm text-chalk-dim hover:text-chalk transition-colors">
                Sign in
              </button>
            </SignInButton>
            <SignUpButton mode="modal">
              <button className="text-sm font-medium bg-chalk text-felt-950 px-3.5 py-1.5 rounded-full hover:bg-meeple-amber transition-colors">
                Join
              </button>
            </SignUpButton>
          </Show>
        </div>
      </div>
    </header>
  );
}

/**
 * Three stacked bars: game boxes seen edge-on, which is what a shelf actually
 * looks like. The colours are three of the player colours used throughout.
 */
function ShelfMark() {
  return (
    <span aria-hidden className="flex items-end gap-[3px] h-5">
      <span className="w-[5px] h-3.5 rounded-[1px] bg-meeple-red transition-all duration-300 group-hover:h-5" />
      <span className="w-[5px] h-5 rounded-[1px] bg-meeple-amber transition-all duration-300 group-hover:h-3" />
      <span className="w-[5px] h-4 rounded-[1px] bg-meeple-teal transition-all duration-300 group-hover:h-5" />
    </span>
  );
}
