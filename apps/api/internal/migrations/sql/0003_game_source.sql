-- Tracks where a game row came from.
--
-- Shelf ships a small hand-curated catalogue so the site is usable before a
-- BoardGameGeek API token is issued. Marking those rows lets them be replaced
-- wholesale once real imports are running, without touching imported games.
ALTER TABLE games
    ADD COLUMN IF NOT EXISTS source text NOT NULL DEFAULT 'bgg'
        CHECK (source IN ('bgg', 'seed'));

CREATE INDEX IF NOT EXISTS games_source_idx ON games (source);
