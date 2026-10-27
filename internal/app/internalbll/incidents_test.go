package internalbll

import (
	"context"
	"testing"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/search/meilisearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWFForIncident struct {
	workforcev1.WorkforceServiceClient
}

func (m *mockWFForIncident) CreateIncident(_ context.Context, in *workforcev1.CreateIncidentRequest) (*workforcev1.CreateIncidentResponse, error) {
	return &workforcev1.CreateIncidentResponse{
		Incident: &workforcev1.WorkplaceIncident{
			IncidentID:       "inc-meili-001",
			Title:            in.Title,
			Category:         in.Category,
			InvolvedFelineID: in.InvolvedFelineID,
			Severity:         in.Severity,
			Status:           "open",
			Description:      in.Description,
		},
	}, nil
}

type mockOpsForIncident struct {
	opsv1.OpsServiceClient
}

func (m *mockOpsForIncident) CreateTicket(_ context.Context, in *opsv1.CreateTicketRequest) (*opsv1.CreateTicketResponse, error) {
	return &opsv1.CreateTicketResponse{
		Ticket: &opsv1.ITTicket{
			TicketID: "tick-meili-001",
			Title:    in.Title,
		},
	}, nil
}

type dummyPub struct{}

func (d *dummyPub) Publish(_ string, _ any) error { return nil }

func TestIncidents_MeilisearchIndexing(t *testing.T) {
	meiliClient := meilisearch.NewClient("http://127.0.0.1:7700", "test-key")
	s := NewServer(&mockWFForIncident{}, nil, &mockOpsForIncident{}, &dummyPub{}, nil, Config{
		SearchClient: meiliClient,
	})

	ctx := context.Background()

	// 1. CreateIncident should index into employee_knowledge
	resp, err := s.CreateIncident(ctx, &workforcev1.CreateIncidentRequest{
		Title:            "Spilled milk in corridor B",
		Description:      "Pebbles knocked over the water bowl",
		Category:         "habitat_disruption",
		Severity:         "low",
		InvolvedFelineID: "fel-01",
	})
	require.NoError(t, err)
	assert.Equal(t, "inc-meili-001", resp.Incident.IncidentID)

	// Verify indexed document can be searched via in-memory search
	searchRes, err := meiliClient.Search(ctx, "employee_knowledge", "Spilled milk", meilisearch.SearchOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, searchRes.Hits)
	assert.Equal(t, "inc-meili-001", searchRes.Hits[0]["id"])

	// 2. BroadcastPebbleIncidentAlert should index into employee_knowledge
	bResp, err := s.BroadcastPebbleIncidentAlert(ctx, &internalbllv1.BroadcastPebbleIncidentAlertRequest{
		IncidentID:  "inc-meili-002",
		Title:       "Scratching post collapse",
		Category:    "scratching",
		Severity:    "high",
		Description: "Post in cat lounge 3 fell over",
	})
	require.NoError(t, err)
	assert.Equal(t, "broadcasted", bResp.Status)

	searchRes2, err := meiliClient.Search(ctx, "employee_knowledge", "Scratching post", meilisearch.SearchOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, searchRes2.Hits)
	assert.Equal(t, "inc-meili-002", searchRes2.Hits[0]["id"])
}
