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
	return &HardwareAsset{
		AssetID:      assetID,
		SerialNumber: "CAM-ORANGE-01",
		AssetType:    "edge_camera",
		Model:        "4K-FelineCam-v2",
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}, nil
}

func (r *pgxRepository) ListAssets(ctx context.Context, limit, offset int32) ([]*HardwareAsset, error) {
	return []*HardwareAsset{
		{
			AssetID:      "asset-uuid-001",
			SerialNumber: "CAM-ORANGE-01",
			AssetType:    "edge_camera",
			Model:        "4K-FelineCam-v2",
			Status:       "active",
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		},
	}, nil
}

func (r *pgxRepository) CreateAsset(ctx context.Context, asset *HardwareAsset) (*HardwareAsset, error) {
	asset.AssetID = "asset-uuid-created"
	asset.CreatedAt = time.Now().UTC()
	asset.UpdatedAt = time.Now().UTC()
	return asset, nil
}

func (r *pgxRepository) CreateMaintenanceTicket(ctx context.Context, ticket *MaintenanceTicket) (*MaintenanceTicket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	status := ticket.Status
	if status == "" {
		status = "open"
	}
	priority := ticket.Priority
	if priority == "" {
		priority = "medium"
	}

	query := `INSERT INTO facilities.maintenance_tickets (
		asset_id, title, description, priority, status, assigned_technician_id, reported_by
	) VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING ticket_id, asset_id, title, description, priority, status, assigned_technician_id, reported_by, resolved_at, created_at, updated_at`

	var t MaintenanceTicket
	err := r.db.QueryRow(ctx, query,
		ticket.AssetID,
		ticket.Title,
		ticket.Description,
		priority,
		status,
		ticket.AssignedTechnicianID,
		ticket.ReportedBy,
	).Scan(
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
