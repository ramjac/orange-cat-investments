package workforce

import (
	"context"
	"fmt"
)

func (r *pgxRepository) CreateLeaveRequest(ctx context.Context, req *LeaveRequest) (*LeaveRequest, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.leave_requests (
		leave_id, employee_id, leave_type, start_date, end_date, status, reason
	) VALUES (
		COALESCE(NULLIF($1, '')::uuid, gen_random_uuid_v7()), $2, $3, $4::date, $5::date, COALESCE(NULLIF($6, ''), 'pending'), $7
	) RETURNING leave_id::text, employee_id::text, leave_type, start_date::text, end_date::text, status, reason, created_at, updated_at`
	var res LeaveRequest
	err := r.db.QueryRow(ctx, query, req.LeaveID, req.EmployeeID, req.LeaveType, req.StartDate, req.EndDate, req.Status, req.Reason).Scan(
		&res.LeaveID, &res.EmployeeID, &res.LeaveType, &res.StartDate, &res.EndDate, &res.Status, &res.Reason, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *pgxRepository) GetLeaveRequestByID(ctx context.Context, id string) (*LeaveRequest, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT leave_id::text, employee_id::text, leave_type, start_date::text, end_date::text, status, reason, created_at, updated_at
	          FROM workforce.leave_requests WHERE leave_id::text = $1`
	var res LeaveRequest
	err := r.db.QueryRow(ctx, query, id).Scan(
		&res.LeaveID, &res.EmployeeID, &res.LeaveType, &res.StartDate, &res.EndDate, &res.Status, &res.Reason, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("leave request with id %s not found: %w", id, err)
	}
	return &res, nil
}

func (r *pgxRepository) ListLeaveRequests(ctx context.Context, employeeID, status string) ([]*LeaveRequest, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	canonicalEmpID := employeeID
	if employeeID != "" {
		if emp, err := r.GetEmployeeByID(ctx, employeeID); err == nil && emp != nil {
			canonicalEmpID = emp.EmployeeID
		}
	}

	query := `SELECT lr.leave_id::text, lr.employee_id::text, lr.leave_type, lr.start_date::text, lr.end_date::text, lr.status, lr.reason, lr.created_at, lr.updated_at
	          FROM workforce.leave_requests lr
	          WHERE ($1 = '' OR lr.employee_id::text = $1)
	            AND ($2 = '' OR lr.status = $2)
	          ORDER BY lr.created_at DESC`
	rows, err := r.db.Query(ctx, query, canonicalEmpID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*LeaveRequest
	for rows.Next() {
		var res LeaveRequest
		if err := rows.Scan(&res.LeaveID, &res.EmployeeID, &res.LeaveType, &res.StartDate, &res.EndDate, &res.Status, &res.Reason, &res.CreatedAt, &res.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &res)
	}
	if list == nil {
		list = []*LeaveRequest{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) UpdateLeaveRequestStatus(ctx context.Context, id, status string) (*LeaveRequest, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE workforce.leave_requests
	          SET status = $2, updated_at = CURRENT_TIMESTAMP
	          WHERE leave_id::text = $1
	          RETURNING leave_id::text, employee_id::text, leave_type, start_date::text, end_date::text, status, reason, created_at, updated_at`
	var res LeaveRequest
	err := r.db.QueryRow(ctx, query, id, status).Scan(
		&res.LeaveID, &res.EmployeeID, &res.LeaveType, &res.StartDate, &res.EndDate, &res.Status, &res.Reason, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("leave request with id %s not found: %w", id, err)
	}
	return &res, nil
}

func scanReviewCycle(row interface{ Scan(dest ...any) error }) (*ReviewCycle, error) {
	var res ReviewCycle
	var schedFor string
	var revID, peerRevID, clinAudID, vhrID *string
	err := row.Scan(
		&res.ReviewID, &res.EmployeeID, &res.ReviewType, &res.ReviewPeriod, &res.Phase, &schedFor, &res.Status,
		&revID, &res.Score, &res.Notes, &res.SelfAppraisal, &res.SelfScore, &res.PeerFeedback, &res.PeerScore,
		&peerRevID, &res.ManagerAssessment, &res.ManagerScore, &res.KPIRatings,
		&res.WellnessAuditScore, &res.HabitatRecommendations, &clinAudID, &vhrID,
		&res.CompletedAt, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	res.ScheduledFor = schedFor
	res.ReviewerID = revID
	res.PeerReviewerID = peerRevID
	res.ClinicalAuditorID = clinAudID
	res.VHRRecordID = vhrID
	return &res, nil
}

func (r *pgxRepository) CreateReviewCycle(ctx context.Context, rc *ReviewCycle) (*ReviewCycle, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.review_cycles (
	              review_id, employee_id, review_type, review_period, phase, scheduled_for, status,
	              reviewer_id, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
	              peer_reviewer_id, manager_assessment, manager_score, kpi_ratings,
	              wellness_audit_score, habitat_recommendations, clinical_auditor_id, vhr_record_id, completed_at
	          ) VALUES (
	              COALESCE($1, gen_random_uuid_v7()), $2, $3, COALESCE(NULLIF($4, ''), '2026-Q3'), COALESCE(NULLIF($5, ''), 'self_appraisal'),
	              $6, COALESCE(NULLIF($7, ''), 'scheduled'), $8, $9, $10, $11, $12, $13, $14,
	              $15, $16, $17, $18::jsonb, $19, $20, $21, $22, $23
	          ) RETURNING review_id::text, employee_id::text, review_type, review_period, phase, scheduled_for::text, status,
	                      reviewer_id::text, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
	                      peer_reviewer_id::text, manager_assessment, manager_score, kpi_ratings::text,
	                      wellness_audit_score, habitat_recommendations, clinical_auditor_id::text, vhr_record_id::text,
	                      completed_at, created_at, updated_at`
	var revUUID *string
	if rc.ReviewID != "" {
		revUUID = &rc.ReviewID
	}
	row := r.db.QueryRow(ctx, query,
		revUUID, rc.EmployeeID, rc.ReviewType, rc.ReviewPeriod, rc.Phase,
		rc.ScheduledFor, rc.Status, rc.ReviewerID, rc.Score, rc.Notes,
		rc.SelfAppraisal, rc.SelfScore, rc.PeerFeedback, rc.PeerScore,
		rc.PeerReviewerID, rc.ManagerAssessment, rc.ManagerScore, rc.KPIRatings,
		rc.WellnessAuditScore, rc.HabitatRecommendations, rc.ClinicalAuditorID, rc.VHRRecordID,
		rc.CompletedAt,
	)
	return scanReviewCycle(row)
}

func (r *pgxRepository) GetReviewCycleByID(ctx context.Context, id string) (*ReviewCycle, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT review_id::text, employee_id::text, review_type, review_period, phase, scheduled_for::text, status,
	                 reviewer_id::text, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
	                 peer_reviewer_id::text, manager_assessment, manager_score, kpi_ratings::text,
	                 wellness_audit_score, habitat_recommendations, clinical_auditor_id::text, vhr_record_id::text,
	                 completed_at, created_at, updated_at
	          FROM workforce.review_cycles
	          WHERE review_id::text = $1`
	row := r.db.QueryRow(ctx, query, id)
	res, err := scanReviewCycle(row)
	if err != nil {
		return nil, fmt.Errorf("review cycle with id %s not found: %w", id, err)
	}
	return res, nil
}

func (r *pgxRepository) ListReviewCycles(ctx context.Context, employeeID, status, reviewType string) ([]*ReviewCycle, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	canonicalEmpID := employeeID
	if employeeID != "" {
		if emp, err := r.GetEmployeeByID(ctx, employeeID); err == nil && emp != nil {
			canonicalEmpID = emp.EmployeeID
		}
	}

	query := `SELECT rc.review_id::text, rc.employee_id::text, rc.review_type, rc.review_period, rc.phase, rc.scheduled_for::text, rc.status,
	                 rc.reviewer_id::text, rc.score, rc.notes, rc.self_appraisal, rc.self_score, rc.peer_feedback, rc.peer_score,
	                 rc.peer_reviewer_id::text, rc.manager_assessment, rc.manager_score, rc.kpi_ratings::text,
	                 rc.wellness_audit_score, rc.habitat_recommendations, rc.clinical_auditor_id::text, rc.vhr_record_id::text,
	                 rc.completed_at, rc.created_at, rc.updated_at
	          FROM workforce.review_cycles rc
	          WHERE ($1 = '' OR rc.employee_id::text = $1)
	            AND ($2 = '' OR rc.status = $2)
	            AND ($3 = '' OR rc.review_type = $3)
	          ORDER BY rc.scheduled_for DESC, rc.created_at DESC`
	rows, err := r.db.Query(ctx, query, canonicalEmpID, status, reviewType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*ReviewCycle
	for rows.Next() {
		res, err := scanReviewCycle(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	if list == nil {
		list = []*ReviewCycle{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) UpdateReviewCycle(ctx context.Context, rc *ReviewCycle) (*ReviewCycle, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE workforce.review_cycles
	          SET status = COALESCE($2, status),
	              phase = CASE WHEN $3::text IS NOT NULL THEN $3::text ELSE phase END,
	              reviewer_id = CASE WHEN $4::text IS NOT NULL THEN $4::uuid ELSE reviewer_id END,
	              score = CASE WHEN $5::numeric IS NOT NULL THEN $5::numeric ELSE score END,
	              notes = CASE WHEN $6::text IS NOT NULL THEN $6::text ELSE notes END,
	              self_appraisal = CASE WHEN $7::text IS NOT NULL THEN $7::text ELSE self_appraisal END,
	              self_score = CASE WHEN $8::numeric IS NOT NULL THEN $8::numeric ELSE self_score END,
	              peer_feedback = CASE WHEN $9::text IS NOT NULL THEN $9::text ELSE peer_feedback END,
	              peer_score = CASE WHEN $10::numeric IS NOT NULL THEN $10::numeric ELSE peer_score END,
	              peer_reviewer_id = CASE WHEN $11::text IS NOT NULL THEN $11::uuid ELSE peer_reviewer_id END,
	              manager_assessment = CASE WHEN $12::text IS NOT NULL THEN $12::text ELSE manager_assessment END,
	              manager_score = CASE WHEN $13::numeric IS NOT NULL THEN $13::numeric ELSE manager_score END,
	              kpi_ratings = CASE WHEN $14::text IS NOT NULL THEN $14::jsonb ELSE kpi_ratings END,
	              wellness_audit_score = CASE WHEN $15::numeric IS NOT NULL THEN $15::numeric ELSE wellness_audit_score END,
	              habitat_recommendations = CASE WHEN $16::text IS NOT NULL THEN $16::text ELSE habitat_recommendations END,
	              clinical_auditor_id = CASE WHEN $17::text IS NOT NULL THEN $17::uuid ELSE clinical_auditor_id END,
	              vhr_record_id = CASE WHEN $18::text IS NOT NULL THEN $18::uuid ELSE vhr_record_id END,
	              completed_at = CASE WHEN $19::timestamptz IS NOT NULL THEN $19::timestamptz ELSE completed_at END,
	              updated_at = CURRENT_TIMESTAMP
	          WHERE review_id::text = $1
	          RETURNING review_id::text, employee_id::text, review_type, review_period, phase, scheduled_for::text, status,
	                    reviewer_id::text, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
	                    peer_reviewer_id::text, manager_assessment, manager_score, kpi_ratings::text,
	                    wellness_audit_score, habitat_recommendations, clinical_auditor_id::text, vhr_record_id::text,
	                    completed_at, created_at, updated_at`
	row := r.db.QueryRow(ctx, query,
		rc.ReviewID, rc.Status, rc.Phase, rc.ReviewerID, rc.Score, rc.Notes,
		rc.SelfAppraisal, rc.SelfScore, rc.PeerFeedback, rc.PeerScore,
		rc.PeerReviewerID, rc.ManagerAssessment, rc.ManagerScore, rc.KPIRatings,
		rc.WellnessAuditScore, rc.HabitatRecommendations, rc.ClinicalAuditorID, rc.VHRRecordID,
		rc.CompletedAt,
	)
	res, err := scanReviewCycle(row)
	if err != nil {
		return nil, fmt.Errorf("review cycle with id %s not found: %w", rc.ReviewID, err)
	}
	return res, nil
}

func (r *pgxRepository) CreateWorkplaceIncident(ctx context.Context, inc *WorkplaceIncident) (*WorkplaceIncident, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.incidents (
		incident_id, title, category, involved_feline_id, involved_human_id, severity, status, description, resolution_notes
	) VALUES (
		COALESCE(NULLIF($1, '')::uuid, gen_random_uuid_v7()), $2, $3, $4, $5, COALESCE(NULLIF($6, ''), 'low'), COALESCE(NULLIF($7, ''), 'open'), $8, $9
	) RETURNING incident_id::text, title, category, involved_feline_id::text, involved_human_id::text, severity, status, description, resolution_notes, reported_at, resolved_at`
	var res WorkplaceIncident
	var fID, hID *string
	err := r.db.QueryRow(ctx, query, inc.IncidentID, inc.Title, inc.Category, inc.InvolvedFelineID, inc.InvolvedHumanID, inc.Severity, inc.Status, inc.Description, inc.ResolutionNotes).Scan(
		&res.IncidentID, &res.Title, &res.Category, &fID, &hID, &res.Severity, &res.Status, &res.Description, &res.ResolutionNotes, &res.ReportedAt, &res.ResolvedAt,
	)
	if err != nil {
		return nil, err
	}
	res.InvolvedFelineID = fID
	res.InvolvedHumanID = hID
	return &res, nil
}

func (r *pgxRepository) ListWorkplaceIncidents(ctx context.Context, status string) ([]*WorkplaceIncident, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT incident_id::text, title, category, involved_feline_id::text, involved_human_id::text, severity, status, description, resolution_notes, reported_at, resolved_at
	          FROM workforce.incidents
	          WHERE ($1 = '' OR status = $1)
	          ORDER BY reported_at DESC`
	rows, err := r.db.Query(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*WorkplaceIncident
	for rows.Next() {
		var res WorkplaceIncident
		var fID, hID *string
		if err := rows.Scan(&res.IncidentID, &res.Title, &res.Category, &fID, &hID, &res.Severity, &res.Status, &res.Description, &res.ResolutionNotes, &res.ReportedAt, &res.ResolvedAt); err != nil {
			return nil, err
		}
		res.InvolvedFelineID = fID
		res.InvolvedHumanID = hID
		list = append(list, &res)
	}
	if list == nil {
		list = []*WorkplaceIncident{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) UpdateWorkplaceIncidentStatus(ctx context.Context, id, status, resolutionNotes string) (*WorkplaceIncident, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	switch status {
	case "open", "under_review", "resolved":
	default:
		return nil, fmt.Errorf("invalid incident status %q: status must be one of 'open', 'under_review', 'resolved'", status)
	}

	query := `UPDATE workforce.incidents
	          SET status = $2,
	              resolution_notes = CASE WHEN $3 <> '' THEN $3 ELSE resolution_notes END,
	              resolved_at = CASE WHEN $2 = 'resolved' THEN CURRENT_TIMESTAMP ELSE resolved_at END
	          WHERE incident_id::text = $1
	          RETURNING incident_id::text, title, category, involved_feline_id::text, involved_human_id::text, severity, status, description, resolution_notes, reported_at, resolved_at`
	var res WorkplaceIncident
	var fID, hID *string
	err := r.db.QueryRow(ctx, query, id, status, resolutionNotes).Scan(
		&res.IncidentID, &res.Title, &res.Category, &fID, &hID, &res.Severity, &res.Status, &res.Description, &res.ResolutionNotes, &res.ReportedAt, &res.ResolvedAt,
	)
	if err != nil {
		return nil, err
	}
	res.InvolvedFelineID = fID
	res.InvolvedHumanID = hID
	return &res, nil
}

