-- Keeps game_stats exact under concurrent writes.
--
-- Every rating change applies an O(1) delta rather than recomputing an
-- aggregate, and the INSERT ... ON CONFLICT DO UPDATE takes a row lock on the
-- game_stats row. Two people rating the same game at the same instant therefore
-- serialise on that row and both deltas land — no lost update, no drift, and no
-- periodic reconciliation job to keep honest.

CREATE OR REPLACE FUNCTION apply_rating_delta(
    p_game_id    bigint,
    p_count_diff integer,
    p_sum_diff   numeric
) RETURNS void AS $$
BEGIN
    INSERT INTO game_stats (game_id, num_ratings, rating_sum, updated_at)
    VALUES (p_game_id, GREATEST(p_count_diff, 0), GREATEST(p_sum_diff, 0), now())
    ON CONFLICT (game_id) DO UPDATE
        SET num_ratings = game_stats.num_ratings + p_count_diff,
            rating_sum  = game_stats.rating_sum  + p_sum_diff,
            updated_at  = now();
END;
$$ LANGUAGE plpgsql;

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
        PERFORM apply_rating_delta(OLD.game_id, -1, -OLD.value);
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS ratings_stats_sync_trigger ON ratings;
CREATE TRIGGER ratings_stats_sync_trigger
    AFTER INSERT OR UPDATE OR DELETE ON ratings
    FOR EACH ROW EXECUTE FUNCTION ratings_stats_sync();

-- Give every game a stats row up front so browse queries never depend on one
-- existing yet.
CREATE OR REPLACE FUNCTION games_stats_bootstrap() RETURNS trigger AS $$
BEGIN
    INSERT INTO game_stats (game_id) VALUES (NEW.id) ON CONFLICT DO NOTHING;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS games_stats_bootstrap_trigger ON games;
CREATE TRIGGER games_stats_bootstrap_trigger
    AFTER INSERT ON games
    FOR EACH ROW EXECUTE FUNCTION games_stats_bootstrap();

-- Recomputes C, the global mean, from every rating cast. Called hourly by cron;
-- cheap enough to run on demand at this scale.
CREATE OR REPLACE FUNCTION refresh_global_stats() RETURNS void AS $$
BEGIN
    UPDATE global_stats
       SET mean_rating = COALESCE(
               (SELECT sum(rating_sum) / NULLIF(sum(num_ratings), 0)
                  FROM game_stats
                 WHERE num_ratings > 0),
               7.0
           ),
           updated_at = now()
     WHERE id = true;
END;
$$ LANGUAGE plpgsql;

-- Backfill for databases that already hold data when this migration runs.
INSERT INTO game_stats (game_id) SELECT id FROM games ON CONFLICT DO NOTHING;

UPDATE game_stats gs
   SET num_ratings = agg.n,
       rating_sum  = agg.s
  FROM (SELECT game_id, count(*) AS n, sum(value) AS s FROM ratings GROUP BY game_id) agg
 WHERE gs.game_id = agg.game_id
   AND (gs.num_ratings, gs.rating_sum) IS DISTINCT FROM (agg.n, agg.s);

SELECT refresh_global_stats();
