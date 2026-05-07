CREATE TYPE link_state AS ENUM (
    'active',
    'private',
    'deleted',
    'unavailable'
);

CREATE TABLE links (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    github_node_id text NOT NULL UNIQUE,
    github_database_id bigint NOT NULL,

    initial_full_name text NOT NULL,
    current_full_name text NOT NULL,
    current_url text NOT NULL,

    last_successful_full_name text,
    last_successful_url text,

    state link_state NOT NULL DEFAULT 'active',
    state_reason text,
    state_changed_at timestamptz,

    created_at timestamptz NOT NULL DEFAULT now(),
    verified_at timestamptz NOT NULL DEFAULT now(),
    verify_after timestamptz NOT NULL DEFAULT now() + interval '1 hour',

    fail_count integer NOT NULL DEFAULT 0,
    last_error text
);

CREATE TABLE github_node_id_aliases (
    node_id text PRIMARY KEY,
    link_id bigint NOT NULL REFERENCES links(id),
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX links_verify_after_idx
    ON links (verify_after)
    WHERE state = 'active';

CREATE INDEX links_state_idx ON links (state);

CREATE INDEX github_node_id_aliases_link_id_idx
    ON github_node_id_aliases (link_id);

CREATE INDEX links_current_full_name_lower_idx
    ON links (LOWER(current_full_name));

CREATE INDEX links_initial_full_name_lower_idx
    ON links (LOWER(initial_full_name));
