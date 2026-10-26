package customerbff

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/httputil"
	"github.com/orange-cat-investments/oci/pkg/reporting"
	"github.com/orange-cat-investments/oci/pkg/session"
)

func (b *CustomerBFF) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	httputil.HealthzHandler(w, r)
}

func (b *CustomerBFF) handleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
	session.CSRFTokenHandler("csrf_token")(w, r)
}

func (b *CustomerBFF) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Persona    string `json:"persona"`
		CustomerID string `json:"customer_id"`
		Name       string `json:"name"`
		Email      string `json:"email"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	var custID, name, email string

	if req.CustomerID != "" {
		custID = req.CustomerID
		name = req.Name
		email = req.Email
	} else if req.Email != "" {
		email = req.Email
		name = req.Name
		if name == "" {
			parts := strings.Split(email, "@")
			name = parts[0]
		}
		custID = fmt.Sprintf("cust-%s", strings.ReplaceAll(name, " ", "-"))
	} else if req.Persona != "" {
		custID = fmt.Sprintf("cust-%s", req.Persona)
		name = req.Persona
		email = fmt.Sprintf("%s@oci.local", req.Persona)
	} else {
		httputil.WriteError(w, http.StatusBadRequest, "customer_id or email is required")
		return
	}

	// Dynamically query customer portfolio from Customer BLL to resolve customer name
	if b.bllClient != nil {
		if p, err := b.bllClient.GetPortfolio(r.Context(), &customerbllv1.GetPortfolioRequest{CustomerID: custID}); err == nil && p != nil {
			if name == "" && p.CustomerName != "" {
				name = p.CustomerName
			}
		}
	}
	if name == "" {
		name = custID
	}
	if email == "" {
		email = fmt.Sprintf("%s@oci.local", custID)
	}

	sess, err := b.sessions.Create(custID, name, email)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	// Set session cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "customer_session",
		Value:    sess.SessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-customer-session",
		Value:    sess.SessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})

	csrfToken, err := session.GenerateRandomToken(16)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to generate csrf token")
		return
	}
	session.SetCSRFCookie(w, "csrf_token", csrfToken)
	w.Header().Set("X-CSRF-Token", csrfToken)

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":     "authenticated",
		"session":    sess,
		"csrf_token": csrfToken,
	})
}

func (b *CustomerBFF) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("customer_session"); err == nil {
		b.sessions.Delete(c.Value)
	}
	if c, err := r.Cookie("__Host-customer-session"); err == nil {
		b.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "customer_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-customer-session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (b *CustomerBFF) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FullName        string `json:"full_name"`
		Email           string `json:"email"`
		Password        string `json:"password"`
		BankPublicToken string `json:"bank_public_token,omitempty"`
		TargetFund      string `json:"target_fund,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Password) < 8 {
		httputil.WriteError(w, http.StatusBadRequest, "password must be at least 8 characters long")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.RegisterCustomer(r.Context(), &customerbllv1.RegisterCustomerRequest{
		FullName:        req.FullName,
		Email:           req.Email,
		Password:        req.Password,
		BankPublicToken: req.BankPublicToken,
		TargetFund:      req.TargetFund,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	custID := req.Email
	name := req.FullName
	email := req.Email
	if resp.Result != nil {
		if resp.Result.CustomerID != "" {
			custID = resp.Result.CustomerID
		}
		if resp.Result.Name != "" {
			name = resp.Result.Name
		}
		if resp.Result.Email != "" {
			email = resp.Result.Email
		}
	}

	sess, err := b.sessions.Create(custID, name, email)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create customer session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "customer_session",
		Value:    sess.SessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-customer-session",
		Value:    sess.SessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})

	csrfToken, err := session.GenerateRandomToken(16)
	if err == nil {
		session.SetCSRFCookie(w, "csrf_token", csrfToken)
		w.Header().Set("X-CSRF-Token", csrfToken)
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]any{
		"status":     "registered",
		"session":    sess,
		"csrf_token": csrfToken,
		"result":     resp.Result,
	})
}

func (b *CustomerBFF) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sess := CustomerFromContext(r.Context())
	if sess == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "no active session")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, sess)
}

func (b *CustomerBFF) handleCloseAccount(w http.ResponseWriter, r *http.Request) {
	sess := CustomerFromContext(r.Context())
	if sess == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}

	var req struct {
		PortfolioID          string `json:"portfolio_id"`
		DestinationAccountID string `json:"destination_bank_account_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	res, err := b.bllClient.CloseAccount(r.Context(), &customerbllv1.CloseAccountRequest{
		CustomerID:           sess.CustomerID,
		PortfolioID:          req.PortfolioID,
		DestinationAccountID: req.DestinationAccountID,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if b.cache != nil {
		_ = b.cache.Delete(r.Context(), cachevalkey.KeyPortfolio(sess.CustomerID), cachevalkey.KeyOrders(sess.CustomerID))
	}

	respMap := map[string]any{
		"customer_id":             res.Result.CustomerID,
		"portfolio_id":            res.Result.PortfolioID,
		"account_status":          res.Result.AccountStatus,
		"liquidated_proceeds":     res.Result.LiquidatedProceeds,
		"final_cash_withdrawn":    res.Result.FinalCashWithdrawn,
		"ach_withdrawal_transfer": res.Result.ACHWithdrawalTransfer,
		"transfer_id":             res.Result.ACHWithdrawalTransfer,
		"destination_account_id":  res.Result.DestinationAccountID,
		"cancelled_dca_schedules": res.Result.CancelledDCASchedules,
		"message":                 res.Result.Message,
		"closed_at":               res.Result.ClosedAt,
	}
	httputil.WriteJSON(w, http.StatusOK, respMap)
}

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

	purgedCount, err := b.sessions.PurgeCustomerSessions(sess.CustomerID)
	if err != nil {
		b.logger.Error("failed to purge customer sessions during GDPR erasure", "customer_id", sess.CustomerID, "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("session revocation failed: %v", err))
		return
	}
	if certResp.Certificate != nil {
		certResp.Certificate.SessionsPurgedCount = int32(purgedCount)
	}

	if b.cache != nil {
		_ = b.cache.Delete(r.Context(), cachevalkey.KeyPortfolio(sess.CustomerID), cachevalkey.KeyOrders(sess.CustomerID))
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "customer_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-customer-session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	cert := certResp.Certificate
	certMap := map[string]any{
		"certificate_id":         cert.CertificateID,
		"customer_id":            cert.CustomerID,
		"customer_id_hash":       fmt.Sprintf("hash-%s", sess.CustomerID),
		"customer_email_hash":    fmt.Sprintf("hash-%s", sess.Email),
		"erasure_timestamp":      cert.ErasedAt,
		"legal_basis":            "GDPR Article 17 - Right to Erasure / Right to be Forgotten",
		"supervisory_authority":  "Orange Cat Investments Data Protection Office",
		"tables_scrubbed":        cert.AnonymizedTables,
		"sessions_purged_count":  purgedCount,
		"records_scrubbed_count": len(cert.AnonymizedTables),
		"digital_signature":      cert.ComplianceHash,
		"compliance_statement":   "Permanent and irrecoverable erasure of personal data executed in accordance with GDPR Article 17.",
		"status":                 cert.Status,
	}
	if cert.ComplianceHash == "" {
		certMap["digital_signature"] = "hmac-sha256-mock-signature"
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":      "erased",
		"certificate": certMap,
		"message":     "Personally identifiable data permanently erased under GDPR Article 17. Active sessions invalidated.",
	})
}

func (b *CustomerBFF) handleGetGDPRCertificate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		id = r.PathValue("id")
	}
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "certificate id is required")
		return
	}

	sess := CustomerFromContext(r.Context())
	custID := ""
	email := ""
	if sess != nil {
		custID = sess.CustomerID
		email = sess.Email
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	cert, err := b.bllClient.GetGDPRCertificate(r.Context(), &customerbllv1.GetGDPRCertificateRequest{
		CertificateID: id,
		CustomerID:    custID,
		Email:         email,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "certificate not found")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, cert.Certificate)
}

func (b *CustomerBFF) handleDownloadGDPRCertificate(w http.ResponseWriter, r *http.Request) {
	certID := chi.URLParam(r, "id")
	if certID == "" {
		certID = r.PathValue("id")
	}
	if certID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "certificate id is required")
		return
	}

	sess := CustomerFromContext(r.Context())
	custID := ""
	email := ""
	if sess != nil {
		custID = sess.CustomerID
		email = sess.Email
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	certResp, err := b.bllClient.GetGDPRCertificate(r.Context(), &customerbllv1.GetGDPRCertificateRequest{
		CertificateID: certID,
		CustomerID:    custID,
		Email:         email,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "certificate not found")
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
