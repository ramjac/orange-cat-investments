package workforce

import (
	"context"
	"fmt"
	"strings"
)

func (r *pgxRepository) GetEmployeeByID(ctx context.Context, id string) (*Employee, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	exactQuery := `SELECT employee_id::text, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at
	          FROM workforce.employees
	          WHERE employee_id::text = $1 OR lower(email) = lower($1)`
	var emp Employee
	err := r.db.QueryRow(ctx, exactQuery, id).Scan(
		&emp.EmployeeID, &emp.EmployeeType, &emp.FirstName, &emp.LastName, &emp.Email,
		&emp.RoleTitle, &emp.Department, &emp.Status, &emp.HiredAt, &emp.CreatedAt, &emp.UpdatedAt,
	)
	if err == nil {
		return &emp, nil
	}

	cleanName := strings.TrimPrefix(strings.TrimPrefix(id, "emp-feline-"), "emp-human-")
	aliasQuery := `SELECT employee_id::text, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at
	          FROM workforce.employees
	          WHERE lower(first_name) = lower($1)
	          LIMIT 2`
	rows, err := r.db.Query(ctx, aliasQuery, cleanName)
	if err != nil {
		return nil, fmt.Errorf("failed searching employee alias: %w", err)
	}
	defer rows.Close()

	var matches []*Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.EmployeeID, &e.EmployeeType, &e.FirstName, &e.LastName, &e.Email, &e.RoleTitle, &e.Department, &e.Status, &e.HiredAt, &e.CreatedAt, &e.UpdatedAt); err == nil {
			matches = append(matches, &e)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("employee with id %s not found", id)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous employee alias %s matches multiple employees", id)
	}
	return matches[0], nil
}

func (r *pgxRepository) ListEmployees(ctx context.Context, employeeType, status string) ([]*Employee, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT employee_id::text, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at
	          FROM workforce.employees
	          WHERE ($1 = '' OR employee_type = $1)
	            AND ($2 = '' OR status = $2)
	          ORDER BY hired_at ASC, created_at ASC`
	rows, err := r.db.Query(ctx, query, employeeType, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Employee
	for rows.Next() {
		var emp Employee
		if err := rows.Scan(
			&emp.EmployeeID, &emp.EmployeeType, &emp.FirstName, &emp.LastName, &emp.Email,
			&emp.RoleTitle, &emp.Department, &emp.Status, &emp.HiredAt, &emp.CreatedAt, &emp.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &emp)
	}
	if list == nil {
		list = []*Employee{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) CreateEmployee(ctx context.Context, emp *Employee) (*Employee, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.employees (
		employee_id, employee_type, first_name, last_name, email, role_title, department, status, hired_at
	) VALUES (
		COALESCE(NULLIF($1, '')::uuid, gen_random_uuid_v7()), $2, $3, $4, $5, $6, $7,
		COALESCE(NULLIF($8, ''), 'onboarding'),
		COALESCE(NULLIF($9::timestamptz, '0001-01-01 00:00:00+00'::timestamptz), CURRENT_TIMESTAMP)
	) RETURNING employee_id::text, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at`
	var res Employee
	err := r.db.QueryRow(ctx, query,
		emp.EmployeeID, emp.EmployeeType, emp.FirstName, emp.LastName, emp.Email,
		emp.RoleTitle, emp.Department, emp.Status, emp.HiredAt,
	).Scan(
		&res.EmployeeID, &res.EmployeeType, &res.FirstName, &res.LastName, &res.Email,
		&res.RoleTitle, &res.Department, &res.Status, &res.HiredAt, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *pgxRepository) UpdateEmployeeStatus(ctx context.Context, id, status string) (*Employee, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE workforce.employees
	          SET status = $2, updated_at = CURRENT_TIMESTAMP
	          WHERE employee_id::text = $1
	          RETURNING employee_id::text, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at`
	var res Employee
	err := r.db.QueryRow(ctx, query, id, status).Scan(
		&res.EmployeeID, &res.EmployeeType, &res.FirstName, &res.LastName, &res.Email,
		&res.RoleTitle, &res.Department, &res.Status, &res.HiredAt, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("employee with id %s not found: %w", id, err)
	}
	return &res, nil
}

func (r *pgxRepository) CreateOnboardingTask(ctx context.Context, task *OnboardingTask) (*OnboardingTask, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.onboarding_checklists (
		checklist_id, employee_id, task_name, category, is_completed, completed_at, assigned_to
	) VALUES (
		COALESCE(NULLIF($1, '')::uuid, gen_random_uuid_v7()), $2::uuid, $3, $4, $5, $6, NULLIF($7, '')::uuid
	) RETURNING checklist_id::text, employee_id::text, task_name, category, is_completed, completed_at, assigned_to::text, created_at, updated_at`
	var res OnboardingTask
	var assignedTo *string
	err := r.db.QueryRow(ctx, query,
		task.ChecklistID, task.EmployeeID, task.TaskName, task.Category, task.IsCompleted, task.CompletedAt, task.AssignedTo,
	).Scan(
		&res.ChecklistID, &res.EmployeeID, &res.TaskName, &res.Category, &res.IsCompleted, &res.CompletedAt, &assignedTo, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	res.AssignedTo = assignedTo
	return &res, nil
}

func (r *pgxRepository) ListOnboardingChecklist(ctx context.Context, employeeID string) ([]*OnboardingTask, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT checklist_id::text, employee_id::text, task_name, category, is_completed, completed_at, assigned_to::text, created_at, updated_at
	          FROM workforce.onboarding_checklists
	          WHERE employee_id::text = $1
	          ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, query, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*OnboardingTask
	for rows.Next() {
		var res OnboardingTask
		var assignedTo *string
		if err := rows.Scan(&res.ChecklistID, &res.EmployeeID, &res.TaskName, &res.Category, &res.IsCompleted, &res.CompletedAt, &assignedTo, &res.CreatedAt, &res.UpdatedAt); err != nil {
			return nil, err
		}
		res.AssignedTo = assignedTo
		list = append(list, &res)
	}
	if list == nil {
		list = []*OnboardingTask{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) UpdateOnboardingTask(ctx context.Context, taskID string, isCompleted bool) (*OnboardingTask, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE workforce.onboarding_checklists
	          SET is_completed = $2,
	              completed_at = CASE WHEN $2 = TRUE THEN CURRENT_TIMESTAMP ELSE NULL END,
	              updated_at = CURRENT_TIMESTAMP
	          WHERE checklist_id::text = $1
	          RETURNING checklist_id::text, employee_id::text, task_name, category, is_completed, completed_at, assigned_to::text, created_at, updated_at`
	var res OnboardingTask
	var assignedTo *string
	err := r.db.QueryRow(ctx, query, taskID, isCompleted).Scan(
		&res.ChecklistID, &res.EmployeeID, &res.TaskName, &res.Category, &res.IsCompleted, &res.CompletedAt, &assignedTo, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("task with id %s not found: %w", taskID, err)
	}
	res.AssignedTo = assignedTo
	return &res, nil
}

func (r *pgxRepository) GetCareScheduleByFelineID(ctx context.Context, felineID string) (*CareSchedule, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	felineUUID := felineID
	if emp, err := r.GetEmployeeByID(ctx, felineID); err == nil && emp != nil {
		felineUUID = emp.EmployeeID
	}

	query := `SELECT cs.schedule_id::text, cs.feline_id::text, cs.dietary_plan, cs.feeding_times, cs.special_medical_needs, cs.preferred_perch_zone, cs.emergency_medical_hold, cs.caretaker_id::text, cs.created_at, cs.updated_at
	          FROM workforce.care_schedules cs
	          WHERE cs.feline_id::text = $1`
	var cs CareSchedule
	var caretakerID *string
	err := r.db.QueryRow(ctx, query, felineUUID).Scan(
		&cs.ScheduleID, &cs.FelineID, &cs.DietaryPlan, &cs.FeedingTimes, &cs.SpecialMedicalNeeds, &cs.PreferredPerchZone, &cs.EmergencyMedicalHold, &caretakerID, &cs.CreatedAt, &cs.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("care schedule for feline %s not found: %w", felineID, err)
	}
	cs.CaretakerID = caretakerID
	return &cs, nil
}

func (r *pgxRepository) UpsertCareSchedule(ctx context.Context, cs *CareSchedule) (*CareSchedule, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	felineUUID := cs.FelineID
	if emp, err := r.GetEmployeeByID(ctx, cs.FelineID); err == nil && emp != nil {
		felineUUID = emp.EmployeeID
	}

	query := `INSERT INTO workforce.care_schedules (
		schedule_id, feline_id, dietary_plan, feeding_times, special_medical_needs, preferred_perch_zone, emergency_medical_hold, caretaker_id
	) VALUES (
		COALESCE(NULLIF($1, '')::uuid, gen_random_uuid_v7()), $2::uuid, $3, $4, $5, $6, $7, NULLIF($8, '')::uuid
	) ON CONFLICT (feline_id) DO UPDATE SET
		dietary_plan = EXCLUDED.dietary_plan,
		feeding_times = EXCLUDED.feeding_times,
		special_medical_needs = EXCLUDED.special_medical_needs,
		preferred_perch_zone = EXCLUDED.preferred_perch_zone,
		emergency_medical_hold = EXCLUDED.emergency_medical_hold,
		caretaker_id = EXCLUDED.caretaker_id,
		updated_at = CURRENT_TIMESTAMP
	RETURNING schedule_id::text, feline_id::text, dietary_plan, feeding_times, special_medical_needs, preferred_perch_zone, emergency_medical_hold, caretaker_id::text, created_at, updated_at`
	var res CareSchedule
	var caretakerID *string
	err := r.db.QueryRow(ctx, query,
		cs.ScheduleID, felineUUID, cs.DietaryPlan, cs.FeedingTimes, cs.SpecialMedicalNeeds, cs.PreferredPerchZone, cs.EmergencyMedicalHold, cs.CaretakerID,
	).Scan(
		&res.ScheduleID, &res.FelineID, &res.DietaryPlan, &res.FeedingTimes, &res.SpecialMedicalNeeds, &res.PreferredPerchZone, &res.EmergencyMedicalHold, &caretakerID, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	res.CaretakerID = caretakerID
	return &res, nil
}

func (r *pgxRepository) SetEmergencyMedicalHold(ctx context.Context, felineID string, hold bool) (*CareSchedule, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	felineUUID := felineID
	if emp, err := r.GetEmployeeByID(ctx, felineID); err == nil && emp != nil {
		felineUUID = emp.EmployeeID
	}

	query := `INSERT INTO workforce.care_schedules (
		schedule_id, feline_id, dietary_plan, feeding_times, emergency_medical_hold
	) VALUES (
		gen_random_uuid_v7(), $1::uuid, 'Standard feline nutritional plan', ARRAY['09:00 AM', '05:00 PM'], $2
	) ON CONFLICT (feline_id) DO UPDATE SET
		emergency_medical_hold = EXCLUDED.emergency_medical_hold,
		updated_at = CURRENT_TIMESTAMP
	RETURNING schedule_id::text, feline_id::text, dietary_plan, feeding_times, special_medical_needs, preferred_perch_zone, emergency_medical_hold, caretaker_id::text, created_at, updated_at`
	var res CareSchedule
	var caretakerID *string
	err := r.db.QueryRow(ctx, query, felineUUID, hold).Scan(
		&res.ScheduleID, &res.FelineID, &res.DietaryPlan, &res.FeedingTimes, &res.SpecialMedicalNeeds, &res.PreferredPerchZone, &res.EmergencyMedicalHold, &caretakerID, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	res.CaretakerID = caretakerID
	return &res, nil
}

func (r *pgxRepository) CreateVHRRecord(ctx context.Context, rec *VHRRecord) (*VHRRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.vhr_records (
		record_id, feline_id, veterinarian_id, visit_date, weight_kg, dental_score, vaccination_status, prescriptions, clinical_notes
	) VALUES (
		COALESCE(NULLIF($1, '')::uuid, gen_random_uuid_v7()), $2, $3, $4::date, $5, $6, $7, $8, $9
	) RETURNING record_id::text, feline_id::text, veterinarian_id::text, visit_date::text, weight_kg, dental_score, vaccination_status, prescriptions, clinical_notes, created_at, updated_at`
	var res VHRRecord
	var vetID *string
	var vDate string
	err := r.db.QueryRow(ctx, query, rec.RecordID, rec.FelineID, rec.VeterinarianID, rec.VisitDate, rec.WeightKG, rec.DentalScore, rec.VaccinationStatus, rec.Prescriptions, rec.ClinicalNotes).Scan(
		&res.RecordID, &res.FelineID, &vetID, &vDate, &res.WeightKG, &res.DentalScore, &res.VaccinationStatus, &res.Prescriptions, &res.ClinicalNotes, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	res.VeterinarianID = vetID
	res.VisitDate = vDate
	return &res, nil
}

func (r *pgxRepository) ListVHRRecords(ctx context.Context, felineID string) ([]*VHRRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT record_id::text, feline_id::text, veterinarian_id::text, visit_date::text, weight_kg, dental_score, vaccination_status, prescriptions, clinical_notes, created_at, updated_at
	          FROM workforce.vhr_records
	          WHERE ($1 = '' OR feline_id::text = $1)
	          ORDER BY visit_date DESC`
	rows, err := r.db.Query(ctx, query, felineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*VHRRecord
	for rows.Next() {
		var res VHRRecord
		var vetID *string
		var vDate string
		if err := rows.Scan(&res.RecordID, &res.FelineID, &vetID, &vDate, &res.WeightKG, &res.DentalScore, &res.VaccinationStatus, &res.Prescriptions, &res.ClinicalNotes, &res.CreatedAt, &res.UpdatedAt); err != nil {
			return nil, err
		}
		res.VeterinarianID = vetID
		res.VisitDate = vDate
		list = append(list, &res)
	}
	if list == nil {
		list = []*VHRRecord{}
	}
	return list, rows.Err()
}
