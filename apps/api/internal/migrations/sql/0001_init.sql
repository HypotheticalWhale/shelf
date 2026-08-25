-- Shelf initial schema.
-- Ratings are 1..10 in half-steps, matching the scale board gamers already know.

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Users mirror Clerk identities. Clerk owns authentication; we own profile and
-- blog data. Rows are created just-in-time on the first authenticated request.
CREATE TABLE IF NOT EXISTS users (
    id           text PRIMARY KEY,
    username     citext UNIQUE NOT NULL,
    display_name text,
    avatar_url   text,
    bio          text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS games (
    id             bigserial PRIMARY KEY,
    bgg_id         integer UNIQUE NOT NULL,
    slug           text UNIQUE NOT NULL,
    name           text NOT NULL,
    year_published integer,
    description    text,
    image_url      text,
    thumbnail_url  text,
    min_players    integer,
    max_players    integer,
    min_playtime   integer,
    max_playtime   integer,
    weight         numeric(4,2),
    designers      text[] NOT NULL DEFAULT '{}',
    categories     text[] NOT NULL DEFAULT '{}',
    mechanics      text[] NOT NULL DEFAULT '{}',
    imported_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS games_name_trgm_idx ON games USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS games_year_idx      ON games (year_published DESC NULLS LAST);

CREATE TABLE IF NOT EXISTS ratings (
    user_id    text   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_id    bigint NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    value      numeric(3,1) NOT NULL CHECK (value >= 1 AND value <= 10),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, game_id)
);

CREATE INDEX IF NOT EXISTS ratings_game_idx        ON ratings (game_id);
CREATE INDEX IF NOT EXISTS ratings_user_recent_idx ON ratings (user_id, updated_at DESC);

-- Denormalised aggregates, kept exact by the trigger in 0002. Storing the sum
-- rather than the mean lets the trigger apply O(1) deltas instead of rescanning
-- every rating for a game on each write.
CREATE TABLE IF NOT EXISTS game_stats (
    game_id     bigint PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
    num_ratings integer NOT NULL DEFAULT 0,
    rating_sum  numeric NOT NULL DEFAULT 0,
    avg_rating  numeric GENERATED ALWAYS AS (
        CASE WHEN num_ratings > 0 THEN rating_sum / num_ratings END
    ) STORED,
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS game_stats_popular_idx ON game_stats (num_ratings DESC);

-- Single-row table holding the Bayesian prior. `mean_rating` is C (the global
-- mean) and `prior_weight` is m (how many "average" votes every game is seeded
-- with). BGG uses m=100, which is right at their volume and far too aggressive
-- for a new site — 5 lets real signal surface while still blocking a lone 10/10
-- from topping the chart.
CREATE TABLE IF NOT EXISTS global_stats (
    id           boolean PRIMARY KEY DEFAULT true CHECK (id),
    mean_rating  numeric NOT NULL DEFAULT 7.0,
    prior_weight integer NOT NULL DEFAULT 5,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

INSERT INTO global_stats (id) VALUES (true) ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS shelf_items (
    user_id    text   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_id    bigint NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    status     text   NOT NULL CHECK (status IN ('owned','wishlist','played')),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, game_id, status)
);

CREATE INDEX IF NOT EXISTS shelf_items_user_idx ON shelf_items (user_id, status);

-- The "personal blog": reviews and session reports, optionally tied to a game.
CREATE TABLE IF NOT EXISTS posts (
    id           bigserial PRIMARY KEY,
    user_id      text   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_id      bigint REFERENCES games(id) ON DELETE SET NULL,
    slug         text NOT NULL,
    title        text NOT NULL,
    body_md      text NOT NULL,
    published_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, slug)
);

CREATE INDEX IF NOT EXISTS posts_author_idx    ON posts (user_id, published_at DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS posts_game_idx      ON posts (game_id) WHERE game_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS posts_published_idx ON posts (published_at DESC) WHERE published_at IS NOT NULL;
