package workforce

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVHRRecords(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	t.Run("Create with Auto-Generated ID and Optional Fields", func(t *testing.T) {
		vetID := "emp-human-elena"
		prescriptions := "Amoxicillin 50mg daily"
		rec := &VHRRecord{
			FelineID:          "emp-feline-garfield",
			VeterinarianID:    &vetID,
			VisitDate:         "2026-11-01",
			WeightKG:          5.4,
			DentalScore:       9,
			VaccinationStatus: "up_to_date",
			Prescriptions:     &prescriptions,
			ClinicalNotes:     "Excellent wellness checkup and routine teeth cleaning",
		}

		created, err := repo.CreateVHRRecord(ctx, rec)
		require.NoError(t, err)
		assert.NotEmpty(t, created.RecordID)
		assert.Equal(t, "emp-feline-garfield", created.FelineID)
		assert.NotNil(t, created.VeterinarianID)
		assert.Equal(t, vetID, *created.VeterinarianID)
		assert.Equal(t, "2026-11-01", created.VisitDate)
		assert.Equal(t, 5.4, created.WeightKG)
		assert.Equal(t, 9, created.DentalScore)
		assert.Equal(t, "up_to_date", created.VaccinationStatus)
		assert.NotNil(t, created.Prescriptions)
		assert.Equal(t, prescriptions, *created.Prescriptions)
		assert.Equal(t, "Excellent wellness checkup and routine teeth cleaning", created.ClinicalNotes)
		assert.False(t, created.CreatedAt.IsZero())
		assert.False(t, created.UpdatedAt.IsZero())
	})

	t.Run("Create with Explicit Custom ID and Nil Optional Fields", func(t *testing.T) {
		customID := "vhr-custom-999"
		rec := &VHRRecord{
			RecordID:          customID,
			FelineID:          "emp-feline-barneby",
			VisitDate:         "2026-11-02",
			WeightKG:          4.8,
			DentalScore:       8,
			VaccinationStatus: "pending_booster",
			ClinicalNotes:     "Routine health check, booster needed",
		}

		created, err := repo.CreateVHRRecord(ctx, rec)
		require.NoError(t, err)
		assert.Equal(t, customID, created.RecordID)
		assert.Equal(t, "emp-feline-barneby", created.FelineID)
		assert.Nil(t, created.VeterinarianID)
		assert.Nil(t, created.Prescriptions)
		assert.False(t, created.CreatedAt.IsZero())
		assert.False(t, created.UpdatedAt.IsZero())
	})

	t.Run("List and Filter VHR Records", func(t *testing.T) {
		r := NewMockRepository()
		vetID := "emp-human-elena"
		prescriptions := "Amoxicillin 50mg"

		// Create record 1 for Garfield (2026-11-01)
		rec1 := &VHRRecord{
			FelineID:          "emp-feline-garfield",
			VeterinarianID:    &vetID,
			VisitDate:         "2026-11-01",
			WeightKG:          5.4,
			DentalScore:       9,
			VaccinationStatus: "up_to_date",
			ClinicalNotes:     "Garfield Visit 1",
		}
		c1, err := r.CreateVHRRecord(ctx, rec1)
		require.NoError(t, err)

		// Create record 2 for Garfield (2026-11-10) - later date with prescription
		rec2 := &VHRRecord{
			FelineID:          "emp-feline-garfield",
			VeterinarianID:    &vetID,
			VisitDate:         "2026-11-10",
			WeightKG:          5.5,
			DentalScore:       9,
			VaccinationStatus: "up_to_date",
			Prescriptions:     &prescriptions,
			ClinicalNotes:     "Garfield Visit 2",
		}
		c2, err := r.CreateVHRRecord(ctx, rec2)
		require.NoError(t, err)

		// Create record 3 for Barneby (2026-11-05)
		rec3 := &VHRRecord{
			FelineID:          "emp-feline-barneby",
			VeterinarianID:    &vetID,
			VisitDate:         "2026-11-05",
			WeightKG:          4.9,
			DentalScore:       8,
			VaccinationStatus: "up_to_date",
			ClinicalNotes:     "Barneby Visit 1",
		}
		c3, err := r.CreateVHRRecord(ctx, rec3)
		require.NoError(t, err)

		// Filter for Garfield
		garfieldRecords, err := r.ListVHRRecords(ctx, "emp-feline-garfield")
		require.NoError(t, err)
		assert.Len(t, garfieldRecords, 2)
		// Should be sorted by VisitDate DESC (c2: 2026-11-10 before c1: 2026-11-01)
		assert.Equal(t, c2.RecordID, garfieldRecords[0].RecordID)
		assert.Equal(t, "2026-11-10", garfieldRecords[0].VisitDate)
		assert.Equal(t, 5.5, garfieldRecords[0].WeightKG)
		assert.Equal(t, &prescriptions, garfieldRecords[0].Prescriptions)

		assert.Equal(t, c1.RecordID, garfieldRecords[1].RecordID)
		assert.Equal(t, "2026-11-01", garfieldRecords[1].VisitDate)
		assert.Equal(t, 5.4, garfieldRecords[1].WeightKG)

		// Filter for Barneby
		barnebyRecords, err := r.ListVHRRecords(ctx, "emp-feline-barneby")
		require.NoError(t, err)
		assert.Len(t, barnebyRecords, 1)
		assert.Equal(t, c3.RecordID, barnebyRecords[0].RecordID)
		assert.Equal(t, 4.9, barnebyRecords[0].WeightKG)

		// Empty feline ID should list all records sorted by VisitDate DESC
		allRecords, err := r.ListVHRRecords(ctx, "")
		require.NoError(t, err)
		assert.Len(t, allRecords, 3)
		assert.Equal(t, "2026-11-10", allRecords[0].VisitDate)
		assert.Equal(t, "2026-11-05", allRecords[1].VisitDate)
		assert.Equal(t, "2026-11-01", allRecords[2].VisitDate)

		// Query for a feline with no records
		noRecords, err := r.ListVHRRecords(ctx, "emp-feline-nonexistent")
		require.NoError(t, err)
		assert.NotNil(t, noRecords)
		assert.Empty(t, noRecords)
	})
}

func TestWorkplaceIncidents(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	felineID := "emp-feline-garfield"
	humanID := "emp-human-alice"

	// Create Workplace Incident
	newInc := &WorkplaceIncident{
		Title:            "Laser Pointer Interference",
		Category:         "safety",
		InvolvedFelineID: &felineID,
		InvolvedHumanID:  &humanID,
		Severity:         "medium",
		Description:      "Uncalibrated laser beam temporarily disrupted feline observation focus",
	}
	created, err := repo.CreateWorkplaceIncident(ctx, newInc)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.IncidentID)
	assert.Equal(t, "open", created.Status)
	assert.Equal(t, "medium", created.Severity)
	assert.False(t, created.ReportedAt.IsZero())

	// List Workplace Incidents
	incidents, err := repo.ListWorkplaceIncidents(ctx, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, incidents)

	openIncidents, err := repo.ListWorkplaceIncidents(ctx, "open")
	assert.NoError(t, err)
	assert.NotEmpty(t, openIncidents)
	for _, inc := range openIncidents {
		assert.Equal(t, "open", inc.Status)
	}

	// Update Workplace Incident Status to under_review
	reviewNotes := "Safety committee investigating optical dispersion pattern"
	underReview, err := repo.UpdateWorkplaceIncidentStatus(ctx, created.IncidentID, "under_review", reviewNotes)
	assert.NoError(t, err)
	assert.Equal(t, "under_review", underReview.Status)
	assert.NotNil(t, underReview.ResolutionNotes)
	assert.Equal(t, reviewNotes, *underReview.ResolutionNotes)
	assert.Nil(t, underReview.ResolvedAt)

	// Update Workplace Incident Status to resolved
	resolveNotes := "Laser pointer recalibrated and safety barrier installed"
	resolved, err := repo.UpdateWorkplaceIncidentStatus(ctx, created.IncidentID, "resolved", resolveNotes)
	assert.NoError(t, err)
	assert.Equal(t, "resolved", resolved.Status)
	assert.NotNil(t, resolved.ResolutionNotes)
	assert.Equal(t, resolveNotes, *resolved.ResolutionNotes)
	assert.NotNil(t, resolved.ResolvedAt)

	// Error handling: invalid status
	_, err = repo.UpdateWorkplaceIncidentStatus(ctx, created.IncidentID, "invalid_status", "Notes")
	assert.Error(t, err)

	// Error handling: non-existent ID
	_, err = repo.UpdateWorkplaceIncidentStatus(ctx, "non-existent-inc-id", "resolved", "Notes")
	assert.Error(t, err)
}

func TestWorkforcePgxRepository_NilDB(t *testing.T) {
	repo := &pgxRepository{db: nil}
	ctx := context.Background()

	_, err := repo.GetEmployeeByID(ctx, "any")
	assert.Error(t, err)

	_, err = repo.ListEmployees(ctx, "", "")
	assert.Error(t, err)

	_, err = repo.CreateEmployee(ctx, &Employee{})
	assert.Error(t, err)

	_, err = repo.UpdateEmployeeStatus(ctx, "any", "active")
	assert.Error(t, err)

	_, err = repo.CreateOnboardingTask(ctx, &OnboardingTask{})
	assert.Error(t, err)

	_, err = repo.ListOnboardingChecklist(ctx, "any")
	assert.Error(t, err)

	_, err = repo.UpdateOnboardingTask(ctx, "any", true)
	assert.Error(t, err)

	_, err = repo.GetCareScheduleByFelineID(ctx, "any")
	assert.Error(t, err)

	_, err = repo.UpsertCareSchedule(ctx, &CareSchedule{})
	assert.Error(t, err)

	_, err = repo.SetEmergencyMedicalHold(ctx, "any", true)
	assert.Error(t, err)

	_, err = repo.CreateLeaveRequest(ctx, &LeaveRequest{})
	assert.Error(t, err)

	_, err = repo.GetLeaveRequestByID(ctx, "any")
	assert.Error(t, err)

	_, err = repo.ListLeaveRequests(ctx, "", "")
	assert.Error(t, err)

	_, err = repo.UpdateLeaveRequestStatus(ctx, "any", "approved")
	assert.Error(t, err)

	_, err = repo.CreateReviewCycle(ctx, &ReviewCycle{})
	assert.Error(t, err)

	_, err = repo.GetReviewCycleByID(ctx, "any")
	assert.Error(t, err)

	_, err = repo.ListReviewCycles(ctx, "", "", "")
	assert.Error(t, err)

	_, err = repo.UpdateReviewCycle(ctx, &ReviewCycle{})
	assert.Error(t, err)

	_, err = repo.CreateVHRRecord(ctx, &VHRRecord{})
	assert.Error(t, err)

	_, err = repo.ListVHRRecords(ctx, "any")
	assert.Error(t, err)

	_, err = repo.CreateWorkplaceIncident(ctx, &WorkplaceIncident{})
	assert.Error(t, err)

	_, err = repo.ListWorkplaceIncidents(ctx, "")
	assert.Error(t, err)

	_, err = repo.UpdateWorkplaceIncidentStatus(ctx, "any", "resolved", "notes")
	assert.Error(t, err)

	_, err = repo.CreateExecutiveReport(ctx, &ExecutiveReport{})
	assert.Error(t, err)

	_, err = repo.GetExecutiveReportByID(ctx, "any")
	assert.Error(t, err)

	_, err = repo.GetExecutiveReportByPeriod(ctx, "any")
	assert.Error(t, err)

	_, err = repo.ListExecutiveReports(ctx, 10, 0)
	assert.Error(t, err)

	_, err = repo.UpdateExecutiveReportStatus(ctx, "any", "archived", nil, nil)
	assert.Error(t, err)
}

func TestExecutiveReports(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	// Verify pre-seeded reports
	reports, err := repo.ListExecutiveReports(ctx, 10, 0)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(reports), 2)

	// Get by Period
	rep09, err := repo.GetExecutiveReportByPeriod(ctx, "2026-09")
	assert.NoError(t, err)
	assert.Equal(t, "2026-09", rep09.Period)
	assert.Equal(t, 1240000.00, rep09.TotalRevenue)
	assert.Equal(t, 12, rep09.TotalHeadcount)

	// Create new Executive Report
	pdfPath := "/remote.php/dav/files/admin/Executive/Reports/2026-10/executive_summary_report_2026-10.pdf"
	csvPath := "/remote.php/dav/files/admin/Executive/Reports/2026-10/executive_summary_report_2026-10.csv"
	newRep := &ExecutiveReport{
		ReportID:           "rep-2026-10",
		Period:             "2026-10",
		Title:              "OCI Executive Management Report - October 2026",
		Status:             "archived",
		TotalRevenue:       1350000.00,
		OperatingExpenses:  320000.00,
		PayrollExpenses:    430000.00,
		NetProfit:          600000.00,
		OperationalBurn:    750000.00,
		CashReserves:       9800000.00,
		RunwayMonths:       13.06,
		TotalHeadcount:     12,
		HumanHeadcount:     10,
		FelineHeadcount:    2,
		AttendanceRate:     99.40,
		OvertimeHours:      44.00,
		TimesheetsApproved: 19,
		WebDAVPathPDF:      &pdfPath,
		WebDAVPathCSV:      &csvPath,
		GeneratedBy:        "system-reporting-worker",
	}

	created, err := repo.CreateExecutiveReport(ctx, newRep)
	assert.NoError(t, err)
	assert.Equal(t, "rep-2026-10", created.ReportID)
	assert.Equal(t, "archived", created.Status)

	// Get by ID
	fetched, err := repo.GetExecutiveReportByID(ctx, "rep-2026-10")
	assert.NoError(t, err)
	assert.Equal(t, created.Title, fetched.Title)

	// Update status
	updated, err := repo.UpdateExecutiveReportStatus(ctx, "rep-2026-10", "archived", &pdfPath, &csvPath)
	assert.NoError(t, err)
	assert.Equal(t, "archived", updated.Status)
}
