package saga

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/forgejo"
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
	"github.com/orange-cat-investments/oci/pkg/cots/snipeit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgejoVCSAdapter(t *testing.T) {
	ctx := context.Background()

	t.Run("Nil client fallback", func(t *testing.T) {
		adapter := NewForgejoVCSAdapter(nil)

		acc, err := adapter.ProvisionAccount(ctx, "john.doe", "john@oci.local", "John Doe", "ssh-ed25519 AAA...")
		require.NoError(t, err)
		assert.Equal(t, "john.doe", acc.Username)
		assert.Equal(t, "john@oci.local", acc.Email)
		assert.Equal(t, "John Doe", acc.FullName)
		assert.Equal(t, "Active", acc.Status)
		assert.Empty(t, acc.SSHKeyIDs)

		err = adapter.RevokeAccess(ctx, "john.doe")
		assert.NoError(t, err)
	})

	t.Run("Create user success with SSH key", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/admin/users":
				assert.Equal(t, http.MethodPost, r.Method)
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(forgejo.User{
					ID:       10,
					UserName: "alice.dev",
					Email:    "alice@oci.local",
					FullName: "Alice Dev",
				})
			case "/api/v1/admin/users/alice.dev/keys":
				assert.Equal(t, http.MethodPost, r.Method)
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(forgejo.PublicKey{
					ID:    505,
					Title: "Primary Developer SSH Key",
					Key:   "ssh-ed25519 AAA...",
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		client := forgejo.NewClient(forgejo.Config{BaseURL: server.URL, Token: "test-token"})
		adapter := NewForgejoVCSAdapter(client)

		acc, err := adapter.ProvisionAccount(ctx, "alice.dev", "alice@oci.local", "Alice Dev", "ssh-ed25519 AAA...")
		require.NoError(t, err)
		assert.Equal(t, "alice.dev", acc.Username)
		assert.Equal(t, "alice@oci.local", acc.Email)
		assert.Equal(t, "Alice Dev", acc.FullName)
		assert.Equal(t, []int64{505}, acc.SSHKeyIDs)
		assert.Equal(t, "Active", acc.Status)
	})

	t.Run("User exists fallback and SSH key error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/admin/users":
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"message": "user already exists"}`))
			case "/api/v1/users/existing.user":
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(forgejo.User{
					ID:       20,
					UserName: "existing.user",
					Email:    "existing@oci.local",
					FullName: "Existing User",
				})
			case "/api/v1/admin/users/existing.user/keys":
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"message": "key already added"}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		client := forgejo.NewClient(forgejo.Config{BaseURL: server.URL})
		adapter := NewForgejoVCSAdapter(client)

		acc, err := adapter.ProvisionAccount(ctx, "existing.user", "existing@oci.local", "Existing User", "ssh-ed25519 AAA...")
		require.NoError(t, err)
		assert.Equal(t, "existing.user", acc.Username)
		assert.Empty(t, acc.SSHKeyIDs)
	})

	t.Run("User creation fails and GetUser fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message": "internal server error"}`))
		}))
		defer server.Close()

		client := forgejo.NewClient(forgejo.Config{BaseURL: server.URL})
		adapter := NewForgejoVCSAdapter(client)

		acc, err := adapter.ProvisionAccount(ctx, "bad.user", "bad@oci.local", "Bad User", "")
		require.Error(t, err)
		assert.Nil(t, acc)
		assert.Contains(t, err.Error(), "failed to provision Forgejo user")
	})

	t.Run("RevokeAccess with client", func(t *testing.T) {
		var deletedUser string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/admin/users/departed.user" {
				deletedUser = "departed.user"
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		client := forgejo.NewClient(forgejo.Config{BaseURL: server.URL})
		adapter := NewForgejoVCSAdapter(client)

		err := adapter.RevokeAccess(ctx, "departed.user")
		assert.NoError(t, err)
		assert.Equal(t, "departed.user", deletedUser)
	})
}

func TestNextcloudWorkspaceAdapter(t *testing.T) {
	ctx := context.Background()

	t.Run("Quota and group folder derivation logic with nil client", func(t *testing.T) {
		adapter := NewNextcloudWorkspaceAdapter(nil)

		// 1. Default quota (25 GB) and standard department
		ws1, err := adapter.ProvisionWorkspace(ctx, "bob.smith", "bob@oci.local", "Finance", "")
		require.NoError(t, err)
		assert.Equal(t, "25 GB", ws1.Quota)
		assert.Equal(t, []string{"all-staff", "finance"}, ws1.GroupFolders)

		// 2. Department with "Engineering" -> 50 GB
		ws2, err := adapter.ProvisionWorkspace(ctx, "eng.user", "eng@oci.local", "Software Engineering", "")
		require.NoError(t, err)
		assert.Equal(t, "50 GB", ws2.Quota)
		assert.Equal(t, []string{"all-staff", "software-engineering"}, ws2.GroupFolders)

		// 3. Department with "Facilities" -> 50 GB
		ws3, err := adapter.ProvisionWorkspace(ctx, "fac.user", "fac@oci.local", "Facilities Operations", "")
		require.NoError(t, err)
		assert.Equal(t, "50 GB", ws3.Quota)
		assert.Equal(t, []string{"all-staff", "facilities-operations"}, ws3.GroupFolders)

		// 4. Custom Quota provided
		ws4, err := adapter.ProvisionWorkspace(ctx, "exec.user", "exec@oci.local", "Executive", "100 GB")
		require.NoError(t, err)
		assert.Equal(t, "100 GB", ws4.Quota)
		assert.Equal(t, []string{"all-staff", "executive"}, ws4.GroupFolders)

		// 5. Empty department
		ws5, err := adapter.ProvisionWorkspace(ctx, "nodept.user", "nodept@oci.local", "", "")
		require.NoError(t, err)
		assert.Equal(t, []string{"all-staff"}, ws5.GroupFolders)

		err = adapter.LockAndArchiveStorage(ctx, "bob.smith", "/vault/path")
		assert.NoError(t, err)
	})

	t.Run("ProvisionWorkspace with client success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/ocs/v1.php/cloud/users", r.URL.Path)
			assert.Equal(t, "format=json", r.URL.RawQuery)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ocs":{"meta":{"status":"ok","statuscode":100}}}`))
		}))
		defer server.Close()

		client := nextcloud.NewClient(nextcloud.Config{BaseURL: server.URL, AdminUser: "admin", AdminPassword: "pass"})
		adapter := NewNextcloudWorkspaceAdapter(client)

		ws, err := adapter.ProvisionWorkspace(ctx, "carol.dev", "carol@oci.local", "Engineering", "50 GB")
		require.NoError(t, err)
		assert.Equal(t, "carol.dev", ws.Username)
		assert.Equal(t, "50 GB", ws.Quota)
		assert.Equal(t, "Active", ws.Status)
	})

	t.Run("ProvisionWorkspace with client error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "failed"}`))
		}))
		defer server.Close()

		client := nextcloud.NewClient(nextcloud.Config{BaseURL: server.URL})
		adapter := NewNextcloudWorkspaceAdapter(client)

		ws, err := adapter.ProvisionWorkspace(ctx, "fail.user", "fail@oci.local", "Engineering", "")
		require.Error(t, err)
		assert.Nil(t, ws)
		assert.Contains(t, err.Error(), "failed to provision Nextcloud workspace")
	})

	t.Run("LockAndArchiveStorage with client", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/ocs/v1.php/cloud/users/test.user/disable":
				assert.Equal(t, http.MethodPut, r.Method)
				w.WriteHeader(http.StatusOK)
			case "/ocs/v1.php/cloud/users/test.user/archive":
				assert.Equal(t, http.MethodPost, r.Method)
				w.WriteHeader(http.StatusOK)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		client := nextcloud.NewClient(nextcloud.Config{BaseURL: server.URL})
		adapter := NewNextcloudWorkspaceAdapter(client)

		err := adapter.LockAndArchiveStorage(ctx, "test.user", "/vault/archives/test.tar.gz")
		assert.NoError(t, err)
	})
}

func TestSnipeITHardwareAdapter(t *testing.T) {
	ctx := context.Background()

	t.Run("AssignInitialHardware asset allocation logic with nil client", func(t *testing.T) {
		adapter := NewSnipeITHardwareAdapter(nil)

		// Regular role/dept -> 3 base assets
		assets1, err := adapter.AssignInitialHardware(ctx, "emp-101", "Analyst", "Finance")
		require.NoError(t, err)
		assert.Len(t, assets1, 3)
		assert.Equal(t, "BADGE-emp-101", assets1[0].AssetTag)
		assert.Equal(t, "MBP-emp-101", assets1[1].AssetTag)
		assert.Equal(t, "DESK-emp-101", assets1[2].AssetTag)

		// Role contains "engineer" -> 4 assets including watch
		assets2, err := adapter.AssignInitialHardware(ctx, "emp-102", "Senior Engineer", "Product")
		require.NoError(t, err)
		assert.Len(t, assets2, 4)
		assert.Equal(t, "WATCH-emp-102", assets2[3].AssetTag)
		assert.Equal(t, "Pebble Time Steel On-Call Wearable", assets2[3].Name)

		// Department contains "facilities" -> 4 assets including watch
		assets3, err := adapter.AssignInitialHardware(ctx, "emp-103", "Manager", "Facilities Ops")
		require.NoError(t, err)
		assert.Len(t, assets3, 4)
		assert.Equal(t, "WATCH-emp-103", assets3[3].AssetTag)
	})

	t.Run("AssignInitialHardware with client checkout", func(t *testing.T) {
		checkedOutAsset := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/api/v1/hardware/101/checkout" {
				checkedOutAsset = true
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(snipeit.Asset{ID: 101, StatusLabel: "Deployed"})
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		client := snipeit.NewClient(snipeit.Config{BaseURL: server.URL, APIKey: "test-key"})
		adapter := NewSnipeITHardwareAdapter(client)

		assets, err := adapter.AssignInitialHardware(ctx, "emp-104", "Developer", "IT")
		require.NoError(t, err)
		assert.Len(t, assets, 3)
		assert.True(t, checkedOutAsset)
	})

	t.Run("CreateAssetRecoveryTickets with nil client fallback", func(t *testing.T) {
		adapter := NewSnipeITHardwareAdapter(nil)

		summary, err := adapter.CreateAssetRecoveryTickets(ctx, "emp-105", "Resignation")
		require.NoError(t, err)
		assert.Contains(t, summary.TicketID, "RECOVERY-emp-105-")
		assert.Equal(t, 2, summary.TotalItems)
		assert.Equal(t, "Pending Recovery", summary.Status)
		assert.Len(t, summary.Assets, 2)
	})

	t.Run("CreateAssetRecoveryTickets with client returning assets", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == "/api/v1/hardware" {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{
					"rows": []snipeit.Asset{
						{AssetTag: "MBP-emp-106", Name: "MacBook Pro", Model: "Laptop"},
						{AssetTag: "BADGE-emp-106", Name: "Security Badge", Model: "Access Badge"},
					},
				})
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		client := snipeit.NewClient(snipeit.Config{BaseURL: server.URL})
		adapter := NewSnipeITHardwareAdapter(client)

		summary, err := adapter.CreateAssetRecoveryTickets(ctx, "emp-106", "Separation")
		require.NoError(t, err)
		assert.Contains(t, summary.TicketID, "RECOVER-emp-106-")
		assert.Equal(t, 2, summary.TotalItems)
		assert.Equal(t, "Pending Recovery", summary.Status)
		assert.Len(t, summary.Assets, 2)
		assert.Equal(t, "MBP-emp-106", summary.Assets[0].AssetTag)
	})
}

func TestFrappeHRAdapter(t *testing.T) {
	ctx := context.Background()

	adapter := NewFrappeHRAdapter(nil)

	err := adapter.UpdateEmployeeStatus(ctx, "emp-201", "separated")
	assert.NoError(t, err)

	err = adapter.LogSeparation(ctx, "emp-201", time.Now().Format("2006-01-02"), "Career Transition")
	assert.NoError(t, err)
}
