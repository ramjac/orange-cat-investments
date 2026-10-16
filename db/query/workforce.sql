-- name: GetEmployeeByID :one
SELECT employee_id, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at
FROM workforce.employees
WHERE employee_id = $1;

-- name: ListEmployees :many
SELECT employee_id, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at
FROM workforce.employees
ORDER BY created_at DESC;

-- name: CreateEmployee :one
INSERT INTO workforce.employees (
    employee_id, employee_type, first_name, last_name, email, role_title, department, status, hired_at
) VALUES (
    COALESCE($1, gen_random_uuid_v7()), $2, $3, $4, $5, $6, $7, COALESCE($8, 'onboarding'), COALESCE($9, CURRENT_TIMESTAMP)
) RETURNING employee_id, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at;

-- name: UpdateEmployeeStatus :one
UPDATE workforce.employees
SET status = $2, updated_at = CURRENT_TIMESTAMP
WHERE employee_id = $1
RETURNING employee_id, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at;

-- name: CreateOnboardingTask :one
INSERT INTO workforce.onboarding_checklists (
    employee_id, task_name, category, assigned_to
) VALUES (
    $1, $2, $3, $4
) RETURNING checklist_id, employee_id, task_name, category, is_completed, completed_at, assigned_to, created_at, updated_at;

-- name: ListOnboardingChecklist :many
SELECT checklist_id, employee_id, task_name, category, is_completed, completed_at, assigned_to, created_at, updated_at
FROM workforce.onboarding_checklists
WHERE employee_id = $1
ORDER BY created_at ASC;

-- name: UpdateOnboardingTask :one
UPDATE workforce.onboarding_checklists
SET is_completed = $2, completed_at = CASE WHEN $2 = TRUE THEN CURRENT_TIMESTAMP ELSE NULL END, updated_at = CURRENT_TIMESTAMP
WHERE checklist_id = $1
RETURNING checklist_id, employee_id, task_name, category, is_completed, completed_at, assigned_to, created_at, updated_at;

-- name: GetCareScheduleByFelineID :one
SELECT schedule_id, feline_id, dietary_plan, feeding_times, special_medical_needs, preferred_perch_zone, emergency_medical_hold, caretaker_id, created_at, updated_at
FROM workforce.care_schedules
WHERE feline_id = $1;

-- name: UpsertCareSchedule :one
INSERT INTO workforce.care_schedules (
    feline_id, dietary_plan, feeding_times, special_medical_needs, preferred_perch_zone, emergency_medical_hold, caretaker_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (schedule_id) DO UPDATE
SET dietary_plan = EXCLUDED.dietary_plan,
    feeding_times = EXCLUDED.feeding_times,
    special_medical_needs = EXCLUDED.special_medical_needs,
    preferred_perch_zone = EXCLUDED.preferred_perch_zone,
    emergency_medical_hold = EXCLUDED.emergency_medical_hold,
    caretaker_id = EXCLUDED.caretaker_id,
    updated_at = CURRENT_TIMESTAMP
RETURNING schedule_id, feline_id, dietary_plan, feeding_times, special_medical_needs, preferred_perch_zone, emergency_medical_hold, caretaker_id, created_at, updated_at;

-- name: UpdateEmergencyMedicalHold :one
UPDATE workforce.care_schedules
SET emergency_medical_hold = $2, updated_at = CURRENT_TIMESTAMP
WHERE feline_id = $1
RETURNING schedule_id, feline_id, dietary_plan, feeding_times, special_medical_needs, preferred_perch_zone, emergency_medical_hold, caretaker_id, created_at, updated_at;

-- name: CreateLeaveRequest :one
INSERT INTO workforce.leave_requests (
    leave_id, employee_id, leave_type, start_date, end_date, status, reason
) VALUES (
    COALESCE($1, gen_random_uuid_v7()), $2, $3, $4, $5, COALESCE($6, 'pending'), $7
) RETURNING leave_id, employee_id, leave_type, start_date, end_date, status, reason, created_at, updated_at;

-- name: GetLeaveRequestByID :one
SELECT leave_id, employee_id, leave_type, start_date, end_date, status, reason, created_at, updated_at
FROM workforce.leave_requests
WHERE leave_id = $1;

-- name: ListLeaveRequests :many
SELECT leave_id, employee_id, leave_type, start_date, end_date, status, reason, created_at, updated_at
FROM workforce.leave_requests
WHERE ($1::uuid IS NULL OR employee_id = $1)
  AND ($2::text = '' OR status = $2)
ORDER BY created_at DESC;

-- name: UpdateLeaveRequestStatus :one
UPDATE workforce.leave_requests
SET status = $2, updated_at = CURRENT_TIMESTAMP
WHERE leave_id = $1
RETURNING leave_id, employee_id, leave_type, start_date, end_date, status, reason, created_at, updated_at;

-- name: CreateReviewCycle :one
INSERT INTO workforce.review_cycles (
    review_id, employee_id, review_type, review_period, phase, scheduled_for, status,
    reviewer_id, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
    peer_reviewer_id, manager_assessment, manager_score, kpi_ratings,
    wellness_audit_score, habitat_recommendations, clinical_auditor_id, vhr_record_id, completed_at
) VALUES (
    COALESCE($1, gen_random_uuid_v7()), $2, $3, COALESCE($4, '2026-Q3'), COALESCE($5, 'self_appraisal'),
    $6, COALESCE($7, 'scheduled'), $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
) RETURNING review_id, employee_id, review_type, review_period, phase, scheduled_for, status,
            reviewer_id, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
            peer_reviewer_id, manager_assessment, manager_score, kpi_ratings,
            wellness_audit_score, habitat_recommendations, clinical_auditor_id, vhr_record_id,
            completed_at, created_at, updated_at;

-- name: GetReviewCycleByID :one
SELECT review_id, employee_id, review_type, review_period, phase, scheduled_for, status,
       reviewer_id, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
       peer_reviewer_id, manager_assessment, manager_score, kpi_ratings,
       wellness_audit_score, habitat_recommendations, clinical_auditor_id, vhr_record_id,
       completed_at, created_at, updated_at
FROM workforce.review_cycles
WHERE review_id = $1;

-- name: ListReviewCycles :many
SELECT review_id, employee_id, review_type, review_period, phase, scheduled_for, status,
       reviewer_id, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
       peer_reviewer_id, manager_assessment, manager_score, kpi_ratings,
       wellness_audit_score, habitat_recommendations, clinical_auditor_id, vhr_record_id,
       completed_at, created_at, updated_at
FROM workforce.review_cycles
WHERE ($1::uuid IS NULL OR employee_id = $1)
  AND ($2::text = '' OR status = $2)
  AND ($3::text = '' OR review_type = $3)
ORDER BY scheduled_for DESC, created_at DESC;

-- name: UpdateReviewCycle :one
UPDATE workforce.review_cycles
SET status = COALESCE($2, status),
    phase = COALESCE($3, phase),
    reviewer_id = COALESCE($4, reviewer_id),
    score = COALESCE($5, score),
    notes = COALESCE($6, notes),
    self_appraisal = COALESCE($7, self_appraisal),
    self_score = COALESCE($8, self_score),
    peer_feedback = COALESCE($9, peer_feedback),
    peer_score = COALESCE($10, peer_score),
    peer_reviewer_id = COALESCE($11, peer_reviewer_id),
    manager_assessment = COALESCE($12, manager_assessment),
    manager_score = COALESCE($13, manager_score),
    kpi_ratings = COALESCE($14, kpi_ratings),
    wellness_audit_score = COALESCE($15, wellness_audit_score),
    habitat_recommendations = COALESCE($16, habitat_recommendations),
    clinical_auditor_id = COALESCE($17, clinical_auditor_id),
    vhr_record_id = COALESCE($18, vhr_record_id),
    completed_at = COALESCE($19, completed_at),
    updated_at = CURRENT_TIMESTAMP
WHERE review_id = $1
RETURNING review_id, employee_id, review_type, review_period, phase, scheduled_for, status,
          reviewer_id, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
          peer_reviewer_id, manager_assessment, manager_score, kpi_ratings,
          wellness_audit_score, habitat_recommendations, clinical_auditor_id, vhr_record_id,
          completed_at, created_at, updated_at;

-- name: CreateExecutiveReport :one
INSERT INTO workforce.executive_reports (
    report_id, period, title, status,
    total_revenue, operating_expenses, payroll_expenses, net_profit,
    operational_burn, cash_reserves, runway_months,
    total_headcount, human_headcount, feline_headcount,
    attendance_rate, overtime_hours, timesheets_approved,
    webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by
) VALUES (
    $1, $2, $3, COALESCE($4, 'archived'),
    $5, $6, $7, $8,
    $9, $10, $11,
    $12, $13, $14,
    $15, $16, $17,
    $18, $19, $20, $21, COALESCE($22, 'system-reporting-worker')
) RETURNING *;

-- name: GetExecutiveReportByID :one
SELECT * FROM workforce.executive_reports
WHERE report_id = $1;

-- name: GetExecutiveReportByPeriod :one
SELECT * FROM workforce.executive_reports
WHERE period = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: ListExecutiveReports :many
SELECT * FROM workforce.executive_reports
ORDER BY period DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateExecutiveReportStatus :one
UPDATE workforce.executive_reports
SET status = $2,
    webdav_path_pdf = COALESCE($3, webdav_path_pdf),
    webdav_path_csv = COALESCE($4, webdav_path_csv),
    updated_at = CURRENT_TIMESTAMP
WHERE report_id = $1
RETURNING *;


