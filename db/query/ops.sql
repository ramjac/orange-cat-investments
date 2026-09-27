-- name: GetITTicketByID :one
SELECT ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, author_username, acknowledged_at, acknowledged_by, created_at, updated_at
FROM ops.it_tickets
WHERE ticket_id = $1;

-- name: ListOpenITTickets :many
SELECT ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, author_username, acknowledged_at, acknowledged_by, created_at, updated_at
FROM ops.it_tickets
WHERE state = 'open'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListITTickets :many
SELECT ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, author_username, acknowledged_at, acknowledged_by, created_at, updated_at
FROM ops.it_tickets
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateITTicket :one
INSERT INTO ops.it_tickets (
    forgejo_issue_id, forgejo_repo, title, body, state, author_username
) VALUES (
    $1, $2, $3, $4, 'open', $5
) RETURNING ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, author_username, acknowledged_at, acknowledged_by, created_at, updated_at;

-- name: AcknowledgeITTicket :one
UPDATE ops.it_tickets
SET state = 'acknowledged',
    acknowledged_at = CURRENT_TIMESTAMP,
    acknowledged_by = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE ticket_id = $1
RETURNING ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, author_username, acknowledged_at, acknowledged_by, created_at, updated_at;

-- name: CloseITTicket :one
UPDATE ops.it_tickets
SET state = 'closed',
    updated_at = CURRENT_TIMESTAMP
WHERE ticket_id = $1
RETURNING ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, author_username, acknowledged_at, acknowledged_by, created_at, updated_at;
