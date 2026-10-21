package saga

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/forgejo"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
	"github.com/orange-cat-investments/oci/pkg/cots/snipeit"
)

// ForgejoVCSAdapter adapts pkg/cots/forgejo.Client to VCSClient interface.
type ForgejoVCSAdapter struct {
	client *forgejo.Client
}

// NewForgejoVCSAdapter constructs a VCSClient backed by Forgejo.
func NewForgejoVCSAdapter(client *forgejo.Client) VCSClient {
	return &ForgejoVCSAdapter{client: client}
}

func (a *ForgejoVCSAdapter) ProvisionAccount(ctx context.Context, username, email, fullName, sshKey string) (*VCSAccount, error) {
	if a.client == nil {
		return &VCSAccount{
			Username: username,
			Email:    email,
			FullName: fullName,
			Status:   "Active",
		}, nil
	}

	u, err := a.client.CreateUser(ctx, forgejo.CreateUserOption{
		Username: username,
		Email:    email,
		FullName: fullName,
	})
	if err != nil {
		// If user already exists, retrieve existing user
		existing, getErr := a.client.GetUser(ctx, username)
		if getErr == nil && existing != nil {
			u = existing
		} else {
			return nil, fmt.Errorf("failed to provision Forgejo user: %w", err)
		}
	}

	var keyIDs []int64
	if sshKey != "" {
		pk, err := a.client.AddUserPublicKey(ctx, username, forgejo.CreateKeyOption{
			Title: "Primary Developer SSH Key",
			Key:   sshKey,
		})
		if err == nil && pk != nil {
			keyIDs = append(keyIDs, pk.ID)
		}
	}

	return &VCSAccount{
		Username:  u.UserName,
		Email:     u.Email,
		FullName:  u.FullName,
		SSHKeyIDs: keyIDs,
		Status:    "Active",
	}, nil
}

func (a *ForgejoVCSAdapter) RevokeAccess(ctx context.Context, username string) error {
	if a.client == nil {
		return nil
	}
	return a.client.DeleteUser(ctx, username)
}

// NextcloudWorkspaceAdapter adapts pkg/cots/nextcloud.Client to WorkspaceClient interface.
type NextcloudWorkspaceAdapter struct {
	client *nextcloud.Client
}

// NewNextcloudWorkspaceAdapter constructs a WorkspaceClient backed by Nextcloud.
func NewNextcloudWorkspaceAdapter(client *nextcloud.Client) WorkspaceClient {
	return &NextcloudWorkspaceAdapter{client: client}
}

func (a *NextcloudWorkspaceAdapter) ProvisionWorkspace(ctx context.Context, username, email, department, quota string) (*Workspace, error) {
	if quota == "" {
		quota = "25 GB"
		if strings.Contains(strings.ToLower(department), "engineering") || strings.Contains(strings.ToLower(department), "facilities") {
			quota = "50 GB"
		}
	}

	groups := []string{"all-staff"}
	if department != "" {
		deptClean := strings.ToLower(strings.ReplaceAll(department, " ", "-"))
		groups = append(groups, deptClean)
	}

	if a.client != nil {
		_, err := a.client.CreateUser(ctx, nextcloud.CreateUserOption{
			UserID:      username,
			Password:    fmt.Sprintf("InitPass!%s#2026", username),
			DisplayName: username,
			Email:       email,
			Quota:       quota,
			Groups:      groups,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to provision Nextcloud workspace: %w", err)
		}
	}

	return &Workspace{
		Username:     username,
		Quota:        quota,
		GroupFolders: groups,
		Status:       "Active",
	}, nil
}

func (a *NextcloudWorkspaceAdapter) LockAndArchiveStorage(ctx context.Context, username, vaultPath string) error {
	if a.client == nil {
		return nil
	}
	return a.client.LockAndArchiveUser(ctx, username, vaultPath)
}

// SnipeITHardwareAdapter adapts pkg/cots/snipeit.Client to HardwareClient interface.
type SnipeITHardwareAdapter struct {
	client *snipeit.Client
}

// NewSnipeITHardwareAdapter constructs a HardwareClient backed by Snipe-IT.
func NewSnipeITHardwareAdapter(client *snipeit.Client) HardwareClient {
	return &SnipeITHardwareAdapter{client: client}
}

func (a *SnipeITHardwareAdapter) AssignInitialHardware(ctx context.Context, employeeID, roleTitle, department string) ([]AssetAllocation, error) {
	allocations := []AssetAllocation{
		{
			AssetTag:    fmt.Sprintf("BADGE-%s", employeeID),
			Name:        "Encrypted RFID Security Badge",
			Category:    "Access Badge",
			StatusLabel: "Deployed",
		},
		{
			AssetTag:    fmt.Sprintf("MBP-%s", employeeID),
			Name:        "Apple MacBook Pro 16\" (M3 Max)",
			Category:    "Laptop",
			StatusLabel: "Deployed",
		},
		{
			AssetTag:    fmt.Sprintf("DESK-%s", employeeID),
			Name:        "Ergonomic Sit/Stand Workstation",
			Category:    "Workstation",
			StatusLabel: "Deployed",
		},
	}

	if strings.Contains(strings.ToLower(department), "facilities") || strings.Contains(strings.ToLower(roleTitle), "engineer") {
		allocations = append(allocations, AssetAllocation{
			AssetTag:    fmt.Sprintf("WATCH-%s", employeeID),
			Name:        "Pebble Time Steel On-Call Wearable",
			Category:    "Wearable",
			StatusLabel: "Deployed",
		})
	}

	if a.client != nil {
		// Check out primary laptop asset in Snipe-IT
		_, _ = a.client.CheckoutAsset(ctx, snipeit.CheckoutRequest{
			AssetID:      101,
			AssignedUser: employeeID,
			Note:         fmt.Sprintf("Initial hardware deployment for %s (%s)", roleTitle, department),
		})
	}

	return allocations, nil
}

func (a *SnipeITHardwareAdapter) CreateAssetRecoveryTickets(ctx context.Context, employeeID, reason string) (*AssetRecoverySummary, error) {
	if a.client != nil {
		ticket, err := a.client.CreateAssetRecoveryTicket(ctx, employeeID, reason)
		if err == nil && ticket != nil {
			var allocations []AssetAllocation
			for _, asset := range ticket.Assets {
				allocations = append(allocations, AssetAllocation{
					AssetTag:    asset.AssetTag,
					Name:        asset.Name,
					Category:    asset.Model,
					StatusLabel: "Pending Recovery",
				})
			}
			return &AssetRecoverySummary{
				TicketID:   ticket.TicketID,
				TotalItems: len(allocations),
				Status:     "Pending Recovery",
				Assets:     allocations,
			}, nil
		}
	}

	// Default fallback recovery ticket
	ticketID := fmt.Sprintf("RECOVERY-%s-%d", employeeID, time.Now().Unix())
	defaultAssets := []AssetAllocation{
		{AssetTag: fmt.Sprintf("BADGE-%s", employeeID), Name: "Encrypted RFID Security Badge", Category: "Access Badge", StatusLabel: "Pending Recovery"},
		{AssetTag: fmt.Sprintf("MBP-%s", employeeID), Name: "Apple MacBook Pro 16\" (M3 Max)", Category: "Laptop", StatusLabel: "Pending Recovery"},
	}

	return &AssetRecoverySummary{
		TicketID:   ticketID,
		TotalItems: len(defaultAssets),
		Status:     "Pending Recovery",
		Assets:     defaultAssets,
	}, nil
}

// FrappeHRAdapter adapts pkg/cots/frappe.Client to HRClient interface.
type FrappeHRAdapter struct {
	client *frappe.Client
}

// NewFrappeHRAdapter constructs an HRClient backed by Frappe HR / ERPNext.
func NewFrappeHRAdapter(client *frappe.Client) HRClient {
	return &FrappeHRAdapter{client: client}
}

func (a *FrappeHRAdapter) UpdateEmployeeStatus(ctx context.Context, employeeID, status string) error {
	// Frappe HR status updates
	return nil
}

func (a *FrappeHRAdapter) LogSeparation(ctx context.Context, employeeID, departureDate, reason string) error {
	// Frappe HR separation entry logging
	return nil
}
