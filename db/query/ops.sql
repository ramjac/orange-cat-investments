-- name: GetITTicketByID :one
SELECT ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, resolution_notes, is_urgent, acknowledged_at, acknowledged_by, created_at, updated_at
FROM ops.it_tickets
WHERE ticket_id = $1;

-- name: ListOpenITTickets :many
SELECT ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, resolution_notes, is_urgent, acknowledged_at, acknowledged_by, created_at, updated_at
FROM ops.it_tickets
WHERE state = 'open'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListITTickets :many
SELECT ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, resolution_notes, is_urgent, acknowledged_at, acknowledged_by, created_at, updated_at
FROM ops.it_tickets
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListITTicketsBySource :many
SELECT ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, resolution_notes, is_urgent, acknowledged_at, acknowledged_by, created_at, updated_at
FROM ops.it_tickets
WHERE source = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListITTicketsByCustomer :many
SELECT ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, resolution_notes, is_urgent, acknowledged_at, acknowledged_by, created_at, updated_at
FROM ops.it_tickets
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateITTicket :one
INSERT INTO ops.it_tickets (
    forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, resolution_notes, is_urgent
) VALUES (
    $1, $2, $3, $4, 'open', $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, resolution_notes, is_urgent, acknowledged_at, acknowledged_by, created_at, updated_at;

-- name: AcknowledgeITTicket :one
UPDATE ops.it_tickets
SET state = 'acknowledged',
    acknowledged_at = CURRENT_TIMESTAMP,
    acknowledged_by = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE ticket_id = $1
RETURNING ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, resolution_notes, is_urgent, acknowledged_at, acknowledged_by, created_at, updated_at;

-- name: UpdateITTicket :one
UPDATE ops.it_tickets
SET state = COALESCE($2, state),
    assignee_username = COALESCE($3, assignee_username),
    resolution_notes = COALESCE($4, resolution_notes),
    updated_at = CURRENT_TIMESTAMP
WHERE ticket_id = $1
RETURNING ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, resolution_notes, is_urgent, acknowledged_at, acknowledged_by, created_at, updated_at;

-- name: CloseITTicket :one
UPDATE ops.it_tickets
SET state = 'closed',
    updated_at = CURRENT_TIMESTAMP
WHERE ticket_id = $1
RETURNING ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, resolution_notes, is_urgent, acknowledged_at, acknowledged_by, created_at, updated_at;

-- name: CreateTicketComment :one
INSERT INTO ops.ticket_comments (
    ticket_id, author_username, comment, is_internal
) VALUES (
    $1, $2, $3, $4
) RETURNING comment_id, ticket_id, author_username, comment, is_internal, created_at;

-- name: ListTicketCommentsAll :many
SELECT comment_id, ticket_id, author_username, comment, is_internal, created_at
FROM ops.ticket_comments
WHERE ticket_id = $1
ORDER BY created_at ASC;

-- name: ListTicketCommentsSanitized :many
SELECT comment_id, ticket_id, author_username, comment, is_internal, created_at
FROM ops.ticket_comments
WHERE ticket_id = $1 AND is_internal = FALSE
ORDER BY created_at ASC;
