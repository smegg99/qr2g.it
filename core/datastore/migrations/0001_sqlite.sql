-- core/datastore/migrations/0001_sqlite.sql
CREATE TABLE links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    github_node_id TEXT NOT NULL UNIQUE,
    github_database_id INTEGER NOT NULL,
    initial_full_name TEXT NOT NULL,
    current_full_name TEXT NOT NULL,
    current_url TEXT NOT NULL,
    last_successful_full_name TEXT,
    last_successful_url TEXT,
    state TEXT NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'private', 'deleted', 'unavailable')),
    state_reason TEXT,
    state_changed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
    verified_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
    verify_after DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now', '+1 hour')),
    fail_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT
);

CREATE TABLE github_node_id_aliases (
    node_id TEXT PRIMARY KEY NOT NULL,
    link_id INTEGER NOT NULL REFERENCES links(id),
    first_seen_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
    last_seen_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
);

CREATE INDEX links_verify_after_idx ON links(verify_after) WHERE state = 'active';
CREATE INDEX links_state_idx ON links(state);
CREATE INDEX github_node_id_aliases_link_id_idx ON github_node_id_aliases(link_id);
CREATE INDEX links_current_full_name_lower_idx ON links(LOWER(current_full_name));
CREATE INDEX links_initial_full_name_lower_idx ON links(LOWER(initial_full_name));
