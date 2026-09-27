package facilities

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HardwareAsset struct {
	AssetID            string     `json:"asset_id"`
	SerialNumber       string     `json:"serial_number"`
	AssetType          string     `json:"asset_type"`
	Model              string     `json:"model"`
	Status             string     `json:"status"`
	ZoneID             *string    `json:"zone_id,omitempty"`
	AssignedEmployeeID *string    `json:"assigned_employee_id,omitempty"`
	LastPingAt         *time.Time `json:"last_ping_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type MaintenanceTicket struct {
	TicketID             string     `json:"ticket_id"`
	AssetID              string     `json:"asset_id"`
	Title                string     `json:"title"`
	Description          string     `json:"description"`
	Priority             string     `json:"priority"`
	Status               string     `json:"status"`
	AssignedTechnicianID *string    `json:"assigned_technician_id,omitempty"`
	ReportedBy           *string    `json:"reported_by,omitempty"`
	ResolvedAt           *time.Time `json:"resolved_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type MaintenanceLog struct {
	LogID         string    `json:"log_id"`
	TicketID      *string   `json:"ticket_id,omitempty"`
	AssetID       string    `json:"asset_id"`
	TechnicianID  *string   `json:"technician_id,omitempty"`
	QRCodeScanned string    `json:"qr_code_scanned"`
	ActionTaken   string    `json:"action_taken"`
	Notes         *string   `json:"notes,omitempty"`
	SyncedAt      time.Time `json:"synced_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type FirmwareRelease struct {
	ReleaseID          string    `json:"release_id"`
	DeviceType         string    `json:"device_type"`
	Version            string    `json:"version"`
	FileURL            string    `json:"file_url"`
	Checksum           string    `json:"checksum"`
	MinHardwareVersion *string   `json:"min_hardware_version,omitempty"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type DeviceOTAJob struct {
	JobID        string     `json:"job_id"`
	AssetID      string     `json:"asset_id"`
	ReleaseID    string     `json:"release_id"`
	Status       string     `json:"status"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	ScheduledAt  time.Time  `json:"scheduled_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Repository interface {
	GetAssetByID(ctx context.Context, assetID string) (*HardwareAsset, error)
	ListAssets(ctx context.Context, limit int32, cursorCreatedAt *time.Time, cursorID *string) ([]*HardwareAsset, error)
	CreateAsset(ctx context.Context, asset *HardwareAsset) (*HardwareAsset, error)
	CreateMaintenanceTicket(ctx context.Context, ticket *MaintenanceTicket) (*MaintenanceTicket, error)
	BatchInsertMaintenanceLogs(ctx context.Context, logs []*MaintenanceLog) ([]*MaintenanceLog, error)
	ListMaintenanceLogsByAsset(ctx context.Context, assetID string) ([]*MaintenanceLog, error)
	CreateFirmwareRelease(ctx context.Context, release *FirmwareRelease) (*FirmwareRelease, error)
	ListFirmwareReleases(ctx context.Context, deviceType string) ([]*FirmwareRelease, error)
	CreateOTAJob(ctx context.Context, job *DeviceOTAJob) (*DeviceOTAJob, error)
	UpdateOTAJobStatus(ctx context.Context, jobID string, status string, errorMsg *string, completedAt *time.Time) (*DeviceOTAJob, error)
	ListOTAJobsByAsset(ctx context.Context, assetID string) ([]*DeviceOTAJob, error)
}

type pgxRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &pgxRepository{db: db}
}

func (r *pgxRepository) GetAssetByID(ctx context.Context, assetID string) (*HardwareAsset, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var a HardwareAsset
	query := `SELECT asset_id, serial_number, asset_type, model, status, zone_id, assigned_employee_id, last_ping_at, created_at, updated_at
              FROM facilities.hardware_assets WHERE asset_id = $1`
	err := r.db.QueryRow(ctx, query, assetID).Scan(
		&a.AssetID,
		&a.SerialNumber,
		&a.AssetType,
		&a.Model,
		&a.Status,
		&a.ZoneID,
		&a.AssignedEmployeeID,
		&a.LastPingAt,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *pgxRepository) ListAssets(ctx context.Context, limit int32, cursorCreatedAt *time.Time, cursorID *string) ([]*HardwareAsset, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT asset_id, serial_number, asset_type, model, status, zone_id, assigned_employee_id, last_ping_at, created_at, updated_at
              FROM facilities.hardware_assets
              WHERE ($1::timestamptz IS NULL OR $2::uuid IS NULL OR (created_at, asset_id) < ($1, $2))
              ORDER BY created_at DESC, asset_id DESC
              LIMIT $3`
	rows, err := r.db.Query(ctx, query, cursorCreatedAt, cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*HardwareAsset
	for rows.Next() {
		var a HardwareAsset
		if err := rows.Scan(&a.AssetID, &a.SerialNumber, &a.AssetType, &a.Model, &a.Status, &a.ZoneID, &a.AssignedEmployeeID, &a.LastPingAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		assets = append(assets, &a)
	}
	return assets, rows.Err()
}

func (r *pgxRepository) CreateAsset(ctx context.Context, asset *HardwareAsset) (*HardwareAsset, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO facilities.hardware_assets (
                  serial_number, asset_type, model, status, zone_id
              ) VALUES ($1, $2, $3, $4, $5)
              RETURNING asset_id, serial_number, asset_type, model, status, zone_id, assigned_employee_id, last_ping_at, created_at, updated_at`
	var a HardwareAsset
	err := r.db.QueryRow(ctx, query, asset.SerialNumber, asset.AssetType, asset.Model, asset.Status, asset.ZoneID).
		Scan(
			&a.AssetID,
			&a.SerialNumber,
			&a.AssetType,
			&a.Model,
			&a.Status,
			&a.ZoneID,
			&a.AssignedEmployeeID,
			&a.LastPingAt,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *pgxRepository) CreateMaintenanceTicket(ctx context.Context, ticket *MaintenanceTicket) (*MaintenanceTicket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO facilities.maintenance_tickets (
                  asset_id, title, description, priority, status
              ) VALUES ($1, $2, $3, $4, 'open')
              RETURNING ticket_id, asset_id, title, description, priority, status, assigned_technician_id, reported_by, resolved_at, created_at, updated_at`
	var t MaintenanceTicket
	err := r.db.QueryRow(ctx, query, ticket.AssetID, ticket.Title, ticket.Description, ticket.Priority).
		Scan(
			&t.TicketID,
			&t.AssetID,
			&t.Title,
			&t.Description,
			&t.Priority,
			&t.Status,
			&t.AssignedTechnicianID,
			&t.ReportedBy,
			&t.ResolvedAt,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *pgxRepository) BatchInsertMaintenanceLogs(ctx context.Context, logs []*MaintenanceLog) ([]*MaintenanceLog, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `INSERT INTO facilities.maintenance_logs (ticket_id, asset_id, technician_id, qr_code_scanned, action_taken, notes, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING log_id, synced_at, created_at`
	result := make([]*MaintenanceLog, len(logs))
	for i, l := range logs {
		cp := *l
		if cp.CreatedAt.IsZero() {
			cp.CreatedAt = time.Now().UTC()
		}
		err := r.db.QueryRow(ctx, query, cp.TicketID, cp.AssetID, cp.TechnicianID, cp.QRCodeScanned, cp.ActionTaken, cp.Notes, cp.CreatedAt).Scan(&cp.LogID, &cp.SyncedAt, &cp.CreatedAt)
		if err != nil {
			return nil, err
		}
		result[i] = &cp
	}
	return result, nil
}

func (r *pgxRepository) ListMaintenanceLogsByAsset(ctx context.Context, assetID string) ([]*MaintenanceLog, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `SELECT log_id, ticket_id, asset_id, technician_id, qr_code_scanned, action_taken, notes, synced_at, created_at FROM facilities.maintenance_logs WHERE asset_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*MaintenanceLog
	for rows.Next() {
		var l MaintenanceLog
		if err := rows.Scan(&l.LogID, &l.TicketID, &l.AssetID, &l.TechnicianID, &l.QRCodeScanned, &l.ActionTaken, &l.Notes, &l.SyncedAt, &l.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &l)
	}
	return list, rows.Err()
}

func (r *pgxRepository) CreateFirmwareRelease(ctx context.Context, release *FirmwareRelease) (*FirmwareRelease, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if release.Status == "" {
		release.Status = "published"
	}
	query := `INSERT INTO facilities.firmware_releases (device_type, version, file_url, checksum, min_hardware_version, status) VALUES ($1, $2, $3, $4, $5, $6) RETURNING release_id, created_at, updated_at`
	err := r.db.QueryRow(ctx, query, release.DeviceType, release.Version, release.FileURL, release.Checksum, release.MinHardwareVersion, release.Status).Scan(&release.ReleaseID, &release.CreatedAt, &release.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return release, nil
}

func (r *pgxRepository) ListFirmwareReleases(ctx context.Context, deviceType string) ([]*FirmwareRelease, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `SELECT release_id, device_type, version, file_url, checksum, min_hardware_version, status, created_at, updated_at FROM facilities.firmware_releases WHERE device_type = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, deviceType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*FirmwareRelease
	for rows.Next() {
		var f FirmwareRelease
		if err := rows.Scan(&f.ReleaseID, &f.DeviceType, &f.Version, &f.FileURL, &f.Checksum, &f.MinHardwareVersion, &f.Status, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &f)
	}
	return list, rows.Err()
}

func (r *pgxRepository) CreateOTAJob(ctx context.Context, job *DeviceOTAJob) (*DeviceOTAJob, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `INSERT INTO facilities.device_ota_jobs (asset_id, release_id, status) VALUES ($1, $2, 'pending') RETURNING job_id, status, error_message, scheduled_at, completed_at, created_at, updated_at`
	err := r.db.QueryRow(ctx, query, job.AssetID, job.ReleaseID).Scan(&job.JobID, &job.Status, &job.ErrorMessage, &job.ScheduledAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *pgxRepository) UpdateOTAJobStatus(ctx context.Context, jobID string, status string, errorMsg *string, completedAt *time.Time) (*DeviceOTAJob, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `UPDATE facilities.device_ota_jobs SET status = $2, error_message = $3, completed_at = $4, updated_at = CURRENT_TIMESTAMP WHERE job_id = $1 RETURNING job_id, asset_id, release_id, status, error_message, scheduled_at, completed_at, created_at, updated_at`
	var j DeviceOTAJob
	err := r.db.QueryRow(ctx, query, jobID, status, errorMsg, completedAt).Scan(&j.JobID, &j.AssetID, &j.ReleaseID, &j.Status, &j.ErrorMessage, &j.ScheduledAt, &j.CompletedAt, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *pgxRepository) ListOTAJobsByAsset(ctx context.Context, assetID string) ([]*DeviceOTAJob, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `SELECT job_id, asset_id, release_id, status, error_message, scheduled_at, completed_at, created_at, updated_at FROM facilities.device_ota_jobs WHERE asset_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DeviceOTAJob
	for rows.Next() {
		var j DeviceOTAJob
		if err := rows.Scan(&j.JobID, &j.AssetID, &j.ReleaseID, &j.Status, &j.ErrorMessage, &j.ScheduledAt, &j.CompletedAt, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &j)
	}
	return list, rows.Err()
}
