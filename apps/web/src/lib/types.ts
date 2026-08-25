// Mirrors the JSON the Go API returns. Kept hand-written and small so the
// contract between the two services is visible in one place.

export type User = {
  id: string;
  username: string;
  displayName: string | null;
  avatarUrl: string | null;
  bio: string | null;
  createdAt: string;
};

export type Game = {
  id: number;
  bggId: number;
  slug: string;
  name: string;
  yearPublished: number | null;
  description?: string | null;
  imageUrl: string | null;
  thumbnailUrl: string | null;
  minPlayers: number | null;
  maxPlayers: number | null;
  minPlaytime: number | null;
  maxPlaytime: number | null;
  weight: number | null;
  designers?: string[];
  categories?: string[];
  mechanics?: string[];
  numRatings: number;
  score: number;
  mean: number;
  viewerRating: number | null;
  viewerShelf?: string[];
};

export type GameDetail = Game & { histogram: number[] };

export type Prior = { meanRating: number; priorWeight: number };

export type GamePage = { games: Game[]; total: number; prior: Prior };

export type Rating = {
  gameId: number;
  value: number;
  updatedAt: string;
  game?: Game;
};

export type ShelfItem = { status: string; createdAt: string; game?: Game };

export type Post = {
  id: number;
  slug: string;
  title: string;
  bodyMd: string;
  publishedAt: string | null;
  createdAt: string;
  updatedAt: string;
  author?: User;
  game?: Game | null;
};

export type Profile = {
  user: User;
  recentRatings: Rating[];
  shelf: ShelfItem[];
  posts: Post[];
};

export type Collector = {
  user: User;
  ownedCount: number;
  ratedCount: number;
  postCount: number;
  avgRating: number;
  shelfPeek: Game[];
};
