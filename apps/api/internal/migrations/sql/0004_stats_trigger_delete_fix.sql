-- Lets a game be deleted when it has ratings.
--
-- Deleting a game cascades to its ratings, and each of those deletes fires
-- ratings_stats_sync, which applied a -1 delta through apply_rating_delta.
-- That function upserts, so it re-inserted a game_stats row referencing the
-- game being deleted, and the foreign key rejected it — making any rated game
-- undeletable, including through `import -clear-seed`.
--
-- On DELETE the stats row is now only adjusted while the game still exists.
-- When the game is going away its game_stats row is cascading away too, so
-- there is nothing left to keep accurate.

CREATE OR REPLACE FUNCTION ratings_stats_sync() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM apply_rating_delta(NEW.game_id, 1, NEW.value);

    ELSIF TG_OP = 'UPDATE' THEN
        -- Re-rating the same game: the count is unchanged, only the sum moves.
        IF NEW.game_id <> OLD.game_id THEN
            PERFORM apply_rating_delta(OLD.game_id, -1, -OLD.value);
            PERFORM apply_rating_delta(NEW.game_id,  1,  NEW.value);
        ELSIF NEW.value <> OLD.value THEN
            PERFORM apply_rating_delta(NEW.game_id, 0, NEW.value - OLD.value);
        END IF;

    ELSIF TG_OP = 'DELETE' THEN
        IF EXISTS (SELECT 1 FROM games WHERE id = OLD.game_id) THEN
            PERFORM apply_rating_delta(OLD.game_id, -1, -OLD.value);
        END IF;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
