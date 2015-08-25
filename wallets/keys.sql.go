package wallets

//go:generate go run gen.go wallets keySQL keys.sql
const keySQL = `CREATE OR REPLACE FUNCTION key_index(n bigint) RETURNS integer[]
	LANGUAGE plpgsql
	AS $$
DECLARE
	maxint32 int := x'7fffffff'::int;
BEGIN
	RETURN ARRAY[(n>>31) & maxint32, n & maxint32];
END;
$$;
`
