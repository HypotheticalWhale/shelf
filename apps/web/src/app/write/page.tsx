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
    <div className="mx-auto max-w-3xl px-5 py-9">
      <div className="flex items-baseline justify-between gap-4 flex-wrap">
        <h1 className="font-display font-800 text-3xl tracking-[-0.03em]">
          Write it down.
        </h1>
        <p className="text-sm text-chalk-faint">
          A review, a session report, an argument.
        </p>
      </div>

      <Composer defaultGameSlug={Array.isArray(game) ? game[0] : game} />
    </div>
  );
}
