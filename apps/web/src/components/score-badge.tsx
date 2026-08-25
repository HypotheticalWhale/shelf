"use client";

import { useEffect } from "react";
import { motion, useMotionValue, useSpring, useTransform } from "motion/react";
import { scoreColor } from "@/lib/format";

type Props = {
  score: number;
  numRatings: number;
  size?: "sm" | "lg";
};

/**
 * The aggregate score, animated.
 *
 * The number springs to its new value rather than snapping, which is what makes
 * a rating feel like it landed somewhere — you watch your vote move the score.
 */
export function ScoreBadge({ score, numRatings, size = "sm" }: Props) {
  const value = useMotionValue(score);
  const spring = useSpring(value, { stiffness: 140, damping: 18, mass: 0.6 });
  const text = useTransform(spring, (v) => v.toFixed(1));

  useEffect(() => {
    value.set(score);
  }, [score, value]);

  const large = size === "lg";

  // An unrated game technically scores exactly the global mean, but printing
  // that number on every card reads as real data and makes the whole
  // catalogue look identically rated. Say plainly that nobody has voted yet.
  if (numRatings === 0) {
    return (
      <div className="flex items-baseline gap-2">
        <span
          className={
            large
              ? "font-mono font-bold text-5xl leading-none text-chalk-faint/50"
              : "font-mono font-bold text-lg leading-none text-chalk-faint/50"
          }
          aria-hidden
        >
          –
        </span>
        <span className={large ? "text-sm text-chalk-faint" : "text-[11px] text-chalk-faint"}>
          not rated yet
        </span>
      </div>
    );
  }

  return (
    <div className="flex items-baseline gap-2">
      <motion.span
        className={
          large
            ? "font-mono font-bold text-5xl tabular-nums leading-none"
            : "font-mono font-bold text-lg tabular-nums leading-none"
        }
        style={{ color: scoreColor(score) }}
      >
        {text}
      </motion.span>
      <span
        className={large ? "text-sm text-chalk-faint" : "text-[11px] text-chalk-faint"}
      >
        {numRatings} {numRatings === 1 ? "rating" : "ratings"}
      </span>
    </div>
  );
}
