package snipeit

import (
	"context"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/snipeit"
)

// Client wraps snipeit.Client with simulation conveniences such as
// simulated hardware recovery tickets for offline testing.
type Client struct {
	*snipeit.Client
}

// Config re-exports snipeit.Config.
type Config = snipeit.Config

// AssetRecoveryTicket re-exports snipeit.AssetRecoveryTicket.
type AssetRecoveryTicket = snipeit.AssetRecoveryTicket

// Asset re-exports snipeit.Asset.
type Asset = snipeit.Asset

// NewClient initializes a simulation Snipe-IT client.
func NewClient(cfg Config) *Client {
	return &Client{
		Client: snipeit.NewClient(cfg),
	}
}

// CreateAssetRecoveryTicket creates an equipment recovery ticket, falling back
// to simulated assets if Snipe-IT is not reachable.
func (c *Client) CreateAssetRecoveryTicket(ctx context.Context, employeeID, reason string) (*AssetRecoveryTicket, error) {
	ticket, err := c.Client.CreateAssetRecoveryTicket(ctx, employeeID, reason)
	if err == nil && ticket != nil {
		return ticket, nil
	}

	// Fallback simulated ticket for testing and simulation
	ticketID := fmt.Sprintf("RECOVER-%s-%d", employeeID, time.Now().Unix())
	defaultAssets := []Asset{
		{AssetTag: fmt.Sprintf("BADGE-%s", employeeID), Name: "Encrypted RFID Security Badge", Model: "Access Badge", StatusLabel: "Pending Recovery"},
		{AssetTag: fmt.Sprintf("MBP-%s", employeeID), Name: "Apple MacBook Pro 16\" (M3 Max)", Model: "Laptop", StatusLabel: "Pending Recovery"},
	}

	return &AssetRecoveryTicket{
		TicketID:   ticketID,
		EmployeeID: employeeID,
		Assets:     defaultAssets,
		Status:     "Pending Recovery",
		Reason:     reason,
		CreatedAt:  time.Now().UTC(),
	}, nil
}
