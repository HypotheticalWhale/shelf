import { notFound } from "next/navigation";
import Link from "next/link";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { apiGetOrNull } from "@/lib/api";
import { formatDate } from "@/lib/format";
import type { Post } from "@/lib/types";

export async function generateMetadata({ params }: PageProps<"/u/[username]/[slug]">) {
  const { username, slug } = await params;
  const post = await apiGetOrNull<Post>(`/users/${username}/posts/${slug}`, {
    revalidate: 300,
  });
  return { title: post?.title ?? "Post" };
}

export default async function PostPage({ params }: PageProps<"/u/[username]/[slug]">) {
  const { username, slug } = await params;

  const post = await apiGetOrNull<Post>(`/users/${username}/posts/${slug}`, {
    authenticated: true,
  });
  if (!post) notFound();

  return (
    <article className="mx-auto max-w-2xl px-5 py-14">
      {post.game && (
        <Link
          href={`/games/${post.game.slug}`}
          className="font-mono text-[11px] uppercase tracking-[0.18em] text-meeple-teal hover:underline"
        >
          {post.game.name}
        </Link>
      )}

      <h1 className="mt-3 font-display font-800 text-[clamp(2rem,5vw,3rem)] leading-[1.02] tracking-[-0.035em]">
        {post.title}
      </h1>

      <p className="mt-4 font-mono text-xs text-chalk-faint">
        {post.author && (
          <Link href={`/u/${post.author.username}`} className="hover:text-chalk-dim">
            {post.author.displayName || post.author.username}
          </Link>
        )}
        {" · "}
        {formatDate(post.publishedAt ?? post.createdAt)}
        {!post.publishedAt && " · draft"}
      </p>

      <div className="prose-shelf mt-9">
        <ReactMarkdown remarkPlugins={[remarkGfm]}>{post.bodyMd}</ReactMarkdown>
      </div>
    </article>
  );
}
