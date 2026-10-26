package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (h *WorkforceHandler) getBIMetrics(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = time.Now().UTC().Format("2006-01")
	}

	fin, err := h.wfSvc.GetFinancialMetrics(r.Context(), period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get financial metrics: "+err.Error())
		return
	}

	wf, err := h.wfSvc.GetWorkforceMetrics(r.Context(), period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get workforce metrics: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"period":    period,
		"financial": fin,
		"workforce": wf,
	})
}

func (h *WorkforceHandler) listExecutiveReports(w http.ResponseWriter, r *http.Request) {
	reports, err := h.wfSvc.ListExecutiveReports(r.Context(), 50, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
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

	rep, err := h.wfSvc.GenerateMonthlyReport(r.Context(), req.Period, req.GeneratedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate monthly report: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rep)
}

func (h *WorkforceHandler) getExecutiveReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		id = parts[len(parts)-1]
	}

	rep, err := h.wfSvc.GetExecutiveReport(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "executive report not found: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (h *WorkforceHandler) downloadExecutiveReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	rep, err := h.wfSvc.GetExecutiveReport(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "executive report not found: "+err.Error())
		return
	}

	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="executive_summary_report_%s.csv"`, rep.Period))
		csvContent := ""
		if rep.CSVContent != nil && *rep.CSVContent != "" {
			csvContent = *rep.CSVContent
		} else if rep.WebDAVPathCSV != nil && *rep.WebDAVPathCSV != "" {
			if content, err := h.wfSvc.DownloadWebDAVFile(r.Context(), *rep.WebDAVPathCSV); err == nil && len(content) > 0 {
				csvContent = string(content)
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(csvContent))
		return
	}

	// Default to PDF
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="executive_summary_report_%s.pdf"`, rep.Period))
	if len(rep.PDFContent) > 0 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(rep.PDFContent)
		return
	}
	if rep.WebDAVPathPDF != nil && *rep.WebDAVPathPDF != "" {
		if content, err := h.wfSvc.DownloadWebDAVFile(r.Context(), *rep.WebDAVPathPDF); err == nil && len(content) > 0 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	if rep.CSVContent != nil {
		_, _ = w.Write([]byte(*rep.CSVContent))
	}
}
