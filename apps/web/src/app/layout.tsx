import type { Metadata } from "next";
import { Bricolage_Grotesque, Inter_Tight, JetBrains_Mono } from "next/font/google";
import { ClerkProvider } from "@clerk/nextjs";
import { SiteHeader } from "@/components/site-header";
import { LiveScoresProvider } from "@/components/live-scores";
import "./globals.css";

const display = Bricolage_Grotesque({
  variable: "--font-bricolage",
  subsets: ["latin"],
  weight: ["600", "700", "800"],
});

const sans = Inter_Tight({
  variable: "--font-inter-tight",
  subsets: ["latin"],
});

const mono = JetBrains_Mono({
  variable: "--font-jetbrains",
  subsets: ["latin"],
  weight: ["400", "500", "700"],
});

export const metadata: Metadata = {
  title: {
    default: "Shelf — rate board games in one tap",
    template: "%s · Shelf",
  },
  description:
    "A board game collection you actually enjoy using. Rate a game in one tap, keep your shelf, and write about what you played.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <ClerkProvider>
      <html
        lang="en"
        className={`${display.variable} ${sans.variable} ${mono.variable} h-full`}
      >
        <body className="relative min-h-full flex flex-col">
          <LiveScoresProvider>
            <SiteHeader />
            <main className="relative z-10 flex-1">{children}</main>
          </LiveScoresProvider>
          <footer className="relative z-10 border-t border-rule-soft mt-24">
            <div className="shell py-8 flex flex-wrap gap-x-6 gap-y-2 items-baseline justify-between text-sm text-chalk-faint">
              <p className="font-mono text-xs uppercase tracking-[0.18em]">
                Shelf
              </p>
              <p>
                Game data from{" "}
                <a
                  href="https://boardgamegeek.com"
                  className="text-chalk-dim underline underline-offset-3 hover:text-meeple-amber"
                  target="_blank"
                  rel="noreferrer noopener"
                >
                  BoardGameGeek
                </a>
                . Ratings and writing are our own.
              </p>
            </div>
          </footer>
        </body>
      </html>
    </ClerkProvider>
  );
}
