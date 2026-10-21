package saga

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/pkg/cots/forgejo"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
	"github.com/orange-cat-investments/oci/pkg/cots/snipeit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgejoVCSAdapter(t *testing.T) {
	ctx := context.Background()

	t.Run("NewForgejoVCSAdapter and nil client fallback", func(t *testing.T) {
		adapter := NewForgejoVCSAdapter(nil)
		acc, err := adapter.ProvisionAccount(ctx, "jdoe", "jdoe@oci.local", "John Doe", "ssh-rsa AAA...")
		require.NoError(t, err)
		assert.Equal(t, "jdoe", acc.Username)
		assert.Equal(t, "Active", acc.Status)

		err = adapter.RevokeAccess(ctx, "jdoe")
		require.NoError(t, err)
	})

	t.Run("ProvisionAccount success path with SSH key", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/admin/users":
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"id":1,"username":"jdoe","email":"jdoe@oci.local","full_name":"John Doe"}`))
			case "/api/v1/admin/users/jdoe/keys":
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"id":100,"title":"Primary Developer SSH Key","key":"ssh-rsa AAA..."}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		client := forgejo.NewClient(forgejo.Config{
			BaseURL: server.URL,
		})
		adapter := NewForgejoVCSAdapter(client)

		acc, err := adapter.ProvisionAccount(ctx, "jdoe", "jdoe@oci.local", "John Doe", "ssh-rsa AAA...")
		require.NoError(t, err)
		assert.Equal(t, "jdoe", acc.Username)
		assert.Equal(t, "jdoe@oci.local", acc.Email)
		assert.Equal(t, []int64{100}, acc.SSHKeyIDs)
		assert.Equal(t, "Active", acc.Status)
	})

	t.Run("ProvisionAccount existing user fallback path", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/admin/users":
				// User already exists in Forgejo
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"message":"user already exists"}`))
			case "/api/v1/users/jdoe":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id":1,"username":"jdoe","email":"jdoe@oci.local","full_name":"John Doe"}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		client := forgejo.NewClient(forgejo.Config{
			BaseURL: server.URL,
		})
		adapter := NewForgejoVCSAdapter(client)

		acc, err := adapter.ProvisionAccount(ctx, "jdoe", "jdoe@oci.local", "John Doe", "")
		require.NoError(t, err)
		assert.Equal(t, "jdoe", acc.Username)
		assert.Equal(t, "John Doe", acc.FullName)
	})

	t.Run("ProvisionAccount error path - CreateUser and GetUser fail", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"internal server error"}`))
		}))
		defer server.Close()

		client := forgejo.NewClient(forgejo.Config{
			BaseURL: server.URL,
		})
		adapter := NewForgejoVCSAdapter(client)

		acc, err := adapter.ProvisionAccount(ctx, "jdoe", "jdoe@oci.local", "John Doe", "")
		require.Error(t, err)
		assert.Nil(t, acc)
		assert.Contains(t, err.Error(), "failed to provision Forgejo user")
	})

	t.Run("ProvisionAccount SSH key add failure ignored", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/admin/users":
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"id":1,"username":"jdoe","email":"jdoe@oci.local","full_name":"John Doe"}`))
			case "/api/v1/admin/users/jdoe/keys":
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"message":"invalid key format"}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		client := forgejo.NewClient(forgejo.Config{
			BaseURL: server.URL,
		})
		adapter := NewForgejoVCSAdapter(client)

		acc, err := adapter.ProvisionAccount(ctx, "jdoe", "jdoe@oci.local", "John Doe", "invalid-key")
		require.NoError(t, err)
		assert.Equal(t, "jdoe", acc.Username)
		assert.Empty(t, acc.SSHKeyIDs)
	})

	t.Run("RevokeAccess success and failure paths", func(t *testing.T) {
		serverSuccess := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer serverSuccess.Close()

		adapterSuccess := NewForgejoVCSAdapter(forgejo.NewClient(forgejo.Config{BaseURL: serverSuccess.URL}))
		err := adapterSuccess.RevokeAccess(ctx, "jdoe")
		require.NoError(t, err)

		serverErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"server error"}`))
		}))
		defer serverErr.Close()

		adapterErr := NewForgejoVCSAdapter(forgejo.NewClient(forgejo.Config{BaseURL: serverErr.URL}))
		err = adapterErr.RevokeAccess(ctx, "jdoe")
		require.Error(t, err)
	})
}

func TestNextcloudWorkspaceAdapter(t *testing.T) {
	ctx := context.Background()

	t.Run("NewNextcloudWorkspaceAdapter and nil client fallback", func(t *testing.T) {
		adapter := NewNextcloudWorkspaceAdapter(nil)
		ws, err := adapter.ProvisionWorkspace(ctx, "jdoe", "jdoe@oci.local", "Engineering", "")
		require.NoError(t, err)
		assert.Equal(t, "jdoe", ws.Username)
		assert.Equal(t, "50 GB", ws.Quota)
		assert.Contains(t, ws.GroupFolders, "engineering")

		err = adapter.LockAndArchiveStorage(ctx, "jdoe", "/vault/path")
		require.NoError(t, err)
	})

	t.Run("ProvisionWorkspace success path", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ocs":{"meta":{"status":"ok","statuscode":100,"message":"OK"}}}`))
		}))
		defer server.Close()

		client := nextcloud.NewClient(nextcloud.Config{BaseURL: server.URL})
		adapter := NewNextcloudWorkspaceAdapter(client)

		ws, err := adapter.ProvisionWorkspace(ctx, "jdoe", "jdoe@oci.local", "Facilities Support", "")
		require.NoError(t, err)
		assert.Equal(t, "jdoe", ws.Username)
		assert.Equal(t, "50 GB", ws.Quota)
		assert.Contains(t, ws.GroupFolders, "facilities-support")
	})

	t.Run("ProvisionWorkspace error path", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"ocs":{"meta":{"status":"failure","statuscode":500,"message":"Server error"}}}`))
		}))
		defer server.Close()

		client := nextcloud.NewClient(nextcloud.Config{BaseURL: server.URL})
		adapter := NewNextcloudWorkspaceAdapter(client)

		ws, err := adapter.ProvisionWorkspace(ctx, "jdoe", "jdoe@oci.local", "Finance", "100 GB")
		require.Error(t, err)
		assert.Nil(t, ws)
		assert.Contains(t, err.Error(), "failed to provision Nextcloud workspace")
	})

	t.Run("LockAndArchiveStorage success and failure paths", func(t *testing.T) {
		serverSuccess := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ocs":{"meta":{"status":"ok","statuscode":100,"message":"OK"}}}`))
		}))
		defer serverSuccess.Close()

		adapterSuccess := NewNextcloudWorkspaceAdapter(nextcloud.NewClient(nextcloud.Config{BaseURL: serverSuccess.URL}))
		err := adapterSuccess.LockAndArchiveStorage(ctx, "jdoe", "/vault/jdoe.tar.gz")
		require.NoError(t, err)

		serverErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"ocs":{"meta":{"status":"failure","statuscode":500,"message":"Error"}}}`))
		}))
		defer serverErr.Close()

		adapterErr := NewNextcloudWorkspaceAdapter(nextcloud.NewClient(nextcloud.Config{BaseURL: serverErr.URL}))
		err = adapterErr.LockAndArchiveStorage(ctx, "jdoe", "/vault/jdoe.tar.gz")
		require.Error(t, err)
	})
}

func TestSnipeITHardwareAdapter(t *testing.T) {
	ctx := context.Background()

	t.Run("NewSnipeITHardwareAdapter and nil client fallback", func(t *testing.T) {
		adapter := NewSnipeITHardwareAdapter(nil)
		allocs, err := adapter.AssignInitialHardware(ctx, "emp-123", "Staff Engineer", "Engineering")
		require.NoError(t, err)
		assert.Len(t, allocs, 4) // badge, laptop, desk, watch for engineer

		recovery, err := adapter.CreateAssetRecoveryTickets(ctx, "emp-123", "Resignation")
		require.NoError(t, err)
		assert.Equal(t, "Pending Recovery", recovery.Status)
		assert.Contains(t, recovery.TicketID, "RECOVERY-emp-123")
	})

	t.Run("AssignInitialHardware with active client", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"success","messages":"Checkout successful"}`))
		}))
		defer server.Close()

		client := snipeit.NewClient(snipeit.Config{BaseURL: server.URL})
		adapter := NewSnipeITHardwareAdapter(client)

		allocs, err := adapter.AssignInitialHardware(ctx, "emp-456", "Marketing Specialist", "Marketing")
		require.NoError(t, err)
		assert.Len(t, allocs, 3) // standard 3 allocations
	})

	t.Run("CreateAssetRecoveryTickets success path", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"total":2,"rows":[{"id":101,"asset_tag":"MBP-456","name":"MacBook Pro","model":"Laptop"},{"id":102,"asset_tag":"BADGE-456","name":"Security Badge","model":"Badge"}]}`))
		}))
		defer server.Close()

		client := snipeit.NewClient(snipeit.Config{BaseURL: server.URL})
		adapter := NewSnipeITHardwareAdapter(client)

		recovery, err := adapter.CreateAssetRecoveryTickets(ctx, "emp-456", "Separation")
		require.NoError(t, err)
		assert.Equal(t, "Pending Recovery", recovery.Status)
		assert.Equal(t, 2, recovery.TotalItems)
		assert.Len(t, recovery.Assets, 2)
	})

	t.Run("CreateAssetRecoveryTickets client error fallback path", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status":"error","messages":"Database error"}`))
		}))
		defer server.Close()

		client := snipeit.NewClient(snipeit.Config{BaseURL: server.URL})
		adapter := NewSnipeITHardwareAdapter(client)

		// When employeeID is empty, CreateAssetRecoveryTicket returns error, triggering fallback in adapter
		recovery, err := adapter.CreateAssetRecoveryTickets(ctx, "", "Termination")
		require.NoError(t, err)
		assert.Equal(t, "Pending Recovery", recovery.Status)
		assert.Contains(t, recovery.TicketID, "RECOVERY--")
		assert.Len(t, recovery.Assets, 2)
	})
}

func TestFrappeHRAdapter(t *testing.T) {
	ctx := context.Background()

	adapter := NewFrappeHRAdapter(frappe.NewClient(frappe.Config{}))
	err := adapter.UpdateEmployeeStatus(ctx, "emp-101", "active")
	require.NoError(t, err)

	err = adapter.LogSeparation(ctx, "emp-101", "2026-10-21", "Resignation")
	require.NoError(t, err)
}
