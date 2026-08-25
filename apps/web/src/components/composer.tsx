"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { apiRequest } from "@/lib/client";
import { GamePicker } from "./game-picker";
import { cn } from "@/lib/format";
import type { Post } from "@/lib/types";

/**
 * The post composer.
 *
 * Write and preview sit behind one toggle rather than a split pane, so the
 * writing column stays wide enough to read on a laptop. Saving a draft and
 * publishing are separate buttons because they are separate intentions.
 */
export function Composer({ defaultGameSlug }: { defaultGameSlug?: string }) {
  const router = useRouter();
  const { getToken } = useAuth();

  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [gameSlug, setGameSlug] = useState<string | null>(defaultGameSlug ?? null);
  const [tab, setTab] = useState<"write" | "preview">("write");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async (publish: boolean) => {
    if (!title.trim()) {
      setError("Give the post a title first.");
      return;
    }
    setBusy(true);
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
      setBusy(false);
    }
  };

  return (
    <div className="mt-8">
      <input
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        placeholder="Title"
        aria-label="Post title"
        className="w-full bg-transparent font-display font-800 text-3xl tracking-[-0.03em] placeholder:text-chalk-faint/60 outline-none border-b border-rule-soft focus:border-rule pb-3 transition-colors"
      />

      <div className="mt-5 flex items-center gap-3 text-sm">
        <span className="font-mono text-[11px] uppercase tracking-[0.18em] text-chalk-faint shrink-0">
          About
        </span>
        <GamePicker defaultSlug={defaultGameSlug} onChange={setGameSlug} />
      </div>

      <div className="mt-6 flex gap-1 border-b border-rule-soft">
        {(["write", "preview"] as const).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            className={cn(
              "px-3 py-2 text-sm capitalize border-b-2 -mb-px transition-colors",
              tab === t
                ? "border-meeple-amber text-chalk"
                : "border-transparent text-chalk-faint hover:text-chalk-dim",
            )}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === "write" ? (
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder="How did it play?"
          aria-label="Post body"
          rows={18}
          className="mt-4 w-full bg-transparent outline-none resize-y leading-relaxed placeholder:text-chalk-faint/60"
        />
      ) : (
        <div className="prose-shelf mt-5 min-h-[18rem]">
          {body.trim() ? (
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{body}</ReactMarkdown>
          ) : (
            <p className="text-chalk-faint">Nothing to preview yet.</p>
          )}
        </div>
      )}

      {error && <p className="mt-4 text-sm text-meeple-red">{error}</p>}

      <div className="mt-6 flex gap-3">
        <button
          type="button"
          disabled={busy}
          onClick={() => void submit(true)}
          className="bg-chalk text-felt-950 font-medium px-5 py-2 rounded-full hover:bg-meeple-amber transition-colors disabled:opacity-60"
        >
          {busy ? "Saving…" : "Publish"}
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={() => void submit(false)}
          className="border border-rule-soft px-5 py-2 rounded-full text-chalk-dim hover:text-chalk hover:border-rule transition-colors disabled:opacity-60"
        >
          Save draft
        </button>
      </div>
    </div>
  );
}
