-- Announces score changes so connected clients can update without polling.
--
-- The stats trigger already keeps game_stats exact on every rating write; this
-- rides along and publishes the new figures on a Postgres channel. NOTIFY is
-- transactional — subscribers only ever see committed numbers — and costs
-- nothing when nobody is listening.
--
-- The payload carries everything a client needs to redraw a score, so a fan-out
-- of updates never turns into a fan-out of queries.

CREATE OR REPLACE FUNCTION announce_game_stats() RETURNS trigger AS $$
DECLARE
    slug_text text;
    prior     record;
BEGIN
    SELECT g.slug INTO slug_text FROM games g WHERE g.id = NEW.game_id;
    IF slug_text IS NULL THEN
        RETURN NULL;
    END IF;

    SELECT mean_rating, prior_weight INTO prior FROM global_stats WHERE id = true;

    -- 8000 bytes is the NOTIFY payload ceiling; this is far below it.
    PERFORM pg_notify('shelf_game_stats', json_build_object(
        'slug',        slug_text,
        'numRatings',  NEW.num_ratings,
        'ratingSum',   NEW.rating_sum,
        'meanRating',  prior.mean_rating,
        'priorWeight', prior.prior_weight
    )::text);

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS game_stats_announce_trigger ON game_stats;
CREATE TRIGGER game_stats_announce_trigger
    AFTER INSERT OR UPDATE ON game_stats
    FOR EACH ROW EXECUTE FUNCTION announce_game_stats();
