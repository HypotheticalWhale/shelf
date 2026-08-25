"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { Menu, X } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { cn } from "@/lib/format";

type Item = { href: string; label: string; signedInOnly?: boolean };

const ITEMS: Item[] = [
  { href: "/games", label: "Browse" },
  { href: "/people", label: "People" },
  { href: "/posts", label: "Reading" },
  { href: "/shelf", label: "My shelf", signedInOnly: true },
  { href: "/write", label: "Write", signedInOnly: true },
];

/**
 * Navigation for narrow screens.
 *
 * The desktop bar hides its links below the large breakpoint, which left phones
 * with no way to reach Browse, People or their own shelf at all — the only
 * controls on the bar were search and the avatar. This puts every destination
 * one tap away.
 */
export function MobileNav({ signedIn }: { signedIn: boolean }) {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();

  // A route change should close the menu, or it covers the page you asked for.
  useEffect(() => setOpen(false), [pathname]);

  // While the sheet is open the page behind it should not scroll.
  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = prev;
    };
  }, [open]);

  const items = ITEMS.filter((i) => !i.signedInOnly || signedIn);

  return (
    <div className="lg:hidden">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        aria-label={open ? "Close menu" : "Open menu"}
        className="grid size-9 place-items-center rounded-full border border-rule-soft text-chalk-dim transition-colors hover:text-chalk"
      >
        {open ? <X className="size-4" /> : <Menu className="size-4" />}
      </button>

      <AnimatePresence>
        {open && (
          <>
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              onClick={() => setOpen(false)}
              className="fixed inset-x-0 bottom-0 top-16 z-40 bg-felt-950/70 backdrop-blur-sm"
            />
            <motion.nav
              initial={{ y: -12, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              exit={{ y: -12, opacity: 0 }}
              transition={{ duration: 0.18, ease: [0.22, 1, 0.36, 1] }}
              className="fixed inset-x-0 top-16 z-50 border-b border-rule bg-felt-900 px-5 py-3 shadow-2xl"
            >
              {items.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className={cn(
                    "block border-b border-rule-soft py-3 text-base last:border-0",
                    pathname.startsWith(item.href)
                      ? "text-meeple-amber"
                      : "text-chalk-dim",
                  )}
                >
                  {item.label}
                </Link>
              ))}
            </motion.nav>
          </>
        )}
      </AnimatePresence>
    </div>
  );
}
