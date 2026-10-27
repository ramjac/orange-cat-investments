package customerbff

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/httputil"
	"github.com/orange-cat-investments/oci/pkg/reporting"
)

func (b *CustomerBFF) handleGDPRErasure(w http.ResponseWriter, r *http.Request) {
	sess := CustomerFromContext(r.Context())
	if sess == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	certResp, err := b.bllClient.ExecuteGDPRErasure(r.Context(), &customerbllv1.ExecuteGDPRErasureRequest{
		CustomerID: sess.CustomerID,
		Email:      sess.Email,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("gdpr erasure failed: %v", err))
		return
	}

	// Invalidate local session in Valkey/memory
	b.sessions.Delete(sess.SessionID)

	// Format response
	var certModel *reporting.GDPRDeletionCertificate
	if certResp.Certificate != nil {
		certModel = &reporting.GDPRDeletionCertificate{
			CertificateID:        certResp.Certificate.CertificateID,
			CustomerIDHash:       certResp.Certificate.CustomerIDHash,
			CustomerEmailHash:    certResp.Certificate.CustomerEmailHash,
			LegalBasis:           certResp.Certificate.LegalBasis,
			SupervisoryAuthority: certResp.Certificate.SupervisoryAuthority,
			TablesScrubbed:       certResp.Certificate.AnonymizedTables,
			SessionsPurgedCount:  int(certResp.Certificate.SessionsPurgedCount),
			RecordsScrubbedCount: int(certResp.Certificate.RecordsScrubbedCount),
			DigitalSignature:     certResp.Certificate.ComplianceHash,
			ComplianceStatement:  certResp.Certificate.ComplianceStatement,
		}
		if t, err := time.Parse(time.RFC3339, certResp.Certificate.ErasedAt); err == nil {
			certModel.ErasureTimestamp = t
		} else {
			certModel.ErasureTimestamp = time.Now().UTC()
		}
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":      "erased",
		"certificate": certModel,
		"erased_at":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (b *CustomerBFF) handleGetGDPRCertificate(w http.ResponseWriter, r *http.Request) {
	certID := chi.URLParam(r, "id")
	if certID == "" {
		certID = r.PathValue("id")
	}
	if certID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "certificate id required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	certResp, err := b.bllClient.GetGDPRCertificate(r.Context(), &customerbllv1.GetGDPRCertificateRequest{
		CertificateID: certID,
	})
	if err != nil || certResp.Certificate == nil {
		httputil.WriteError(w, http.StatusNotFound, "gdpr certificate not found")
		return
	}

	certModel := &reporting.GDPRDeletionCertificate{
		CertificateID:        certResp.Certificate.CertificateID,
		CustomerIDHash:       certResp.Certificate.CustomerIDHash,
		CustomerEmailHash:    certResp.Certificate.CustomerEmailHash,
		LegalBasis:           certResp.Certificate.LegalBasis,
		SupervisoryAuthority: certResp.Certificate.SupervisoryAuthority,
		TablesScrubbed:       certResp.Certificate.AnonymizedTables,
		SessionsPurgedCount:  int(certResp.Certificate.SessionsPurgedCount),
		RecordsScrubbedCount: int(certResp.Certificate.RecordsScrubbedCount),
		DigitalSignature:     certResp.Certificate.ComplianceHash,
		ComplianceStatement:  certResp.Certificate.ComplianceStatement,
	}
	if t, err := time.Parse(time.RFC3339, certResp.Certificate.ErasedAt); err == nil {
		certModel.ErasureTimestamp = t
	}

	httputil.WriteJSON(w, http.StatusOK, certModel)
}

func (b *CustomerBFF) handleDownloadGDPRCertificate(w http.ResponseWriter, r *http.Request) {
	certID := chi.URLParam(r, "id")
	if certID == "" {
		certID = r.PathValue("id")
	}
	if certID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "certificate id required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	certResp, err := b.bllClient.GetGDPRCertificate(r.Context(), &customerbllv1.GetGDPRCertificateRequest{
		CertificateID: certID,
	})
	if err != nil || certResp.Certificate == nil {
		httputil.WriteError(w, http.StatusNotFound, "gdpr certificate not found")
		return
	}

	pdfBytes := certResp.PDFBytes
	if len(pdfBytes) == 0 && certResp.Certificate != nil {
		certModel := &reporting.GDPRDeletionCertificate{
			CertificateID:        certResp.Certificate.CertificateID,
			CustomerIDHash:       certResp.Certificate.CustomerIDHash,
			CustomerEmailHash:    certResp.Certificate.CustomerEmailHash,
			LegalBasis:           certResp.Certificate.LegalBasis,
			SupervisoryAuthority: certResp.Certificate.SupervisoryAuthority,
			TablesScrubbed:       certResp.Certificate.AnonymizedTables,
			SessionsPurgedCount:  int(certResp.Certificate.SessionsPurgedCount),
			RecordsScrubbedCount: int(certResp.Certificate.RecordsScrubbedCount),
			DigitalSignature:     certResp.Certificate.ComplianceHash,
			ComplianceStatement:  certResp.Certificate.ComplianceStatement,
		}
		if t, err := time.Parse(time.RFC3339, certResp.Certificate.ErasedAt); err == nil {
			certModel.ErasureTimestamp = t
		} else {
			certModel.ErasureTimestamp = time.Now().UTC()
		}
		pdfBytes, _ = reporting.GenerateGDPRCertificatePDF(certModel)
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"gdpr-certificate-%s.pdf\"", certID))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}
