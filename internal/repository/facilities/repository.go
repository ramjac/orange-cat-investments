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

type Repository interface {
	GetAssetByID(ctx context.Context, assetID string) (*HardwareAsset, error)
	ListAssets(ctx context.Context, limit, offset int32) ([]*HardwareAsset, error)
	CreateAsset(ctx context.Context, asset *HardwareAsset) (*HardwareAsset, error)
	CreateMaintenanceTicket(ctx context.Context, ticket *MaintenanceTicket) (*MaintenanceTicket, error)
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
	var asset HardwareAsset
	query := `SELECT asset_id, serial_number, asset_type, model, status, zone_id, assigned_employee_id, last_ping_at, created_at, updated_at
              FROM facilities.hardware_assets WHERE asset_id = $1`
	err := r.db.QueryRow(ctx, query, assetID).Scan(
		&asset.AssetID,
		&asset.SerialNumber,
		&asset.AssetType,
		&asset.Model,
		&asset.Status,
		&asset.ZoneID,
		&asset.AssignedEmployeeID,
		&asset.LastPingAt,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *pgxRepository) ListAssets(ctx context.Context, limit, offset int32) ([]*HardwareAsset, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT asset_id, serial_number, asset_type, model, status, zone_id, assigned_employee_id, last_ping_at, created_at, updated_at
              FROM facilities.hardware_assets ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*HardwareAsset
	for rows.Next() {
		var asset HardwareAsset
		if err := rows.Scan(
			&asset.AssetID,
			&asset.SerialNumber,
			&asset.AssetType,
			&asset.Model,
			&asset.Status,
			&asset.ZoneID,
			&asset.AssignedEmployeeID,
			&asset.LastPingAt,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		); err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
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
	var created HardwareAsset
	err := r.db.QueryRow(ctx, query, asset.SerialNumber, asset.AssetType, asset.Model, asset.Status, asset.ZoneID).Scan(
		&created.AssetID,
		&created.SerialNumber,
		&created.AssetType,
		&created.Model,
		&created.Status,
		&created.ZoneID,
		&created.AssignedEmployeeID,
		&created.LastPingAt,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (r *pgxRepository) CreateMaintenanceTicket(ctx context.Context, ticket *MaintenanceTicket) (*MaintenanceTicket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO facilities.maintenance_tickets (
                  asset_id, title, description, priority, status
              ) VALUES ($1, $2, $3, $4, 'open')
              RETURNING ticket_id, asset_id, title, description, priority, status, assigned_technician_id, reported_by, resolved_at, created_at, updated_at`
	var created MaintenanceTicket
	err := r.db.QueryRow(ctx, query, ticket.AssetID, ticket.Title, ticket.Description, ticket.Priority).Scan(
		&created.TicketID,
		&created.AssetID,
		&created.Title,
		&created.Description,
		&created.Priority,
		&created.Status,
		&created.AssignedTechnicianID,
		&created.ReportedBy,
		&created.ResolvedAt,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &created, nil
}
