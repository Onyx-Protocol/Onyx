UPDATE addresses SET keyset=(SELECT array_agg(xpub) FROM keys WHERE id=ANY(keyset));
UPDATE asset_groups SET keyset=(SELECT array_agg(xpub) FROM keys WHERE id=ANY(keyset));
UPDATE assets SET keyset=(SELECT array_agg(xpub) FROM keys WHERE id=ANY(keyset));
UPDATE rotations SET keyset=(SELECT array_agg(xpub) FROM keys WHERE id=ANY(keyset));
DROP TABLE keys;
