"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

export type LiveScore = {
  numRatings: number;
  score: number;
  mean: number;
};

const LiveScoresContext = createContext<Map<string, LiveScore>>(new Map());

/**
 * Keeps every open page in step with the live catalogue.
 *
 * The API streams a score whenever one changes, sourced from a Postgres NOTIFY
 * on the same trigger that keeps the aggregates exact — so an update is pushed
 * the moment a rating commits, and nothing is polled while the site is quiet.
 *
 * One connection serves the whole page: components read from the shared map
 * rather than each opening a stream of their own.
 */
export function LiveScoresProvider({ children }: { children: React.ReactNode }) {
  const [scores, setScores] = useState<Map<string, LiveScore>>(new Map());
  const sourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    // Reconnection is EventSource's job — it honours the retry interval the
    // server sends, which also covers a serverless instance being recycled.
    const source = new EventSource("/api/events");
    sourceRef.current = source;

    source.addEventListener("stats", (event) => {
      try {
        const data = JSON.parse((event as MessageEvent).data) as
          | (LiveScore & { slug: string })
          | null;
        if (!data?.slug) return;
        setScores((prev) => {
          const next = new Map(prev);
          next.set(data.slug, {
            numRatings: data.numRatings,
            score: data.score,
            mean: data.mean,
          });
          return next;
        });
      } catch {
        // A malformed frame should never take the stream down.
      }
    });

    return () => {
      source.close();
      sourceRef.current = null;
    };
  }, []);

  return (
    <LiveScoresContext.Provider value={scores}>
      {children}
    </LiveScoresContext.Provider>
  );
}

/**
 * The latest pushed score for a game, or null if none has arrived.
 *
 * Only the aggregate figures come from the stream. A viewer's own rating is
 * never overwritten this way — that would let somebody else's vote appear to
 * change yours.
 */
export function useLiveScore(slug: string): LiveScore | null {
  const scores = useContext(LiveScoresContext);
  return useMemo(() => scores.get(slug) ?? null, [scores, slug]);
}
