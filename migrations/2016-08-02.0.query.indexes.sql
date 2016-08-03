CREATE TABLE query_indexes (
    internal_id    serial primary key,
    id             text NOT NULL,
    type           text NOT NULL,
    query          text NOT NULL,
    created_at     timestamp with time zone DEFAULT now() NOT NULL,
    UNIQUE(id)
);
