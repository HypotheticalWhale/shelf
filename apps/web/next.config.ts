import type { NextConfig } from "next";

// Proxy the API through this app's own origin.
//
// The Go service runs as a separate deployment, but the browser only ever
// talks to one host. That removes CORS and its preflight round trip from every
// rating click — the interaction this whole product is built around.
const apiURL = process.env.SHELF_API_URL ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiURL}/:path*` }];
  },
  images: {
    // BGG serves cover art from its own CDN once an API token is configured.
    remotePatterns: [
      { protocol: "https", hostname: "cf.geekdo-images.com" },
      { protocol: "https", hostname: "img.clerk.com" },
    ],
  },
};

export default nextConfig;
