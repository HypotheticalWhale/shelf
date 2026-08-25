import { apiGet } from "@/lib/api";
import { PostRow } from "@/components/post-row";
import type { Post } from "@/lib/types";

export const metadata = { title: "Reading" };

export default async function PostsPage() {
  const { posts } = await apiGet<{ posts: Post[] }>("/posts?limit=40");

  return (
    <div className="mx-auto max-w-3xl px-5 py-12">
      <h1 className="font-display font-800 text-4xl tracking-[-0.03em]">Reading</h1>
      <p className="mt-2 text-chalk-dim">
        Reviews, session reports and arguments from across the shelves.
      </p>

      {posts.length === 0 ? (
        <p className="mt-10 text-sm text-chalk-faint">
          Nothing published yet. Yours could be the first.
        </p>
      ) : (
        <div className="mt-8 divide-y divide-rule-soft border-y border-rule-soft">
          {posts.map((post) => (
            <PostRow key={post.id} post={post} />
          ))}
        </div>
      )}
    </div>
  );
}
