import { clerkMiddleware } from "@clerk/nextjs/server";

// Next 16 renamed this convention from middleware.ts to proxy.ts.
// Sessions are attached everywhere so pages can greet a signed-in reader, but
// nothing is force-protected here: the Go API is the authority on what a given
// user may write, and duplicating those rules in the edge layer would mean two
// places to keep in sync.
export default clerkMiddleware();

export const config = {
  matcher: [
    "/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)",
    "/(api|trpc)(.*)",
  ],
};
