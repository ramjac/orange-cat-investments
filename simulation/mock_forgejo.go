package simulation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// MockForgejoServer provides an in-memory simulation of Forgejo REST API v1.
// Used strictly within simulation tests and offline mock mode, keeping production
// domain handlers completely isolated from VCS / Git host endpoints.
type MockForgejoServer struct {
	mu       sync.RWMutex
	branches map[string]bool
	issues   map[int64]map[string]any
	pulls    map[int64]map[string]any
	reviews  map[int64][]map[string]any
	comments map[int64][]string
	orgs     map[string]map[string]any
	repos    map[string]map[string]any
}

// NewMockForgejoServer initializes the mock Forgejo state with default repository baseline.
func NewMockForgejoServer() *MockForgejoServer {
	m := &MockForgejoServer{
		branches: map[string]bool{"main": true},
		issues:   make(map[int64]map[string]any),
		pulls:    make(map[int64]map[string]any),
		reviews:  make(map[int64][]map[string]any),
		comments: make(map[int64][]string),
		orgs:     make(map[string]map[string]any),
		repos:    make(map[string]map[string]any),
	}

	// Seed Organization "oci"
	m.orgs["oci"] = map[string]any{
		"id":          1,
		"username":    "oci",
		"full_name":   "Orange Cat Investments",
		"description": "Enterprise Monorepo GitOps Platform",
		"visibility":  "public",
	}

	// Seed Repository "oci/monorepo"
	m.repos["oci/monorepo"] = map[string]any{
		"id":                42,
		"name":              "monorepo",
		"full_name":         "oci/monorepo",
		"default_branch":    "main",
		"private":           false,
		"has_actions":       true,
		"has_issues":        true,
		"has_pull_requests": true,
	}

	// Seed Repository "oci/orange-cat-investments"
	m.repos["oci/orange-cat-investments"] = map[string]any{
		"id":                43,
		"name":              "orange-cat-investments",
		"full_name":         "oci/orange-cat-investments",
		"default_branch":    "main",
		"private":           false,
		"has_actions":       true,
		"has_issues":        true,
		"has_pull_requests": true,
	}

	// Seed Card #42 on the engineering kanban backlog
	m.issues[42] = map[string]any{
		"id":         42,
		"number":     42,
		"title":      "feat(gitops): multi-stage containerization & gated promotion pipeline",
		"body":       "Containerize Go backend services and Vue 3 web portals with multi-stage Dockerfiles. Deploy in-cluster Forgejo runner.",
		"state":      "open",
		"assignee":   nil,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	// Seed PR #42 baseline so reviews/merges can be tested independently
	m.pulls[42] = map[string]any{
		"id":         42,
		"number":     42,
		"title":      "feat(gitops): multi-stage containerization & gated promotion pipeline",
		"body":       "Containerize Go backend services and Vue 3 web portals with multi-stage Dockerfiles. Deploy in-cluster Forgejo runner.",
		"state":      "open",
		"head":       map[string]any{"ref": "feature/42-gitops-promotion-pipeline"},
		"base":       map[string]any{"ref": "main"},
		"merged":     false,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	return m
}

// Handler returns an http.Handler that routes Forgejo REST API v1 endpoints.
func (m *MockForgejoServer) Handler() http.Handler {
	mux := http.NewServeMux()

	// Version & Server Info
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "7.0.0"})
	})

	// Organizations
	mux.HandleFunc("POST /api/v1/orgs", m.handleCreateOrg)
	mux.HandleFunc("GET /api/v1/orgs/{org}", m.handleGetOrg)
	mux.HandleFunc("POST /api/v1/orgs/{org}/repos", m.handleCreateOrgRepo)

	// Repositories
	mux.HandleFunc("GET /api/v1/repos/{owner}/{repo}", m.handleGetRepo)
	mux.HandleFunc("PATCH /api/v1/repos/{owner}/{repo}", m.handlePatchRepo)

	// Issues & Kanban
	mux.HandleFunc("POST /api/v1/repos/{owner}/{repo}/issues", m.handleCreateIssue)
	mux.HandleFunc("GET /api/v1/repos/{owner}/{repo}/issues/{index}", m.handleGetIssue)
	mux.HandleFunc("PATCH /api/v1/repos/{owner}/{repo}/issues/{index}", m.handlePatchIssue)
	mux.HandleFunc("POST /api/v1/repos/{owner}/{repo}/issues/{index}/comments", m.handleCreateIssueComment)

	// Git Branches
	mux.HandleFunc("POST /api/v1/repos/{owner}/{repo}/branches", m.handleCreateBranch)
	mux.HandleFunc("GET /api/v1/repos/{owner}/{repo}/branches", m.handleListBranches)

	// File Commits
	mux.HandleFunc("POST /api/v1/repos/{owner}/{repo}/contents/{path...}", m.handleCreateFileCommit)

	// Pull Requests
	mux.HandleFunc("POST /api/v1/repos/{owner}/{repo}/pulls", m.handleCreatePullRequest)
	mux.HandleFunc("GET /api/v1/repos/{owner}/{repo}/pulls", m.handleListPullRequests)
	mux.HandleFunc("GET /api/v1/repos/{owner}/{repo}/pulls/{index}", m.handleGetPullRequest)
	mux.HandleFunc("POST /api/v1/repos/{owner}/{repo}/pulls/{index}/reviews", m.handleCreatePullReview)
	mux.HandleFunc("POST /api/v1/repos/{owner}/{repo}/pulls/{index}/merge", m.handleMergePullRequest)

	return mux
}

func (m *MockForgejoServer) handleGetIssue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var idx int64
	_, _ = fmt.Sscanf(r.PathValue("index"), "%d", &idx)

	m.mu.RLock()
	defer m.mu.RUnlock()

	issue, ok := m.issues[idx]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "issue not found"})
		return
	}
	_ = json.NewEncoder(w).Encode(issue)
}

func (m *MockForgejoServer) handlePatchIssue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var idx int64
	_, _ = fmt.Sscanf(r.PathValue("index"), "%d", &idx)

	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)

	m.mu.Lock()
	defer m.mu.Unlock()

	issue, ok := m.issues[idx]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "issue not found"})
		return
	}

	for k, v := range req {
		issue[k] = v
	}
	issue["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	_ = json.NewEncoder(w).Encode(issue)
}

func (m *MockForgejoServer) handleCreateIssueComment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var idx int64
	_, _ = fmt.Sscanf(r.PathValue("index"), "%d", &idx)

	var req struct {
		Body string `json:"body"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	m.mu.Lock()
	m.comments[idx] = append(m.comments[idx], req.Body)
	m.mu.Unlock()

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":         time.Now().UnixNano(),
		"body":       req.Body,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (m *MockForgejoServer) handleCreateBranch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		BranchName string `json:"new_branch_name"`
		OldRefName string `json:"old_ref_name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	name := req.BranchName
	if name == "" {
		name = "feature/42-gitops-promotion-pipeline"
	}

	m.mu.Lock()
	m.branches[name] = true
	m.mu.Unlock()

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name": name,
		"commit": map[string]any{
			"id":        fmt.Sprintf("sha-%x", time.Now().UnixNano()),
			"message":   "Initial branch creation",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
		"protected": false,
	})
}

func (m *MockForgejoServer) handleListBranches(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]map[string]any, 0, len(m.branches))
	for b := range m.branches {
		list = append(list, map[string]any{
			"name":      b,
			"protected": b == "main",
		})
	}
	_ = json.NewEncoder(w).Encode(list)
}

func (m *MockForgejoServer) handleCreateFileCommit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := r.PathValue("path")

	var req struct {
		Branch  string `json:"branch"`
		Content string `json:"content"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	commitSHA := fmt.Sprintf("sha-%x", time.Now().UnixNano())

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"content": map[string]any{
			"name": path,
			"path": path,
			"sha":  commitSHA,
		},
		"commit": map[string]any{
			"sha":     commitSHA,
			"message": req.Message,
			"author": map[string]any{
				"name":  "Elise Dev",
				"email": "elise.dev@oci.local",
			},
		},
	})
}

func (m *MockForgejoServer) handleCreatePullRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Head  string `json:"head"`
		Base  string `json:"base"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	m.mu.Lock()
	defer m.mu.Unlock()

	prID := int64(42)
	pr := map[string]any{
		"id":         prID,
		"number":     prID,
		"title":      req.Title,
		"body":       req.Body,
		"state":      "open",
		"head":       map[string]any{"ref": req.Head},
		"base":       map[string]any{"ref": req.Base},
		"merged":     false,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	m.pulls[prID] = pr

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(pr)
}

func (m *MockForgejoServer) handleListPullRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]map[string]any, 0, len(m.pulls))
	for _, pr := range m.pulls {
		list = append(list, pr)
	}
	_ = json.NewEncoder(w).Encode(list)
}

func (m *MockForgejoServer) handleGetPullRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var idx int64
	_, _ = fmt.Sscanf(r.PathValue("index"), "%d", &idx)

	m.mu.RLock()
	defer m.mu.RUnlock()

	pr, ok := m.pulls[idx]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "pull request not found"})
		return
	}
	_ = json.NewEncoder(w).Encode(pr)
}

func (m *MockForgejoServer) handleCreatePullReview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var idx int64
	_, _ = fmt.Sscanf(r.PathValue("index"), "%d", &idx)

	var req struct {
		Event string `json:"event"`
		Body  string `json:"body"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	event := req.Event
	if event == "" {
		event = "APPROVED"
	}

	review := map[string]any{
		"id":           time.Now().UnixNano(),
		"state":        event,
		"body":         req.Body,
		"reviewer":     "emp-human-marcus",
		"submitted_at": time.Now().UTC().Format(time.RFC3339),
	}

	m.mu.Lock()
	m.reviews[idx] = append(m.reviews[idx], review)
	m.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(review)
}

func (m *MockForgejoServer) handleMergePullRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var idx int64
	_, _ = fmt.Sscanf(r.PathValue("index"), "%d", &idx)

	m.mu.Lock()
	if pr, ok := m.pulls[idx]; ok {
		pr["merged"] = true
		pr["state"] = "closed"
	}
	if issue, ok := m.issues[idx]; ok {
		issue["state"] = "closed"
	}
	m.mu.Unlock()

	commitSHA := fmt.Sprintf("sha-%x", time.Now().UnixNano())
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"merged":  true,
		"message": fmt.Sprintf("Pull request #%d merged into main", idx),
		"sha":     commitSHA,
	})
}

func (m *MockForgejoServer) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		UserName    string `json:"username"`
		FullName    string `json:"full_name"`
		Description string `json:"description"`
		Visibility  string `json:"visibility"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid request payload"})
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	org := map[string]any{
		"id":          int64(len(m.orgs) + 1),
		"username":    req.UserName,
		"full_name":   req.FullName,
		"description": req.Description,
		"visibility":  req.Visibility,
	}
	m.orgs[req.UserName] = org

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(org)
}

func (m *MockForgejoServer) handleGetOrg(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	orgName := r.PathValue("org")

	m.mu.RLock()
	defer m.mu.RUnlock()

	org, ok := m.orgs[orgName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "organization not found"})
		return
	}
	_ = json.NewEncoder(w).Encode(org)
}

func (m *MockForgejoServer) handleCreateOrgRepo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	orgName := r.PathValue("org")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Private     bool   `json:"private"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid request payload"})
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s/%s", orgName, req.Name)
	repo := map[string]any{
		"id":                int64(len(m.repos) + 100),
		"name":              req.Name,
		"full_name":         key,
		"description":       req.Description,
		"private":           req.Private,
		"default_branch":    "main",
		"has_actions":       true,
		"has_issues":        true,
		"has_pull_requests": true,
	}
	m.repos[key] = repo

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(repo)
}

func (m *MockForgejoServer) handleGetRepo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	owner := r.PathValue("owner")
	repoName := r.PathValue("repo")
	key := fmt.Sprintf("%s/%s", owner, repoName)

	m.mu.RLock()
	defer m.mu.RUnlock()

	repo, ok := m.repos[key]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "repository not found"})
		return
	}
	_ = json.NewEncoder(w).Encode(repo)
}

func (m *MockForgejoServer) handlePatchRepo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	owner := r.PathValue("owner")
	repoName := r.PathValue("repo")
	key := fmt.Sprintf("%s/%s", owner, repoName)

	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)

	m.mu.Lock()
	defer m.mu.Unlock()

	repo, ok := m.repos[key]
	if !ok {
		// Auto-vivify if not existing for convenience in testing
		repo = map[string]any{
			"id":                int64(len(m.repos) + 100),
			"name":              repoName,
			"full_name":         key,
			"default_branch":    "main",
			"has_actions":       true,
			"has_issues":        true,
			"has_pull_requests": true,
		}
		m.repos[key] = repo
	}

	for k, v := range req {
		repo[k] = v
	}
	_ = json.NewEncoder(w).Encode(repo)
}

func (m *MockForgejoServer) handleCreateIssue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid request payload"})
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	idx := int64(len(m.issues) + 1)
	if strings.Contains(req.Title, "42") || strings.Contains(req.Title, "gitops") {
		idx = 42
	}
	issue := map[string]any{
		"id":         idx,
		"number":     idx,
		"title":      req.Title,
		"body":       req.Body,
		"state":      "open",
		"assignee":   nil,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	m.issues[idx] = issue

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(issue)
}
