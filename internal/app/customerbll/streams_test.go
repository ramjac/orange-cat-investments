package customerbll

import (
	"context"
	"testing"

	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWorkforceForStreams struct {
	workforcev1.WorkforceServiceClient
	employees []*workforcev1.Employee
	records   []*workforcev1.VHRRecord
	incidents []*workforcev1.WorkplaceIncident
}

func (m *mockWorkforceForStreams) ListEmployees(_ context.Context, _ *workforcev1.ListEmployeesRequest) (*workforcev1.ListEmployeesResponse, error) {
	return &workforcev1.ListEmployeesResponse{Employees: m.employees}, nil
}

func (m *mockWorkforceForStreams) ListVHRRecords(_ context.Context, _ *workforcev1.ListVHRRecordsRequest) (*workforcev1.ListVHRRecordsResponse, error) {
	return &workforcev1.ListVHRRecordsResponse{Records: m.records}, nil
}

func (m *mockWorkforceForStreams) ListIncidents(_ context.Context, _ *workforcev1.ListIncidentsRequest) (*workforcev1.ListIncidentsResponse, error) {
	return &workforcev1.ListIncidentsResponse{Incidents: m.incidents}, nil
}

func TestBuildFelineExecutiveSummary(t *testing.T) {
	t.Run("nil employee returns nil", func(t *testing.T) {
		res := buildFelineExecutiveSummary(nil, nil)
		assert.Nil(t, res)
	})

	t.Run("employee with no VHR records", func(t *testing.T) {
		emp := &workforcev1.Employee{
			EmployeeID: "emp-01",
			FirstName:  "Whiskers",
			RoleTitle:  "Chief Nap Officer",
		}
		res := buildFelineExecutiveSummary(emp, nil)
		require.NotNil(t, res)
		assert.Equal(t, "emp-01", res.EmployeeID)
		assert.Equal(t, "Whiskers", res.Name)
		assert.Equal(t, "Chief Nap Officer", res.Role)
		assert.Equal(t, "ACTIVE", res.HealthStatus)
		assert.Equal(t, "100%", res.WhiskerSymmetry)
		assert.Equal(t, 25.0, res.PurrFrequencyHz)
		assert.Equal(t, "Observation Lounge", res.PreferredSunbeam)
		assert.Equal(t, int32(0), res.VHRRecordsCnt)
	})

	t.Run("employee with peak performer VHR record and clinical notes", func(t *testing.T) {
		emp := &workforcev1.Employee{
			EmployeeID: "emp-02",
			FirstName:  "Felix",
			RoleTitle:  "VP Purrformance",
		}
		recs := []*workforcev1.VHRRecord{
			{
				RecordID:      "vhr-1",
				FelineID:      "emp-02",
				DentalScore:   5,
				ClinicalNotes: "Perch 3 Window Beam",
			},
		}
		res := buildFelineExecutiveSummary(emp, recs)
		require.NotNil(t, res)
		assert.Equal(t, "PEAK_PERFORMER", res.HealthStatus)
		assert.Equal(t, "99.8%", res.WhiskerSymmetry)
		assert.Equal(t, 26.2, res.PurrFrequencyHz)
		assert.Equal(t, "Perch 3 Window Beam", res.PreferredSunbeam)
		assert.Equal(t, int32(1), res.VHRRecordsCnt)
	})

	t.Run("employee under observation", func(t *testing.T) {
		emp := &workforcev1.Employee{
			EmployeeID: "emp-03",
			FirstName:  "Garfield",
			RoleTitle:  "Lasagna Specialist",
		}
		recs := []*workforcev1.VHRRecord{
			{
				RecordID:    "vhr-2",
				FelineID:    "emp-03",
				DentalScore: 2,
			},
		}
		res := buildFelineExecutiveSummary(emp, recs)
		require.NotNil(t, res)
		assert.Equal(t, "UNDER_OBSERVATION", res.HealthStatus)
		assert.Equal(t, "95.0%", res.WhiskerSymmetry)
		assert.Equal(t, 22.0, res.PurrFrequencyHz)
		assert.Equal(t, "Observation Lounge", res.PreferredSunbeam)
		assert.Equal(t, int32(1), res.VHRRecordsCnt)
	})
}

func TestGetESGTransparency(t *testing.T) {
	mockWF := &mockWorkforceForStreams{
		employees: []*workforcev1.Employee{
			{EmployeeID: "fel-1", FirstName: "Mittens", RoleTitle: "CTO"},
			nil, // Test nil safety in emps list
			{EmployeeID: "fel-2", FirstName: "Shadow", RoleTitle: "Security Lead"},
		},
		records: []*workforcev1.VHRRecord{
			{RecordID: "vhr-101", FelineID: "fel-1", DentalScore: 4, ClinicalNotes: "Balcony Sunbeam"},
		},
		incidents: []*workforcev1.WorkplaceIncident{
			{IncidentID: "inc-1", Title: "Catnip spills"},
		},
	}

	server := NewServer(Config{})
	server.SetWorkforceClient(mockWF)

	resp, err := server.GetESGTransparency(context.Background(), &customerbllv1.GetESGTransparencyRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, 4.98, resp.OverallWelfareScore)
	assert.Equal(t, 100.0, resp.VeterinaryAdherencePct)
	assert.Len(t, resp.FelineExecutives, 2)
	assert.Equal(t, "Mittens", resp.FelineExecutives[0].Name)
	assert.Equal(t, "PEAK_PERFORMER", resp.FelineExecutives[0].HealthStatus)
	assert.Equal(t, "Shadow", resp.FelineExecutives[1].Name)
	assert.Equal(t, "ACTIVE", resp.FelineExecutives[1].HealthStatus)
	assert.Len(t, resp.Incidents, 1)
}
