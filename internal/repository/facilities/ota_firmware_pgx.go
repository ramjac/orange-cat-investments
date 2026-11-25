package facilities

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/orange-cat-investments/oci/pkg/outbox"
)

func (r *pgxRepository) CreateFirmwareRelease(ctx context.Context, release *FirmwareRelease) (*FirmwareRelease, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if release.Status == "" {
		release.Status = "published"
	}
	query := `INSERT INTO facilities.firmware_releases (device_type, version, file_url, checksum, min_hardware_version, cosign_signature, status) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING release_id, created_at, updated_at`
	err := r.db.QueryRow(ctx, query, release.DeviceType, release.Version, release.FileURL, release.Checksum, release.MinHardwareVersion, release.CosignSignature, release.Status).Scan(&release.ReleaseID, &release.CreatedAt, &release.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return release, nil
}

func (r *pgxRepository) ListFirmwareReleases(ctx context.Context, deviceType string) ([]*FirmwareRelease, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	var (
		rows pgx.Rows
		err  error
	)
	if deviceType == "" {
		query := `SELECT release_id, device_type, version, file_url, checksum, min_hardware_version, cosign_signature, status, created_at, updated_at FROM facilities.firmware_releases ORDER BY created_at DESC`
		rows, err = r.db.Query(ctx, query)
	} else {
		query := `SELECT release_id, device_type, version, file_url, checksum, min_hardware_version, cosign_signature, status, created_at, updated_at FROM facilities.firmware_releases WHERE device_type = $1 ORDER BY created_at DESC`
		rows, err = r.db.Query(ctx, query, deviceType)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*FirmwareRelease
	for rows.Next() {
		var f FirmwareRelease
		if err := rows.Scan(&f.ReleaseID, &f.DeviceType, &f.Version, &f.FileURL, &f.Checksum, &f.MinHardwareVersion, &f.CosignSignature, &f.Status, &f.CreatedAt, &f.UpdatedAt); err != nil {
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
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create ota job tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := `INSERT INTO facilities.device_ota_jobs (asset_id, release_id, status) VALUES ($1, $2, 'pending') RETURNING job_id, status, error_message, scheduled_at, completed_at, created_at, updated_at`
	err = tx.QueryRow(ctx, query, job.AssetID, job.ReleaseID).Scan(&job.JobID, &job.Status, &job.ErrorMessage, &job.ScheduledAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		return nil, err
	}

	otaPayload := envelope.OTATriggeredPayload{
		JobID:     job.JobID,
		AssetID:   job.AssetID,
		ReleaseID: job.ReleaseID,
	}
	var devType, ver, fileURL, checksum, sig string
	if relErr := tx.QueryRow(ctx, `SELECT device_type, version, file_url, checksum, cosign_signature FROM facilities.firmware_releases WHERE release_id = $1`, job.ReleaseID).
		Scan(&devType, &ver, &fileURL, &checksum, &sig); relErr == nil {
		otaPayload.DeviceType = devType
		otaPayload.Version = ver
		otaPayload.FileURL = fileURL
		otaPayload.Checksum = checksum
		otaPayload.CosignSignature = sig
	}
	evID, envBytes, err := outbox.BuildEnvelopeBytes(ctx, "", envelope.TopicOTATriggered, job.JobID, otaPayload)
	if err != nil {
		return nil, err
	}
	if err := outbox.InsertTx(ctx, tx, outbox.SchemaFacilities, evID, envelope.TopicOTATriggered, job.JobID, envBytes); err != nil {
		return nil, fmt.Errorf("insert ota outbox event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create ota job tx: %w", err)
	}
	return job, nil
}

func (r *pgxRepository) BatchCreateOTAJobs(ctx context.Context, jobs []*DeviceOTAJob) ([]*DeviceOTAJob, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if len(jobs) == 0 {
		return []*DeviceOTAJob{}, nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin batch create ota jobs tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	batch := &pgx.Batch{}
	query := `INSERT INTO facilities.device_ota_jobs (asset_id, release_id, status) VALUES ($1, $2, 'pending') RETURNING job_id, asset_id, release_id, status, error_message, scheduled_at, completed_at, created_at, updated_at`
	for _, job := range jobs {
		batch.Queue(query, job.AssetID, job.ReleaseID)
	}

	br := tx.SendBatch(ctx, batch)
	result := make([]*DeviceOTAJob, 0, len(jobs))
	for i := 0; i < len(jobs); i++ {
		var j DeviceOTAJob
		err := br.QueryRow().Scan(&j.JobID, &j.AssetID, &j.ReleaseID, &j.Status, &j.ErrorMessage, &j.ScheduledAt, &j.CompletedAt, &j.CreatedAt, &j.UpdatedAt)
		if err != nil {
			_ = br.Close()
			return nil, err
		}
		result = append(result, &j)
	}
	if err := br.Close(); err != nil {
		return nil, err
	}

	var devType, ver, fileURL, checksum, sig string
	relLoaded := false
	if len(result) > 0 {
		if relErr := tx.QueryRow(ctx, `SELECT device_type, version, file_url, checksum, cosign_signature FROM facilities.firmware_releases WHERE release_id = $1`, result[0].ReleaseID).
			Scan(&devType, &ver, &fileURL, &checksum, &sig); relErr == nil {
			relLoaded = true
		}
	}
	for _, j := range result {
		otaPayload := envelope.OTATriggeredPayload{
			JobID:     j.JobID,
			AssetID:   j.AssetID,
			ReleaseID: j.ReleaseID,
		}
		if relLoaded {
			otaPayload.DeviceType = devType
			otaPayload.Version = ver
			otaPayload.FileURL = fileURL
			otaPayload.Checksum = checksum
			otaPayload.CosignSignature = sig
		}
		evID, envBytes, err := outbox.BuildEnvelopeBytes(ctx, "", envelope.TopicOTATriggered, j.JobID, otaPayload)
		if err != nil {
			return nil, err
		}
		if err := outbox.InsertTx(ctx, tx, outbox.SchemaFacilities, evID, envelope.TopicOTATriggered, j.JobID, envBytes); err != nil {
			return nil, fmt.Errorf("insert batch ota outbox event: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit batch create ota jobs tx: %w", err)
	}
	return result, nil
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
