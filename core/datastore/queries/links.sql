-- name: GetLinkByID :one
SELECT
    id, github_node_id, github_database_id,
    initial_full_name, current_full_name, current_url,
    last_successful_full_name, last_successful_url,
    state, state_reason, state_changed_at,
    created_at, verified_at, verify_after,
    fail_count, last_error
FROM links WHERE id = ?1;

-- name: GetLinkByNodeID :one
SELECT
    id, github_node_id, github_database_id,
    initial_full_name, current_full_name, current_url,
    last_successful_full_name, last_successful_url,
    state, state_reason, state_changed_at,
    created_at, verified_at, verify_after,
    fail_count, last_error
FROM links WHERE github_node_id = ?1;

-- name: GetLinkByCurrentFullName :one
SELECT
    id, github_node_id, github_database_id,
    initial_full_name, current_full_name, current_url,
    last_successful_full_name, last_successful_url,
    state, state_reason, state_changed_at,
    created_at, verified_at, verify_after,
    fail_count, last_error
FROM links
WHERE LOWER(current_full_name) = LOWER(?1)
LIMIT 1;

-- name: InsertLink :one
INSERT INTO links (
    github_node_id, github_database_id,
    initial_full_name, current_full_name, current_url,
    verify_after
) VALUES (
    ?1, ?2, ?3, ?3, ?4,
    strftime('%Y-%m-%d %H:%M:%f', 'now', ?5 || ' seconds')
)
ON CONFLICT (github_node_id) DO NOTHING
RETURNING
    id, github_node_id, github_database_id,
    initial_full_name, current_full_name, current_url,
    last_successful_full_name, last_successful_url,
    state, state_reason, state_changed_at,
    created_at, verified_at, verify_after,
    fail_count, last_error;

-- name: UpsertAlias :exec
INSERT INTO github_node_id_aliases (node_id, link_id)
VALUES (?1, ?2)
ON CONFLICT (node_id) DO UPDATE SET last_seen_at = strftime('%Y-%m-%d %H:%M:%f', 'now');

-- name: ApplySuccessfulVerification :one
UPDATE links SET
    current_full_name         = ?2,
    current_url               = ?3,
    last_successful_full_name = ?2,
    last_successful_url       = ?3,
    state                     = 'active',
    state_reason              = NULL,
    state_changed_at          = CASE WHEN state <> 'active' THEN strftime('%Y-%m-%d %H:%M:%f', 'now') ELSE state_changed_at END,
    verified_at               = strftime('%Y-%m-%d %H:%M:%f', 'now'),
    verify_after              = strftime('%Y-%m-%d %H:%M:%f', 'now', ?4 || ' seconds'),
    fail_count                = 0,
    last_error                = NULL
WHERE id = ?1
RETURNING
    id, github_node_id, github_database_id,
    initial_full_name, current_full_name, current_url,
    last_successful_full_name, last_successful_url,
    state, state_reason, state_changed_at,
    created_at, verified_at, verify_after,
    fail_count, last_error;

-- name: ApplyTransientFailure :one
UPDATE links SET
    fail_count   = fail_count + 1,
    last_error   = ?2,
    verify_after = strftime('%Y-%m-%d %H:%M:%f', 'now', ?3 || ' seconds')
WHERE id = ?1
RETURNING
    id, github_node_id, github_database_id,
    initial_full_name, current_full_name, current_url,
    last_successful_full_name, last_successful_url,
    state, state_reason, state_changed_at,
    created_at, verified_at, verify_after,
    fail_count, last_error;

-- name: ApplyTombstone :one
UPDATE links SET
    state            = ?2,
    state_reason     = ?3,
    state_changed_at = CASE WHEN state <> ?2 THEN strftime('%Y-%m-%d %H:%M:%f', 'now') ELSE state_changed_at END,
    verify_after     = strftime('%Y-%m-%d %H:%M:%f', 'now', ?4 || ' seconds'),
    last_error       = ?3
WHERE id = ?1
RETURNING
    id, github_node_id, github_database_id,
    initial_full_name, current_full_name, current_url,
    last_successful_full_name, last_successful_url,
    state, state_reason, state_changed_at,
    created_at, verified_at, verify_after,
    fail_count, last_error;

-- name: ClaimStaleLinks :many
UPDATE links
SET verify_after = strftime('%Y-%m-%d %H:%M:%f', 'now', ?2 || ' seconds')
WHERE id IN (
    SELECT id FROM links
    WHERE state = 'active' AND verify_after <= strftime('%Y-%m-%d %H:%M:%f', 'now')
    ORDER BY verify_after
    LIMIT CAST(?1 AS INTEGER)
)
RETURNING
    id, github_node_id, github_database_id,
    initial_full_name, current_full_name, current_url,
    last_successful_full_name, last_successful_url,
    state, state_reason, state_changed_at,
    created_at, verified_at, verify_after,
    fail_count, last_error;

-- name: CountActive :one
SELECT count(*) FROM links WHERE state = 'active';
