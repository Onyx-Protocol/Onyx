ALTER SEQUENCE address_index_seq

	-- Does not affect the current seq value,
	-- only future ALTER SEQUENCE RESTART commands.
	START WITH 10001

	-- Each process will grab 10,000 units
	-- at a time, and dole them out from memory.
	INCREMENT BY 10000;
