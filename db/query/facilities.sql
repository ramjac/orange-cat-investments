-- name: GetAssetByID :one
SELECT asset_id, serial_number, asset_type, model, status, zone_id, assigned_employee_id, last_ping_at, created_at, updated_at
FROM facilities.hardware_assets
WHERE asset_id = $1;

-- name: ListAssets :many
SELECT asset_id, serial_number, asset_type, model, status, zone_id, assigned_employee_id, last_ping_at, created_at, updated_at
FROM facilities.hardware_assets
WHERE ($1::timestamptz IS NULL OR $2::uuid IS NULL OR (created_at, asset_id) < ($1, $2))
ORDER BY created_at DESC, asset_id DESC
LIMIT $3;

-- name: CreateAsset :one
INSERT INTO facilities.hardware_assets (
    serial_number, asset_type, model, status, zone_id
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING asset_id, serial_number, asset_type, model, status, zone_id, assigned_employee_id, last_ping_at, created_at, updated_at;

-- name: UpdateAssetStatus :one
UPDATE facilities.hardware_assets
SET status = $2, updated_at = CURRENT_TIMESTAMP
WHERE asset_id = $1
RETURNING asset_id, serial_number, asset_type, model, status, zone_id, assigned_employee_id, last_ping_at, created_at, updated_at;

-- name: CreateMaintenanceTicket :one
INSERT INTO facilities.maintenance_tickets (
    asset_id, title, description, priority, status
) VALUES (
    $1, $2, $3, $4, 'open'
) RETURNING ticket_id, asset_id, title, description, priority, status, assigned_technician_id, reported_by, resolved_at, created_at, updated_at;
