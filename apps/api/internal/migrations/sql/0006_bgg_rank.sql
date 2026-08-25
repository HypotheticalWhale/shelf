-- BoardGameGeek's chart position, used only to break ties.
--
-- With tens of thousands of games and almost no ratings yet, every Bayesian
-- score sits at exactly the global mean, so "highest rated" returned whatever
-- Postgres happened to scan first — obscure titles ahead of the classics.
--
-- This column is never shown as a Shelf score and never mixed into one. It only
-- orders games that Shelf's own ratings cannot yet separate, and its influence
-- fades to nothing as real ratings arrive.
ALTER TABLE games ADD COLUMN IF NOT EXISTS bgg_rank integer;

CREATE INDEX IF NOT EXISTS games_bgg_rank_idx ON games (bgg_rank) WHERE bgg_rank IS NOT NULL;
