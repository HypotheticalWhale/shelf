-- Where a game's cover came from.
--
-- Artwork is aggregated from several places and re-hosted on our own storage,
-- so each row has to record its provenance: what to credit, what licence it
-- carries, and which source won. Without that the catalogue accumulates images
-- nobody can account for.
ALTER TABLE games
    ADD COLUMN IF NOT EXISTS image_source     text,
    ADD COLUMN IF NOT EXISTS image_license    text,
    ADD COLUMN IF NOT EXISTS image_credit     text,
    ADD COLUMN IF NOT EXISTS image_origin     text,
    ADD COLUMN IF NOT EXISTS image_checked_at timestamptz;

-- Lets the pipeline resume: pick up whatever has not been looked at yet.
CREATE INDEX IF NOT EXISTS games_image_checked_idx
    ON games (image_checked_at NULLS FIRST);
