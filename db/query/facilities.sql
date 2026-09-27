-- name: GetAssetByID :one
SELECT asset_id, serial_number, asset_type, model, status, zone_id, assigned_employee_id, last_ping_at, created_at, updated_at
FROM facilities.hardware_assets
WHERE asset_id = $1;

-- name: ListAssets :many
SELECT asset_id, serial_number, asset_type, model, status, zone_id, assigned_employee_id, last_ping_at, created_at, updated_at
FROM facilities.hardware_assets
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

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

-- name: InsertMaintenanceLog :one
INSERT INTO facilities.maintenance_logs (
    ticket_id, asset_id, technician_id, qr_code_scanned, action_taken, notes, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING log_id, ticket_id, asset_id, technician_id, qr_code_scanned, action_taken, notes, synced_at, created_at;

-- name: ListMaintenanceLogsByAsset :many
SELECT log_id, ticket_id, asset_id, technician_id, qr_code_scanned, action_taken, notes, synced_at, created_at
FROM facilities.maintenance_logs
WHERE asset_id = $1
ORDER BY created_at DESC;

-- name: CreateFirmwareRelease :one
INSERT INTO facilities.firmware_releases (
    device_type, version, file_url, checksum, min_hardware_version, status
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING release_id, device_type, version, file_url, checksum, min_hardware_version, status, created_at, updated_at;

-- name: ListFirmwareReleases :many
SELECT release_id, device_type, version, file_url, checksum, min_hardware_version, status, created_at, updated_at
FROM facilities.firmware_releases
WHERE device_type = $1
ORDER BY created_at DESC;

-- name: CreateOTAJob :one
INSERT INTO facilities.device_ota_jobs (
    asset_id, release_id, status
) VALUES (
    $1, $2, 'pending'
) RETURNING job_id, asset_id, release_id, status, error_message, scheduled_at, completed_at, created_at, updated_at;

-- name: UpdateOTAJobStatus :one
UPDATE facilities.device_ota_jobs
SET status = $2, error_message = $3, completed_at = $4, updated_at = CURRENT_TIMESTAMP
WHERE job_id = $1
RETURNING job_id, asset_id, release_id, status, error_message, scheduled_at, completed_at, created_at, updated_at;

-- name: ListOTAJobsByAsset :many
SELECT job_id, asset_id, release_id, status, error_message, scheduled_at, completed_at, created_at, updated_at
FROM facilities.device_ota_jobs
WHERE asset_id = $1
ORDER BY created_at DESC;
