ALTER TABLE query_indexes RENAME COLUMN internal_id TO id;
ALTER TABLE query_indexes
    ALTER COLUMN id TYPE text,
    ALTER COLUMN id SET DEFAULT next_chain_id('idx'::text);
