"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import {
  motion,
  useReducedMotion,
  useScroll,
  useSpring,
  useTransform,
} from "motion/react";
import { GameCover } from "./game-cover";
import { RatingPips } from "./rating-pips";
import { useServerState } from "@/lib/use-server-state";
import { useLiveScore } from "./live-scores";
import { formatYear, playerRange, scoreColor } from "@/lib/format";
import type { Game } from "@/lib/types";

/**
 * The landing experience: a tabletop that the page dives into.
 *
 * Scroll drives a single camera move. The table starts almost flat and
 * face-on, tilts and rushes toward the viewer, then dissolves as the catalogue
 * rises through it — one continuous shot rather than a sequence of reveals.
 *
 *   0.00 ──────── 0.35        0.35 ──────── 0.65        0.65 ──────── 1.00
 *        tabletop                camera dive                catalogue
 *
 * The pieces on the table are illustrations, not data: they are what a game in
 * progress looks like. The catalogue below is real — actual games, covers and
 * rating controls.
 */
export function LandingExperience({ games }: { games: Game[] }) {
  const container = useRef<HTMLDivElement>(null);
  const stage = useRef<HTMLDivElement>(null);
  const reduced = useReducedMotion();

  // How far the tabletop has to shrink to fit the space it is given.
  //
  // The scene is drawn at a fixed 850x380 and cropped on anything narrower, but
  // CSS cannot express the fit: `calc(100vw / 880)` divides a length by a number
  // and yields a length, which `scale` rejects. Stepped breakpoints work but
  // shrink the table on laptop windows that had room for it. Measuring gives a
  // continuous fit that is exactly 1 whenever the scene already fits.
  const [fit, setFit] = useState(1);

  useEffect(() => {
    const el = stage.current;
    if (!el) return;
    const measure = () => {
      const w = el.clientWidth;
      const h = el.clientHeight;
      // Leave a margin so the table never touches the edges.
      setFit(Math.min(1, (w - 32) / 880, (h * 0.62) / 380));
    };
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const { scrollYProgress } = useScroll({
    // "end end", not "end start": the stage is pinned only until the container's
    // bottom meets the viewport's bottom, which is one viewport short of the
    // container's full height. Measuring progress to "end start" stretched the
    // animation over a third more scroll than the pin actually lasts, so the
    // catalogue reached full opacity at the very moment the stage unpinned and
    // then slid off the top before it could be read.
    target: container,
    offset: ["start start", "end end"],
  });

  // Tracks the wheel closely rather than trailing it. The original spring was
  // soft enough that the table kept moving after scrolling stopped, which read
  // as lag rather than smoothing.
  const smoothProgress = useSpring(scrollYProgress, {
    stiffness: 220,
    damping: 40,
    restDelta: 0.0005,
  });

  const tableScale = useTransform(
    smoothProgress,
    [0, 0.35, 0.65],
    [1, 1.15, 4.5],
  );
  const tableY = useTransform(smoothProgress, [0, 0.35, 0.65], [0, -40, 180]);
  const tableRotateX = useTransform(
    smoothProgress,
    [0, 0.35, 0.65],
    [8, 12, 72],
  );
  const tableOpacity = useTransform(smoothProgress, [0, 0.48, 0.64], [1, 1, 0]);
  const heroOpacity = useTransform(smoothProgress, [0, 0.25, 0.45], [1, 1, 0]);
  const catalogueY = useTransform(smoothProgress, [0.4, 0.7, 1], [250, 0, 0]);
  const catalogueOpacity = useTransform(smoothProgress, [0.42, 0.62], [0, 1]);

  // A layer faded to zero is still in the document: without this the catalogue
  // cards sat invisibly over the hero and swallowed clicks meant for the page
  // behind them, and the hero kept taking clicks once it had faded out.
  const cataloguePointer = useTransform(catalogueOpacity, (o) =>
    o > 0.5 ? "auto" : "none",
  );
  const heroPointer = useTransform(heroOpacity, (o) =>
    o > 0.5 ? "auto" : "none",
  );

  // One row only: the panel is pinned to the viewport, so a second row is
  // clipped by its lower edge rather than scrolling into view.
  const shown = games.slice(0, 4);

  return (
    <div ref={container} className="relative h-[300vh] bg-[#15110d]">
      <div className="sticky top-16 h-[calc(100vh-4rem)] overflow-hidden">
        {/* Atmosphere */}
        <div className="absolute inset-0">
          <div className="absolute left-1/2 top-[35%] h-[800px] w-[800px] -translate-x-1/2 rounded-full bg-amber-500/[0.08] blur-[160px]" />
          <div className="absolute inset-0 opacity-[0.035] [background-image:radial-gradient(#fff_1px,transparent_1px)] [background-size:24px_24px]" />
        </div>

        {/* Hero copy */}
        <motion.div
          style={{
            opacity: reduced ? 1 : heroOpacity,
            pointerEvents: reduced ? "auto" : heroPointer,
            willChange: "opacity",
          }}
          className="absolute left-1/2 top-[6%] z-20 w-full -translate-x-1/2 px-5 text-center sm:top-[7%] sm:px-6"
        >
          <div className="mx-auto mb-5 w-fit rounded-full border border-amber-300/20 bg-amber-300/5 px-4 py-2 text-xs font-bold uppercase tracking-[0.2em] text-amber-200">
            The tabletop starts here
          </div>

          <h1 className="font-display text-4xl font-black tracking-[-0.05em] text-white sm:text-6xl lg:text-7xl">
            Find games.
            <br />
            <span className="text-amber-300">Find your people.</span>
          </h1>

          <p className="mx-auto mt-5 max-w-lg text-sm leading-6 text-white/45 sm:text-base sm:leading-7">
            Rate a game in one tap, keep the shelf you actually own, and write
            about what you played.
          </p>
        </motion.div>

        {/*
          The dive only reads as depth with a perspective ancestor. Without one,
          rotateX flattens the table instead of tipping it away from the camera.
        */}
        <div
          ref={stage}
          className="absolute inset-0"
          style={{
            perspective: "1200px",
            perspectiveOrigin: "50% 40%",
            // The scene is drawn at a fixed 850x380, so it crops on anything
            // narrower. The fit is measured rather than expressed in CSS —
            // calc() cannot turn viewport units into the unitless ratio `scale`
            // needs — and it sits on this wrapper rather than on the table
            // itself, because motion writes `scale` inline there and an inline
            // style wins. Here the two transforms compose.
            //
            // A string, not a number: React appends "px" to unknown numeric
            // style values, and `scale: 0.41px` is invalid.
            scale: String(fit),
          }}
        >
          <motion.div
            style={{
              scale: reduced ? 1 : tableScale,
              y: reduced ? 0 : tableY,
              rotateX: reduced ? 8 : tableRotateX,
              opacity: reduced ? 1 : tableOpacity,
              transformStyle: "preserve-3d",
              willChange: "transform, opacity",
            }}
            className="absolute left-1/2 top-[68%] h-[380px] w-[850px] -translate-x-1/2 -translate-y-1/2"
          >
            <div className="absolute -bottom-20 left-[8%] h-24 w-[84%] rounded-full bg-black/60 blur-xl" />

            {/* Wooden frame */}
            <div className="absolute inset-0 rounded-[30px] border-[15px] border-[#704426] bg-[#a86c3d] shadow-[0_50px_120px_rgba(0,0,0,.65)]">
              {/* Felt */}
              <div className="absolute inset-3 overflow-hidden rounded-[17px] bg-[#29473d]">
                <div className="absolute inset-0 opacity-20 [background-image:radial-gradient(#fff_.7px,transparent_.7px)] [background-size:8px_8px]" />

                {/* Board */}
                <div className="absolute left-1/2 top-1/2 h-[230px] w-[600px] -translate-x-1/2 -translate-y-1/2 rotate-[-3deg] rounded-2xl border border-amber-100/10 bg-[#aa804a]/25 shadow-inner">
                  <div className="absolute inset-5 rounded-xl border border-dashed border-amber-100/15" />

                  {Array.from({ length: 11 }).map((_, i) => {
                    const angle = (i / 10) * Math.PI;
                    const x = 50 + Math.cos(angle) * 43;
                    const y = 50 + Math.sin(angle) * 34;
                    return (
                      <div
                        key={i}
                        className="absolute h-7 w-7 -translate-x-1/2 -translate-y-1/2 rounded-full border border-amber-100/20 bg-amber-100/10"
                        style={{ left: `${x}%`, top: `${y}%` }}
                      />
                    );
                  })}
                </div>

                <TableCard
                  title="ROOT"
                  x="-260px"
                  y="-80px"
                  rotate={-14}
                  color="#b9d68b"
                  delay={0}
                  reduced={reduced}
                />
                <TableCard
                  title="DUNE"
                  x="280px"
                  y="-70px"
                  rotate={13}
                  color="#d6ad77"
                  delay={0.1}
                  reduced={reduced}
                />
                <TableCard
                  title="WINGSPAN"
                  x="300px"
                  y="80px"
                  rotate={-8}
                  color="#a8c5d0"
                  delay={0.2}
                  reduced={reduced}
                />

                <Dice x="220px" y="-85px" reduced={reduced} />
                <Dice x="-220px" y="95px" red reduced={reduced} />

                <Meeple x="-90px" y="70px" color="#ef4444" reduced={reduced} />
                <Meeple x="60px" y="35px" color="#3b82f6" reduced={reduced} />
                <Meeple x="160px" y="75px" color="#eab308" reduced={reduced} />

                {/* Box in play */}
                <motion.div
                  initial={reduced ? false : { scale: 0 }}
                  animate={{ scale: 1 }}
                  transition={{ delay: 0.8, type: "spring" }}
                  className="absolute left-1/2 top-1/2 h-[155px] w-[110px] -translate-x-1/2 -translate-y-1/2 rotate-2 rounded-xl bg-[#e8d4a8] p-2 shadow-2xl"
                >
                  <div className="h-full rounded-lg border border-black/10 p-2">
                    <div className="h-[65px] rounded-md bg-[#66543b]" />
                    <div className="mt-3 h-2 w-4/5 rounded bg-black/20" />
                    <div className="mt-2 h-2 w-3/5 rounded bg-black/10" />
                    <div className="mt-6 flex justify-between text-[8px] font-black text-[#352719]">
                      <span>STRATEGY</span>
                      <span>8.7</span>
                    </div>
                  </div>
                </motion.div>
              </div>
            </div>
          </motion.div>
        </div>

        {/* Catalogue — real games */}
        <motion.section
          style={{
            opacity: reduced ? 1 : catalogueOpacity,
            y: reduced ? 0 : catalogueY,
            pointerEvents: reduced ? "auto" : cataloguePointer,
            willChange: "transform, opacity",
          }}
          className="absolute inset-0 z-20 flex items-center"
        >
          <div className="shell w-full">
            <div className="mb-6 flex items-end justify-between gap-4 sm:mb-8">
              <div>
                <p className="mb-3 text-xs font-bold uppercase tracking-[0.2em] text-amber-300">
                  Explore
                </p>
                <h2 className="font-display text-3xl font-black tracking-tight text-white sm:text-6xl">
                  What&rsquo;s on the table?
                </h2>
              </div>

              <Link
                href="/games"
                className="hidden shrink-0 rounded-full border border-white/10 bg-white/5 px-5 py-2 text-sm text-white/60 transition hover:bg-white/10 md:block"
              >
                Browse all 31,000 →
              </Link>
            </div>

            <Link
              href="/games"
              className="mb-5 flex items-center rounded-2xl sm:mb-8 border border-white/10 bg-[#221b14] px-5 py-4 transition hover:bg-[#2b231a]"
            >
              <span className="mr-3 text-white/30">⌕</span>
              <span className="text-sm text-white/30">
                Search games, designers, mechanics…
              </span>
            </Link>

            <div className="grid grid-cols-2 gap-3 sm:gap-4 md:grid-cols-4 [&>*:nth-child(n+3)]:hidden md:[&>*:nth-child(n+3)]:block">
              {shown.map((game, i) => (
                <LandingGameCard
                  key={game.slug}
                  game={game}
                  index={i}
                  reduced={reduced}
                />
              ))}
            </div>
          </div>
        </motion.section>
      </div>
    </div>
  );
}

/**
 * A game on the landing catalogue.
 *
 * Holds its own copy of the game so a rating updates the card in place. Without
 * that the pips filled but nothing else moved — no score appeared, no count
 * changed — so rating here looked like it had done nothing at all.
 */
function LandingGameCard({
  game: initial,
  index,
  reduced,
}: {
  game: Game;
  index: number;
  reduced: boolean | null;
}) {
  const [game, setGame] = useServerState(initial);
  const live = useLiveScore(game.slug);
  const shown = live ? { ...game, ...live } : game;

  return (
    <motion.div
      initial={reduced ? false : { opacity: 0, y: 80 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ delay: index * 0.08, type: "spring", stiffness: 100 }}
      className="group overflow-hidden rounded-2xl border border-white/10 bg-white/[0.04] transition hover:-translate-y-2 hover:bg-white/[0.07]"
    >
      <Link href={`/games/${game.slug}`} className="block">
        <div className="relative h-[clamp(170px,30vh,300px)] w-full overflow-hidden">
          <GameCover
            name={game.name}
            slug={game.slug}
            src={game.imageUrl ?? game.thumbnailUrl}
            className="absolute inset-0 size-full"
            full
          />
        </div>
      </Link>

      <div className="p-4">
        <Link href={`/games/${game.slug}`}>
          <h3 className="truncate font-bold text-white transition-colors group-hover:text-amber-300">
            {game.name}
          </h3>
        </Link>

        <div className="mt-2 flex items-baseline justify-between gap-2 text-xs text-white/35">
          <span className="truncate">
            {playerRange(game.minPlayers, game.maxPlayers) ?? "—"}
          </span>
          <span>{formatYear(game.yearPublished) ?? ""}</span>
        </div>

        {/* The score, so a rating visibly lands. */}
        <p className="mt-2 font-mono text-sm">
          {shown.numRatings > 0 ? (
            <>
              <span
                style={{ color: scoreColor(shown.score) }}
                className="font-bold"
              >
                {shown.score.toFixed(1)}
              </span>
              <span className="ml-1.5 text-[11px] text-white/30">
                {shown.numRatings}{" "}
                {shown.numRatings === 1 ? "rating" : "ratings"}
              </span>
            </>
          ) : (
            <span className="text-[11px] text-white/30">not rated yet</span>
          )}
        </p>

        <RatingPips
          slug={game.slug}
          viewerRating={game.viewerRating}
          onRated={setGame}
          className="mt-2.5"
        />
      </div>
    </motion.div>
  );
}

function TableCard({
  title,
  x,
  y,
  rotate,
  color,
  delay,
  reduced,
}: {
  title: string;
  x: string;
  y: string;
  rotate: number;
  color: string;
  delay: number;
  reduced: boolean | null;
}) {
  return (
    <motion.div
      initial={
        reduced
          ? false
          : {
              opacity: 0,
              scale: 0.5,
              x: "calc(-50% + 0px)",
              y: "calc(-50% + 0px)",
              rotate: rotate * 2,
            }
      }
      animate={{
        opacity: 1,
        scale: 1,
        x: `calc(-50% + ${x})`,
        y: `calc(-50% + ${y})`,
        rotate,
      }}
      transition={{
        delay,
        duration: 0.9,
        type: "spring",
        stiffness: 90,
        damping: 13,
      }}
      className="absolute left-1/2 top-1/2 h-[125px] w-[85px] rounded-xl p-1.5 shadow-[0_15px_20px_rgba(0,0,0,.4)]"
      style={{ backgroundColor: color }}
    >
      <div className="flex h-full flex-col rounded-lg border border-black/10 p-2 text-[#30251b]">
        <div className="h-12 rounded-md bg-black/15" />
        <div className="mt-2 text-[8px] font-black">{title}</div>
        <div className="mt-1 h-1.5 w-4/5 rounded bg-black/10" />
        <div className="mt-auto text-[7px] opacity-40">BOARD GAME</div>
      </div>
    </motion.div>
  );
}

function Dice({
  x,
  y,
  red = false,
  reduced,
}: {
  x: string;
  y: string;
  red?: boolean;
  reduced: boolean | null;
}) {
  return (
    <motion.div
      initial={
        reduced
          ? false
          : {
              opacity: 0,
              x: "calc(-50% + 400px)",
              y: "calc(-50% - 250px)",
              rotate: 220,
            }
      }
      animate={{
        opacity: 1,
        x: `calc(-50% + ${x})`,
        y: `calc(-50% + ${y})`,
        rotate: red ? -22 : 32,
      }}
      transition={{
        delay: 0.9,
        duration: 1,
        type: "spring",
        stiffness: 75,
        damping: 11,
      }}
      className={`absolute left-1/2 top-1/2 grid h-12 w-12 place-items-center rounded-xl shadow-[0_10px_15px_rgba(0,0,0,.4)] ${
        red ? "bg-red-400" : "bg-white"
      }`}
    >
      <span
        className={`text-lg font-black ${red ? "text-red-950" : "text-black/70"}`}
      >
        {red ? "6" : "5"}
      </span>
    </motion.div>
  );
}

function Meeple({
  x,
  y,
  color,
  reduced,
}: {
  x: string;
  y: string;
  color: string;
  reduced: boolean | null;
}) {
  return (
    <motion.div
      initial={reduced ? false : { opacity: 0, scale: 0 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ delay: 1, type: "spring", stiffness: 180, damping: 12 }}
      className="absolute left-1/2 top-1/2"
      style={{ x, y }}
    >
      <div className="relative h-[34px] w-[26px] -translate-x-1/2 -translate-y-1/2">
        {/* head */}
        <div
          className="absolute left-1/2 top-0 h-[15px] w-[15px] -translate-x-1/2 rounded-full shadow-md"
          style={{ backgroundColor: color }}
        />
        {/* body */}
        <div
          className="absolute bottom-0 left-1/2 h-[21px] w-[20px] -translate-x-1/2 rounded-t-[10px] rounded-b-[3px] shadow-lg"
          style={{ backgroundColor: color }}
        />
        {/* arms */}
        <div
          className="absolute -left-[3px] top-[15px] h-[10px] w-[8px] -rotate-[18deg] rounded-full"
          style={{ backgroundColor: color }}
        />
        <div
          className="absolute -right-[3px] top-[15px] h-[10px] w-[8px] rotate-[18deg] rounded-full"
          style={{ backgroundColor: color }}
        />
      </div>
    </motion.div>
  );
}
