import { auth } from "@clerk/nextjs/server";
import { redirect } from "next/navigation";
import { Composer } from "@/components/composer";

export const metadata = { title: "Write" };

export default async function WritePage({ searchParams }: PageProps<"/write">) {
  const { userId } = await auth();
  if (!userId) redirect("/sign-in?redirect_url=/write");

  const params = await searchParams;
  const game = params.game;

  return (
    <div className="mx-auto max-w-3xl px-5 py-12">
      <p className="font-mono text-[11px] uppercase tracking-[0.22em] text-meeple-teal">
        New post
      </p>
      <h1 className="mt-3 font-display font-800 text-4xl tracking-[-0.03em]">
        Write it down.
      </h1>
      <p className="mt-2 text-chalk-dim max-w-md">
        A review, a session report, an argument. Save a draft while you think it
        over, publish when it is ready.
      </p>

      <Composer defaultGameSlug={Array.isArray(game) ? game[0] : game} />
    </div>
  );
}
