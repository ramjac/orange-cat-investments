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
    review_id, employee_id, review_type, scheduled_for, status, reviewer_id, score, notes, completed_at
) VALUES (
    COALESCE($1, gen_random_uuid_v7()), $2, $3, $4, COALESCE($5, 'scheduled'), $6, $7, $8, $9
) RETURNING review_id, employee_id, review_type, scheduled_for, status, reviewer_id, score, notes, completed_at, created_at, updated_at;

-- name: GetReviewCycleByID :one
SELECT review_id, employee_id, review_type, scheduled_for, status, reviewer_id, score, notes, completed_at, created_at, updated_at
FROM workforce.review_cycles
WHERE review_id = $1;

-- name: ListReviewCycles :many
SELECT review_id, employee_id, review_type, scheduled_for, status, reviewer_id, score, notes, completed_at, created_at, updated_at
FROM workforce.review_cycles
WHERE ($1::uuid IS NULL OR employee_id = $1)
  AND ($2::text = '' OR status = $2)
  AND ($3::text = '' OR review_type = $3)
ORDER BY scheduled_for DESC, created_at DESC;

-- name: UpdateReviewCycle :one
UPDATE workforce.review_cycles
SET status = COALESCE($2, status),
    reviewer_id = COALESCE($3, reviewer_id),
    score = COALESCE($4, score),
    notes = COALESCE($5, notes),
    completed_at = COALESCE($6, completed_at),
    updated_at = CURRENT_TIMESTAMP
WHERE review_id = $1
RETURNING review_id, employee_id, review_type, scheduled_for, status, reviewer_id, score, notes, completed_at, created_at, updated_at;

