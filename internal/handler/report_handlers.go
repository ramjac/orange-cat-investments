package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
)

func (h *WorkforceHandler) getBIMetrics(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = time.Now().UTC().Format("2006-01")
	}

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.GetBIMetrics(r.Context(), &internalbllv1.GetBIMetricsRequest{Period: period})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get BI metrics: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"period":    period,
		"financial": resp.FinancialMetrics,
		"workforce": resp.WorkforceMetrics,
	})
}

func (h *WorkforceHandler) listExecutiveReports(w http.ResponseWriter, r *http.Request) {
	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.ListExecutiveReports(r.Context(), &workforcev1.ListExecutiveReportsRequest{Limit: 50, Offset: 0})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	reports := resp.Reports
	if reports == nil {
		reports = []*workforcev1.ExecutiveReport{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": reports,
		"total": len(reports),
	})
}

func (h *WorkforceHandler) generateMonthlyReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Period      string `json:"period"`
		GeneratedBy string `json:"generated_by"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Period == "" {
		req.Period = time.Now().UTC().Format("2006-01")
	}
	if req.GeneratedBy == "" {
		req.GeneratedBy = "authorized-executive"
	}

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.GenerateMonthlyReport(r.Context(), &internalbllv1.GenerateMonthlyReportRequest{
		Period:      req.Period,
		GeneratedBy: req.GeneratedBy,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate monthly report: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp.Report)
}

func (h *WorkforceHandler) getExecutiveReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		id = parts[len(parts)-1]
	}

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.GetExecutiveReport(r.Context(), &workforcev1.GetExecutiveReportRequest{ReportID: id})
	if err != nil {
		writeError(w, http.StatusNotFound, "executive report not found: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp.Report)
}

func (h *WorkforceHandler) downloadExecutiveReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	format := strings.ToLower(r.URL.Query().Get("format"))

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.DownloadExecutiveReport(r.Context(), &internalbllv1.DownloadExecutiveReportRequest{
		ReportID: id,
		Format:   format,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "failed to download executive report: "+err.Error())
		return
	}

	if resp.ContentType != "" {
		w.Header().Set("Content-Type", resp.ContentType)
	}
	if resp.Filename != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, resp.Filename))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.Content)
}
