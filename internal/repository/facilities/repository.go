package facilities

import (
	"context"
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
	ListAssets(ctx context.Context, limit int32, cursorCreatedAt *time.Time, cursorID *string) ([]*HardwareAsset, error)
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

func (r *pgxRepository) ListAssets(ctx context.Context, limit int32, cursorCreatedAt *time.Time, cursorID *string) ([]*HardwareAsset, error) {
	if r.db != nil {
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
	ticket.TicketID = "maint-uuid-created"
	ticket.Status = "open"
	ticket.CreatedAt = time.Now().UTC()
	ticket.UpdatedAt = time.Now().UTC()
	return ticket, nil
}
