import Link from "next/link";
import { formatDate } from "@/lib/format";
import type { Post } from "@/lib/types";

/** One entry in a reading list. Excerpt is derived, so authors write only prose. */
export function PostRow({ post, showAuthor = true }: { post: Post; showAuthor?: boolean }) {
  const author = post.author;
  const excerpt = post.bodyMd
    .replace(/[#>*_`\[\]()]/g, "")
    .replace(/\s+/g, " ")
    .trim()
    .slice(0, 180);

  const href = author ? `/u/${author.username}/${post.slug}` : "#";

  return (
    <article className="py-5 group">
      <div className="flex items-baseline gap-3 flex-wrap">
        <Link href={href}>
          <h3 className="font-display font-700 text-lg tracking-[-0.02em] group-hover:text-meeple-amber transition-colors">
            {post.title}
          </h3>
        </Link>
        {post.game && (
          <Link
            href={`/games/${post.game.slug}`}
            className="font-mono text-[11px] uppercase tracking-wider text-meeple-teal hover:underline"
          >
            {post.game.name}
          </Link>
        )}
        {!post.publishedAt && (
          <span className="font-mono text-[10px] uppercase tracking-wider text-chalk-faint border border-rule px-1.5 py-0.5 rounded">
            draft
          </span>
        )}
      </div>

      <p className="mt-1.5 text-sm text-chalk-dim line-clamp-2 max-w-2xl">{excerpt}</p>

      <p className="mt-2 font-mono text-[11px] text-chalk-faint">
        {showAuthor && author && (
          <>
            <Link href={`/u/${author.username}`} className="hover:text-chalk-dim">
              {author.displayName || author.username}
            </Link>
            {" · "}
          </>
        )}
        {formatDate(post.publishedAt ?? post.createdAt)}
      </p>
    </article>
  );
}
