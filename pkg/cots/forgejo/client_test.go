package forgejo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgejoClient_CreateUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/admin/users", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "token secret-token", r.Header.Get("Authorization"))

		var opt CreateUserOption
		err := json.NewDecoder(r.Body).Decode(&opt)
		require.NoError(t, err)
		assert.Equal(t, "elise.dev", opt.Username)
		assert.Equal(t, "elise@oci.local", opt.Email)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(User{
			ID:        42,
			UserName:  opt.Username,
			Email:     opt.Email,
			FullName:  "Elise Dev",
			Active:    true,
			CreatedAt: time.Now().UTC(),
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Token:   "secret-token",
	})

	user, err := client.CreateUser(context.Background(), CreateUserOption{
		Username: "elise.dev",
		Email:    "elise@oci.local",
		FullName: "Elise Dev",
		Password: "SecretDevPass123!",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), user.ID)
	assert.Equal(t, "elise.dev", user.UserName)
}

func TestForgejoClient_CreateUser_PasswordRequired(t *testing.T) {
	client := NewClient(Config{
		BaseURL: "http://dummy.local",
		Token:   "secret-token",
	})

	_, err := client.CreateUser(context.Background(), CreateUserOption{
		Username: "user1",
		Email:    "user1@oci.local",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
}

func TestForgejoClient_AddUserPublicKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/admin/users/elise.dev/keys", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var opt CreateKeyOption
		err := json.NewDecoder(r.Body).Decode(&opt)
		require.NoError(t, err)
		assert.Equal(t, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...", opt.Key)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(PublicKey{
			ID:        101,
			Title:     opt.Title,
			Key:       opt.Key,
			ReadOnly:  false,
			CreatedAt: time.Now().UTC(),
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Token:   "secret-token",
	})

	key, err := client.AddUserPublicKey(context.Background(), "elise.dev", CreateKeyOption{
		Title: "MacBook Pro Key",
		Key:   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(101), key.ID)
	assert.Equal(t, "MacBook Pro Key", key.Title)
}

func TestForgejoClient_DeleteUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/admin/users/separated.user", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Token:   "secret-token",
	})

	err := client.DeleteUser(context.Background(), "separated.user")
	assert.NoError(t, err)
}

func TestForgejoClient_CreateAndManageIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/repos/oci-internal/helpdesk/issues" && r.Method == http.MethodPost:
			var opt CreateIssueOption
			err := json.NewDecoder(r.Body).Decode(&opt)
			require.NoError(t, err)
			assert.Equal(t, "Habitat Climate Controller Offline", opt.Title)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Issue{
				ID:        1,
				Index:     42,
				Title:     opt.Title,
				Body:      opt.Body,
				State:     "open",
				CreatedAt: time.Now().UTC(),
			})

		case r.URL.Path == "/api/v1/repos/oci-internal/helpdesk/issues/42" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Issue{
				ID:        1,
				Index:     42,
				Title:     "Habitat Climate Controller Offline",
				State:     "open",
				CreatedAt: time.Now().UTC(),
			})

		case r.URL.Path == "/api/v1/repos/oci-internal/helpdesk/issues/42" && r.Method == http.MethodPatch:
			var opt UpdateIssueOption
			err := json.NewDecoder(r.Body).Decode(&opt)
			require.NoError(t, err)
			assert.Equal(t, "closed", *opt.State)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Issue{
				ID:    1,
				Index: 42,
				State: "closed",
			})

		case r.URL.Path == "/api/v1/repos/oci-internal/helpdesk/issues/42/comments" && r.Method == http.MethodPost:
			var opt CreateCommentOption
			err := json.NewDecoder(r.Body).Decode(&opt)
			require.NoError(t, err)
			assert.Equal(t, "Thermostat sensor rebooted successfully.", opt.Body)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Comment{
				ID:        99,
				Body:      opt.Body,
				CreatedAt: time.Now().UTC(),
			})

		case r.URL.Path == "/api/v1/repos/oci-internal/helpdesk/issues/42/comments" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]Comment{
				{ID: 99, Body: "Thermostat sensor rebooted successfully.", CreatedAt: time.Now().UTC()},
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Token:   "secret-token",
	})

	ctx := context.Background()

	// 1. Create Issue
	issue, err := client.CreateIssue(ctx, "oci-internal", "helpdesk", CreateIssueOption{
		Title: "Habitat Climate Controller Offline",
		Body:  "Thermal readings unverified in Zone Alpha.",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), issue.Index)

	// 2. Get Issue
	fetched, err := client.GetIssue(ctx, "oci-internal", "helpdesk", 42)
	require.NoError(t, err)
	assert.Equal(t, "Habitat Climate Controller Offline", fetched.Title)

	// 3. Create Comment
	comment, err := client.CreateIssueComment(ctx, "oci-internal", "helpdesk", 42, "Thermostat sensor rebooted successfully.")
	require.NoError(t, err)
	assert.Equal(t, int64(99), comment.ID)

	// 4. List Comments
	comments, err := client.ListIssueComments(ctx, "oci-internal", "helpdesk", 42)
	require.NoError(t, err)
	assert.Len(t, comments, 1)

	// 5. Update Issue
	closedState := "closed"
	updated, err := client.UpdateIssue(ctx, "oci-internal", "helpdesk", 42, UpdateIssueOption{
		State: &closedState,
	})
	require.NoError(t, err)
	assert.Equal(t, "closed", updated.State)
}

func TestForgejoClient_OrganizationAndRepositoryLifecycle(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/orgs", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "token secret-token", r.Header.Get("Authorization"))
		var opt CreateOrgOption
		err := json.NewDecoder(r.Body).Decode(&opt)
		require.NoError(t, err)
		assert.Equal(t, "oci", opt.UserName)

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Organization{
			ID:          10,
			UserName:    opt.UserName,
			FullName:    opt.FullName,
			Description: opt.Description,
			Visibility:  "public",
		})
	})

	mux.HandleFunc("GET /api/v1/orgs/{org}", func(w http.ResponseWriter, r *http.Request) {
		org := r.PathValue("org")
		if org != "oci" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Organization{
			ID:          10,
			UserName:    "oci",
			FullName:    "Orange Cat Investments",
			Description: "Enterprise Monorepo GitOps Platform",
			Visibility:  "public",
		})
	})

	mux.HandleFunc("POST /api/v1/orgs/{org}/repos", func(w http.ResponseWriter, r *http.Request) {
		org := r.PathValue("org")
		assert.Equal(t, "oci", org)

		var opt CreateRepoOption
		err := json.NewDecoder(r.Body).Decode(&opt)
		require.NoError(t, err)
		assert.Equal(t, "monorepo", opt.Name)

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Repository{
			ID:            101,
			Name:          opt.Name,
			FullName:      "oci/" + opt.Name,
			Description:   opt.Description,
			Private:       opt.Private,
			DefaultBranch: "main",
			HasActions:    true,
			HasIssues:     true,
			HasPull:       true,
		})
	})

	mux.HandleFunc("GET /api/v1/repos/{owner}/{repo}", func(w http.ResponseWriter, r *http.Request) {
		owner := r.PathValue("owner")
		repo := r.PathValue("repo")
		if owner != "oci" || repo != "monorepo" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Repository{
			ID:            101,
			Name:          repo,
			FullName:      owner + "/" + repo,
			DefaultBranch: "main",
			HasActions:    true,
			HasIssues:     true,
			HasPull:       true,
		})
	})

	mux.HandleFunc("PATCH /api/v1/repos/{owner}/{repo}", func(w http.ResponseWriter, r *http.Request) {
		owner := r.PathValue("owner")
		repo := r.PathValue("repo")
		assert.Equal(t, "oci", owner)
		assert.Equal(t, "monorepo", repo)

		var opt UpdateRepoOption
		err := json.NewDecoder(r.Body).Decode(&opt)
		require.NoError(t, err)
		assert.True(t, *opt.HasActions)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Repository{
			ID:            101,
			Name:          repo,
			FullName:      owner + "/" + repo,
			DefaultBranch: "main",
			HasActions:    *opt.HasActions,
			HasIssues:     true,
			HasPull:       true,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Token:   "secret-token",
	})
	ctx := context.Background()

	// 1. Create Organization
	org, err := client.CreateOrganization(ctx, CreateOrgOption{
		UserName:    "oci",
		FullName:    "Orange Cat Investments",
		Description: "Enterprise Monorepo GitOps Platform",
	})
	require.NoError(t, err)
	assert.Equal(t, "oci", org.UserName)
	assert.Equal(t, int64(10), org.ID)

	// 2. Get Organization
	fetchedOrg, err := client.GetOrganization(ctx, "oci")
	require.NoError(t, err)
	assert.Equal(t, "Orange Cat Investments", fetchedOrg.FullName)

	// 3. Create Org Repository
	repo, err := client.CreateOrgRepository(ctx, "oci", CreateRepoOption{
		Name:        "monorepo",
		Description: "Main OCI codebase monorepo",
		Private:     false,
	})
	require.NoError(t, err)
	assert.Equal(t, "monorepo", repo.Name)
	assert.Equal(t, "oci/monorepo", repo.FullName)

	// 4. Get Repository
	fetchedRepo, err := client.GetRepository(ctx, "oci", "monorepo")
	require.NoError(t, err)
	assert.Equal(t, "oci/monorepo", fetchedRepo.FullName)

	// 5. Update Repository
	actionsEnabled := true
	updatedRepo, err := client.UpdateRepository(ctx, "oci", "monorepo", UpdateRepoOption{
		HasActions: &actionsEnabled,
	})
	require.NoError(t, err)
	assert.True(t, updatedRepo.HasActions)
}

