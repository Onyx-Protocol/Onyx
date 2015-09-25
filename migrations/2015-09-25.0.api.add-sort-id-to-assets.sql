ALTER TABLE assets ADD COLUMN sort_id text NOT NULL DEFAULT next_chain_id('asset');
CREATE INDEX CONCURRENTLY assets_sort_id ON assets (sort_id);
WITH sorted AS (
	SELECT id FROM assets ORDER BY created_at ASC, id ASC
)
UPDATE assets SET sort_id=next_chain_id('asset')
FROM sorted
WHERE assets.id=sorted.id;
