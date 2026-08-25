"use client";

import { useCallback, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { apiRequest } from "@/lib/client";
import { cn } from "@/lib/format";
import { GamePicker } from "./game-picker";
import type { Post } from "@/lib/types";

/**
 * The post composer, built as a single writing sheet.
 *
 * The earlier version scattered a title field, a bare textarea and its buttons
 * down an empty page, which pushed Publish below the fold where nobody could
 * find it. Everything now lives on one contained surface — a sheet of paper on
 * the table — with the actions pinned to its foot so they are always reachable,
 * and the preview rendered through the same `prose-shelf` styles as a published
 * post, so what you see is what gets published.
 */
export function Composer({ defaultGameSlug }: { defaultGameSlug?: string }) {
  const router = useRouter();
  const { getToken } = useAuth();

  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [gameSlug, setGameSlug] = useState<string | null>(defaultGameSlug ?? null);
  const [tab, setTab] = useState<"write" | "preview">("write");
  const [busy, setBusy] = useState<"publish" | "draft" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const titleRef = useRef<HTMLTextAreaElement>(null);

  // The title is a textarea, not an input, so a long one wraps instead of
  // scrolling out of sight — and it grows to fit rather than scrolling inside
  // a fixed box.
  const fitTitle = useCallback((el: HTMLTextAreaElement | null) => {
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${el.scrollHeight}px`;
  }, []);

  const stats = useMemo(() => {
    const words = body.trim() ? body.trim().split(/\s+/).length : 0;
    // 220 wpm is a fair pace for considered prose rather than skimming.
    return { words, minutes: Math.max(1, Math.round(words / 220)) };
  }, [body]);

  const submit = async (publish: boolean) => {
    if (!title.trim()) {
      setError("Give the post a title first.");
      return;
    }
    setBusy(publish ? "publish" : "draft");
    setError(null);

    try {
      const token = await getToken();
      const post = await apiRequest<Post>("/posts", {
        method: "POST",
        body: JSON.stringify({
          title: title.trim(),
          bodyMd: body,
          gameSlug: gameSlug ?? undefined,
          publish,
        }),
        token,
      });
      const me = await apiRequest<{ username: string }>("/me", { token });
      router.push(`/u/${me.username}/${post.slug}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "";
      setError(
        message === "not found"
          ? "That game is no longer in the catalogue. Clear it and try again."
          : message || "Could not save the post",
      );
      setBusy(null);
    }
  };

  // overflow-clip rather than overflow-hidden: `hidden` turns the sheet into a
  // scroll container, which silently disables `position: sticky` on the action
  // bar and drops Publish below the fold again.
  return (
    <div className="mt-8 rounded-2xl border border-rule bg-felt-900/80 shadow-2xl shadow-felt-950/50 overflow-clip">
      {/* Title */}
      <div className="px-6 sm:px-9 pt-8 pb-5">
        <textarea
          ref={(el) => {
            titleRef.current = el;
            fitTitle(el);
          }}
          value={title}
          onChange={(e) => {
            setTitle(e.target.value);
            fitTitle(e.target);
          }}
          onKeyDown={(e) => {
            // Enter belongs to the body, not the title.
            if (e.key === "Enter") e.preventDefault();
          }}
          placeholder="What did you play?"
          aria-label="Post title"
          maxLength={140}
          rows={1}
          className="w-full bg-transparent font-display font-800 text-[clamp(1.5rem,3.2vw,2.15rem)] leading-[1.15] tracking-[-0.035em] placeholder:text-chalk-faint/40 outline-none resize-none overflow-hidden block"
        />
      </div>

      {/* Attachment band */}
      <div className="px-6 sm:px-9 py-3 border-y border-rule-soft bg-felt-800/40 flex items-center gap-3 flex-wrap">
        <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-chalk-faint shrink-0">
          About
        </span>
        <GamePicker defaultSlug={defaultGameSlug} onChange={setGameSlug} />
      </div>

      {/* Mode switch */}
      <div className="px-6 sm:px-9 pt-4">
        <div
          role="tablist"
          aria-label="Editor mode"
          className="inline-flex rounded-full bg-felt-800 border border-rule-soft p-0.5"
        >
          {(["write", "preview"] as const).map((t) => (
            <button
              key={t}
              role="tab"
              type="button"
              aria-selected={tab === t}
              onClick={() => setTab(t)}
              className={cn(
                "px-4 py-1.5 rounded-full text-xs capitalize transition-colors",
                tab === t
                  ? "bg-chalk text-felt-950 font-medium"
                  : "text-chalk-dim hover:text-chalk",
              )}
            >
              {t}
            </button>
          ))}
        </div>
      </div>

      {/* Surface */}
      <div className="px-6 sm:px-9 py-6 min-h-[26rem]">
        {tab === "write" ? (
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            placeholder={"How did it play?\n\nWhat worked, what dragged, who would you hand it to.\n\nMarkdown works: ## headings, **bold**, > quotes, - lists."}
            aria-label="Post body"
            rows={16}
            className="w-full max-w-[68ch] bg-transparent outline-none resize-none text-[1.0625rem] leading-[1.75] placeholder:text-chalk-faint/40 placeholder:leading-[1.9]"
          />
        ) : (
          <div className="prose-shelf max-w-[68ch]">
            {body.trim() ? (
              <ReactMarkdown remarkPlugins={[remarkGfm]}>{body}</ReactMarkdown>
            ) : (
              <p className="text-chalk-faint">
                Nothing to preview yet. Switch back and write something.
              </p>
            )}
          </div>
        )}
      </div>

      {/* Actions — pinned to the sheet so Publish never falls below the fold. */}
      <div className="sticky bottom-0 px-6 sm:px-9 py-4 border-t border-rule bg-felt-800/95 backdrop-blur flex items-center justify-between gap-4 flex-wrap">
        <p className="font-mono text-[11px] text-chalk-faint" aria-live="polite">
          {stats.words === 0
            ? "empty"
            : `${stats.words} ${stats.words === 1 ? "word" : "words"} · ${stats.minutes} min read`}
        </p>

        <div className="flex items-center gap-2.5">
          <button
            type="button"
            disabled={busy !== null}
            onClick={() => void submit(false)}
            className="px-4 py-2 rounded-full text-sm text-chalk-dim border border-rule-soft hover:text-chalk hover:border-rule transition-colors disabled:opacity-50"
          >
            {busy === "draft" ? "Saving…" : "Save draft"}
          </button>
          <button
            type="button"
            disabled={busy !== null}
            onClick={() => void submit(true)}
            className="px-5 py-2 rounded-full text-sm font-medium bg-chalk text-felt-950 hover:bg-meeple-amber transition-colors disabled:opacity-50"
          >
            {busy === "publish" ? "Publishing…" : "Publish"}
          </button>
        </div>
      </div>

      {error && (
        <p
          role="alert"
          className="px-6 sm:px-9 py-3 text-sm text-meeple-red border-t border-rule-soft bg-meeple-red/5"
        >
          {error}
        </p>
      )}
    </div>
  );
}
