-- Supports browse sorting once the catalogue is tens of thousands of games.
--
-- Sorting alphabetically over 31k rows was the slowest browse path; without an
-- index Postgres sorts the whole table on every page. The Bayesian score cannot
-- be indexed directly because it depends on global_stats, which changes as
-- ratings arrive.
CREATE INDEX IF NOT EXISTS games_name_idx ON games (name);

-- Player-count and playtime filters scan the whole catalogue otherwise.
CREATE INDEX IF NOT EXISTS games_players_idx  ON games (min_players, max_players);
CREATE INDEX IF NOT EXISTS games_playtime_idx ON games (max_playtime);
