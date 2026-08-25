import Link from "next/link";
import { PenLine } from "lucide-react";
import { Suspense } from "react";
import { SignInButton, SignUpButton, Show, UserButton } from "@clerk/nextjs";
import { SearchField } from "./search-field";
import { MobileNav } from "./mobile-nav";

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-40 border-b border-white/5 bg-felt-950/80 backdrop-blur-sm">
      <div className="mx-auto flex h-16 max-w-7xl items-center gap-4 px-5 lg:gap-6">
        <Link href="/" className="group flex items-center gap-2.5 shrink-0">
          <ShelfMark />
          <span className="hidden font-display text-xl font-800 tracking-[-0.03em] sm:inline">
            Shelf
          </span>
        </Link>

        <Show when="signed-in">
          <MobileNav signedIn />
        </Show>
        <Show when="signed-out">
          <MobileNav signedIn={false} />
        </Show>

        <nav className="hidden items-center gap-4 text-sm text-chalk-dim lg:flex">
          <Link href="/games" className="hover:text-chalk transition-colors">
            Browse
          </Link>
          <Link href="/people" className="hover:text-chalk transition-colors">
            People
          </Link>
          <Link href="/posts" className="hover:text-chalk transition-colors">
            Reading
          </Link>
          <Show when="signed-in">
            <Link href="/shelf" className="hover:text-chalk transition-colors">
              My shelf
            </Link>
          </Show>
        </nav>

        {/*
          Search gets the middle of the bar rather than a corner: with tens of
          thousands of games it is the main way in, and it answers in place.

          SearchField reads useSearchParams, which opts any page containing it
          out of static rendering. The header sits in the root layout, so
          without this boundary every static page — including 404 — fails to
          prerender.
        */}
        <div className="mx-auto min-w-0 max-w-xl flex-1">
          <Suspense fallback={<div className="h-9" />}>
            <SearchField />
          </Suspense>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          <Show when="signed-in">
            <Link
              href="/write"
              className="hidden items-center gap-1.5 rounded-full border border-rule-soft px-3 py-1.5 text-sm text-chalk-dim transition-colors hover:border-rule hover:text-chalk sm:inline-flex"
            >
              <PenLine className="size-3.5" aria-hidden />
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
