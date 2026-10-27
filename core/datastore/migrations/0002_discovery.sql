-- core/datastore/migrations/0002_discovery.sql
-- A GitHub 404 cannot tell deleted from private, so stored "deleted" meant unavailable.
UPDATE links SET state = 'unavailable', state_reason = 'repository is no longer publicly available'
WHERE state = 'deleted';

ALTER TABLE links ADD COLUMN description TEXT;
ALTER TABLE links ADD COLUMN language TEXT;
ALTER TABLE links ADD COLUMN archived INTEGER NOT NULL DEFAULT 0;
ALTER TABLE links ADD COLUMN default_branch TEXT;
ALTER TABLE links ADD COLUMN noindex INTEGER NOT NULL DEFAULT 0;
ALTER TABLE links ADD COLUMN noanalytics INTEGER NOT NULL DEFAULT 0;
ALTER TABLE links ADD COLUMN policy_checked_at DATETIME;
ALTER TABLE links ADD COLUMN policy_failed INTEGER NOT NULL DEFAULT 0;
-- Existing links are due at once: they become listed only after a fresh check.
ALTER TABLE links ADD COLUMN policy_check_after DATETIME NOT NULL DEFAULT '1970-01-01 00:00:00.000';

CREATE INDEX links_policy_check_after_idx ON links(policy_check_after) WHERE state = 'active';

-- Operator decisions after a report. 'unlist' hides the repository from public
-- surfaces; 'remove' also stops the redirect. Keyed by link so resubmission keeps it.
CREATE TABLE moderation (
    link_id INTEGER PRIMARY KEY NOT NULL REFERENCES links(id),
    action TEXT NOT NULL CHECK (action IN ('unlist', 'remove')),
    note TEXT,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
);

-- Filtered redirects per link and UTC day. No visitor data.
CREATE TABLE redirect_daily (
    link_id INTEGER NOT NULL REFERENCES links(id),
    day TEXT NOT NULL,
    redirects INTEGER NOT NULL CHECK (redirects >= 0),
    PRIMARY KEY (link_id, day)
) WITHOUT ROWID;

CREATE INDEX redirect_daily_day_idx ON redirect_daily(day);

-- The repository's largest languages as a JSON array of {"name", "share"}, refreshed with its facts.
ALTER TABLE links ADD COLUMN languages TEXT;

-- GitHub's star count, refreshed with the repository's facts; NULL until it is first read.
ALTER TABLE links ADD COLUMN stars INTEGER;

-- The time of each link's latest filtered redirect. No visitor data.
CREATE TABLE redirect_latest (
    link_id INTEGER PRIMARY KEY NOT NULL REFERENCES links(id),
    at DATETIME NOT NULL
);

CREATE TABLE analytics_pauses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    link_id INTEGER NOT NULL REFERENCES links(id),
    reason TEXT NOT NULL CHECK (reason IN ('owner_noanalytics')),
    started_on TEXT NOT NULL,
    ended_on TEXT
);

CREATE UNIQUE INDEX analytics_pauses_open_idx ON analytics_pauses(link_id) WHERE ended_on IS NULL;

CREATE TABLE service_facts (
    key TEXT PRIMARY KEY NOT NULL,
    value TEXT NOT NULL
);

INSERT INTO service_facts (key, value) VALUES ('redirect_collection_started_on', strftime('%Y-%m-%d', 'now'));

CREATE TABLE automation_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    daily_quota INTEGER NOT NULL CHECK (daily_quota > 0),
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
    revoked_at DATETIME,
    last_used_at DATETIME
);

CREATE TABLE automation_usage (
    token_id INTEGER NOT NULL REFERENCES automation_tokens(id),
    day TEXT NOT NULL,
    requests INTEGER NOT NULL CHECK (requests >= 0),
    PRIMARY KEY (token_id, day)
) WITHOUT ROWID;
