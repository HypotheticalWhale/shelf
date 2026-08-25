-- Collapses two tag vocabularies into one.
--
-- The catalogue was assembled from a 2016 snapshot and a hand-curated set, and
-- they name the same ideas differently: "Deck / Pool Building" against "Deck
-- Building", "Co-operative Play" against "Cooperative Game". Filtering by
-- either therefore returned only half the games that matched — "Deck Building"
-- found 8 of several hundred. The snapshot also carries a literal "NA".
--
-- Everything is mapped onto BoardGameGeek's current names, so a later import
-- with a real token lands in the same vocabulary rather than adding a third.

CREATE OR REPLACE FUNCTION normalise_tag(t text) RETURNS text AS $$
BEGIN
    RETURN CASE t
        WHEN 'Deck / Pool Building'            THEN 'Deck Building'
        WHEN 'Deck, Bag, and Pool Building'    THEN 'Deck Building'
        WHEN 'Co-operative Play'               THEN 'Cooperative Game'
        WHEN 'Area Control / Area Influence'   THEN 'Area Majority'
        WHEN 'Area Majority / Influence'       THEN 'Area Majority'
        WHEN 'Press Your Luck'                 THEN 'Push Your Luck'
        WHEN 'Action Point Allowance System'   THEN 'Action Points'
        WHEN 'Card Drafting'                   THEN 'Open Drafting'
        WHEN 'Route/Network Building'          THEN 'Network Building'
        WHEN 'Network and Route Building'      THEN 'Network Building'
        WHEN 'Simultaneous Action Selection'   THEN 'Simultaneous Action Selection'
        WHEN 'Hex-and-Counter'                 THEN 'Hexagon Grid'
        WHEN 'Partnerships'                    THEN 'Team-Based Game'
        WHEN 'Secret Unit Deployment'          THEN 'Hidden Movement'
        WHEN 'Campaign / Battle Card Driven'   THEN 'Card Driven'
        WHEN 'Area Enclosure'                  THEN 'Enclosure'
        ELSE t
    END;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Rewrite, drop the junk values, and de-duplicate what the mapping merged.
UPDATE games
   SET mechanics = COALESCE((
        SELECT array_agg(DISTINCT v ORDER BY v)
          FROM (SELECT normalise_tag(m) AS v FROM unnest(mechanics) m) t
         WHERE v <> '' AND v <> 'NA' AND v IS NOT NULL
   ), '{}'),
       categories = COALESCE((
        SELECT array_agg(DISTINCT v ORDER BY v)
          FROM (SELECT normalise_tag(c) AS v FROM unnest(categories) c) t
         WHERE v <> '' AND v <> 'NA' AND v IS NOT NULL
   ), '{}');
