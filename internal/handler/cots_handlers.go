package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
)

func (h *WorkforceHandler) listGroupFolders(w http.ResponseWriter, r *http.Request) {
	cacheKey := "cache:cots:nextcloud:folders"
	if h.cache != nil {
		if raw, ok, _ := h.cache.GetRaw(r.Context(), cacheKey); ok {
			if cachevalkey.HandleConditionalETag(w, r, raw) {
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(raw)
			return
		}
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.ListGroupFolders(r.Context(), &internalbllv1.ListGroupFoldersRequest{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		folders := resp.GroupFolders
		if folders == nil {
			folders = []nextcloud.GroupFolder{}
		}
		respMap := map[string]any{
			"folders": folders,
			"total":   len(folders),
		}
		data, err := json.Marshal(respMap)
		if err != nil {
			writeJSON(w, http.StatusOK, respMap)
			return
		}
		if h.cache != nil {
			_ = h.cache.SetRaw(r.Context(), cacheKey, data, cachevalkey.TTLCOTSMetadata)
		}
		if cachevalkey.HandleConditionalETag(w, r, data) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) createGroupFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MountPoint string   `json:"mount_point"`
		Quota      string   `json:"quota"`
		Groups     []string `json:"groups"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.MountPoint == "" {
		writeError(w, http.StatusBadRequest, "mount_point cannot be empty")
		return
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.CreateGroupFolder(r.Context(), &internalbllv1.CreateGroupFolderRequest{
			MountPoint: req.MountPoint,
			Groups:     req.Groups,
			Quota:      req.Quota,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if h.cache != nil {
			_ = h.cache.Delete(r.Context(), "cache:cots:nextcloud:folders")
		}
		writeJSON(w, http.StatusCreated, resp.GroupFolder)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

const maxSearchQueryLength = 100

func validateSearchParam(param, paramName string) (string, error) {
	trimmed := strings.TrimSpace(param)
	if len(trimmed) > maxSearchQueryLength {
		return "", fmt.Errorf("%s parameter exceeds maximum length of %d characters", paramName, maxSearchQueryLength)
	}
	for _, r := range trimmed {
		if r < 32 || r == 127 {
			return "", fmt.Errorf("%s parameter contains invalid control characters", paramName)
		}
	}
	return trimmed, nil
}

func (h *WorkforceHandler) listDocuments(w http.ResponseWriter, r *http.Request) {
	deptRaw := r.URL.Query().Get("department")
	catRaw := r.URL.Query().Get("category")
	qRaw := r.URL.Query().Get("q")
	if qRaw == "" {
		qRaw = r.URL.Query().Get("query")
	}
	if qRaw == "" {
		qRaw = r.URL.Query().Get("search")
	}

	dept, err := validateSearchParam(deptRaw, "department")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cat, err := validateSearchParam(catRaw, "category")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	q, err := validateSearchParam(qRaw, "q")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.ListDocuments(r.Context(), &internalbllv1.ListDocumentsRequest{
			Department: dept,
			Category:   cat,
			Query:      q,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		docs := resp.Documents
		if docs == nil {
			docs = []nextcloud.Document{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"documents": docs,
			"total":     len(docs),
		})
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) getDocument(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "document id is required")
		return
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.GetDocument(r.Context(), &internalbllv1.GetDocumentRequest{DocumentID: id})
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp.Document)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) publishDocument(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DocumentID string   `json:"document_id"`
		Title      string   `json:"title"`
		Department string   `json:"department"`
		Category   string   `json:"category"`
		Content    string   `json:"content"`
		Author     string   `json:"author"`
		SharedWith []string `json:"shared_with"`
		ShareWith  []string `json:"share_with"`
		WebDAVPath string   `json:"webdav_path"`
		Version    string   `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "document title cannot be empty")
		return
	}
	shares := req.SharedWith
	if len(shares) == 0 {
		shares = req.ShareWith
	}
	doc := nextcloud.Document{
		DocumentID: req.DocumentID,
		Title:      req.Title,
		Department: req.Department,
		Category:   req.Category,
		Content:    req.Content,
		Author:     req.Author,
		SharedWith: shares,
		WebDAVPath: req.WebDAVPath,
		Version:    req.Version,
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.PublishDocument(r.Context(), &internalbllv1.PublishDocumentRequest{Document: doc})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, resp.Document)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) listCalendarEvents(w http.ResponseWriter, r *http.Request) {
	cal := r.URL.Query().Get("calendar")
	dept := r.URL.Query().Get("department")
	var fromDate, toDate time.Time
	if from := r.URL.Query().Get("from_date"); from != "" {
		fromDate, _ = time.Parse(time.RFC3339, from)
		if fromDate.IsZero() {
			fromDate, _ = time.Parse("2006-01-02", from)
		}
	}
	if to := r.URL.Query().Get("to_date"); to != "" {
		toDate, _ = time.Parse(time.RFC3339, to)
		if toDate.IsZero() {
			toDate, _ = time.Parse("2006-01-02", to)
		}
	}

	if h.bllClient != nil {
		fromStr := ""
		if !fromDate.IsZero() {
			fromStr = fromDate.Format(time.RFC3339)
		}
		toStr := ""
		if !toDate.IsZero() {
			toStr = toDate.Format(time.RFC3339)
		}
		resp, err := h.bllClient.ListCalendarEvents(r.Context(), &internalbllv1.ListCalendarEventsRequest{
			Calendar:   cal,
			Department: dept,
			FromDate:   fromStr,
			ToDate:     toStr,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		events := resp.Events
		if events == nil {
			events = []nextcloud.CalendarEvent{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"events": events,
			"total":  len(events),
		})
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) createCalendarEvent(w http.ResponseWriter, r *http.Request) {
	var ev nextcloud.CalendarEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if ev.Title == "" {
		writeError(w, http.StatusBadRequest, "event title cannot be empty")
		return
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.CreateCalendarEvent(r.Context(), &internalbllv1.CreateCalendarEventRequest{Event: ev})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, resp.Event)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) handleNextcloudChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Sender    string `json:"sender"`
		Recipient string `json:"recipient"`
		Room      string `json:"room"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	resp := map[string]any{
		"message_id": fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		"sender":     req.Sender,
		"recipient":  req.Recipient,
		"room":       req.Room,
		"status":     "delivered",
		"sent_at":    time.Now().UTC().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusCreated, resp)
}
