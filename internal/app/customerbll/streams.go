package customerbll

import (
	"context"
	"time"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) getActiveFelines(ctx context.Context) ([]*workforcev1.Employee, error) {
	if s.wfClient == nil {
		return nil, nil
	}

	s.felineCacheMu.RLock()
	if time.Since(s.felinesAt) < 30*time.Second && s.felinesCached != nil {
		res := s.felinesCached
		s.felineCacheMu.RUnlock()
		return res, nil
	}
	s.felineCacheMu.RUnlock()

	s.felineCacheMu.Lock()
	defer s.felineCacheMu.Unlock()

	if time.Since(s.felinesAt) < 30*time.Second && s.felinesCached != nil {
		return s.felinesCached, nil
	}

	res, err := s.wfClient.ListEmployees(ctx, &workforcev1.ListEmployeesRequest{
		EmployeeType: "feline",
		Status:       "active",
	})
	if err != nil {
		if s.felinesCached != nil {
			return s.felinesCached, nil
		}
		return nil, err
	}

	s.felinesCached = res.Employees
	s.felinesAt = time.Now()
	return res.Employees, nil
}

func (s *Server) getVHRRecordsByFeline(ctx context.Context) (map[string][]*workforcev1.VHRRecord, error) {
	if s.wfClient == nil {
		return nil, nil
	}

	s.vhrCacheMu.RLock()
	if time.Since(s.vhrAt) < 30*time.Second && s.vhrCached != nil {
		res := s.vhrCached
		s.vhrCacheMu.RUnlock()
		return res, nil
	}
	s.vhrCacheMu.RUnlock()

	s.vhrCacheMu.Lock()
	defer s.vhrCacheMu.Unlock()

	if time.Since(s.vhrAt) < 30*time.Second && s.vhrCached != nil {
		return s.vhrCached, nil
	}

	res, err := s.wfClient.ListVHRRecords(ctx, &workforcev1.ListVHRRecordsRequest{})
	if err != nil {
		if s.vhrCached != nil {
			return s.vhrCached, nil
		}
		return nil, err
	}

	vhrByFeline := make(map[string][]*workforcev1.VHRRecord)
	for _, rec := range res.Records {
		vhrByFeline[rec.FelineID] = append(vhrByFeline[rec.FelineID], rec)
	}

	s.vhrCached = vhrByFeline
	s.vhrAt = time.Now()
	return vhrByFeline, nil
}

func buildFelineExecutiveSummary(emp *workforcev1.Employee, records []*workforcev1.VHRRecord) *customerbllv1.FelineExecutiveSummary {
	if emp == nil {
		return nil
	}

	healthStatus := "ACTIVE"
	whiskerSymmetry := "100%"
	purrFreq := 25.0
	preferredSunbeam := "Observation Lounge"

	if len(records) > 0 {
		latest := records[0]
		if latest.DentalScore >= 4 {
			healthStatus = "PEAK_PERFORMER"
			whiskerSymmetry = "99.8%"
			purrFreq = 26.2
		} else {
			healthStatus = "UNDER_OBSERVATION"
			whiskerSymmetry = "95.0%"
			purrFreq = 22.0
		}
		if latest.ClinicalNotes != "" {
			preferredSunbeam = latest.ClinicalNotes
		}
	}

	return &customerbllv1.FelineExecutiveSummary{
		EmployeeID:       emp.EmployeeID,
		Name:             emp.FirstName,
		Role:             emp.RoleTitle,
		HealthStatus:     healthStatus,
		WhiskerSymmetry:  whiskerSymmetry,
		PurrFrequencyHz:  purrFreq,
		PreferredSunbeam: preferredSunbeam,
		VHRRecordsCnt:    int32(len(records)),
	}
}

func (s *Server) GetESGTransparency(ctx context.Context, in *customerbllv1.GetESGTransparencyRequest) (*customerbllv1.GetESGTransparencyResponse, error) {
	felinesList := []*customerbllv1.FelineExecutiveSummary{}
	vhrRecordsCount := 0

	emps, _ := s.getActiveFelines(ctx)
	vhrByFeline, _ := s.getVHRRecordsByFeline(ctx)

	for _, emp := range emps {
		if emp == nil {
			continue
		}
		var records []*workforcev1.VHRRecord
		if vhrByFeline != nil {
			records = vhrByFeline[emp.EmployeeID]
		}
		vhrRecordsCount += len(records)
		if summary := buildFelineExecutiveSummary(emp, records); summary != nil {
			felinesList = append(felinesList, summary)
		}
	}

	var incidents []*workforcev1.WorkplaceIncident
	if s.wfClient != nil {
		if incRes, err := s.wfClient.ListIncidents(ctx, &workforcev1.ListIncidentsRequest{}); err == nil && incRes != nil {
			incidents = incRes.Incidents
		}
	}
	incidentsCnt := len(incidents)
	welfareScore := 5.0
	if incidentsCnt > 0 {
		welfareScore -= float64(incidentsCnt) * 0.02
	}
	if welfareScore < 1.0 {
		welfareScore = 1.0
	}

	adherencePct := 100.0
	if vhrRecordsCount == 0 && len(felinesList) > 0 {
		adherencePct = 98.5
	}

	return &customerbllv1.GetESGTransparencyResponse{
		OverallWelfareScore:    welfareScore,
		VeterinaryAdherencePct: adherencePct,
		NutritionalCompliance:  "100% Grain-Free Prescription Salmon & Turkey",
		AvgDailyNapHours:       14.6,
		PerchComfortRating:     "5.0 / 5.0 (Ergonomic Thermal Cushioning)",
		WorkplaceIncidentsCnt:  int32(incidentsCnt),
		MedicalHoldsIssuedYTD:  0,
		FelineExecutives:       felinesList,
		Incidents:              incidents,
	}, nil
}

func (s *Server) ListCameraStreams(ctx context.Context, in *customerbllv1.ListCameraStreamsRequest) (*customerbllv1.ListCameraStreamsResponse, error) {
	if s.ciClient == nil {
		return &customerbllv1.ListCameraStreamsResponse{Streams: []*coreinvestv1.CameraStream{}}, nil
	}
	res, err := s.ciClient.ListCameraStreams(ctx, &coreinvestv1.ListCameraStreamsRequest{})
	if err != nil {
		return &customerbllv1.ListCameraStreamsResponse{Streams: []*coreinvestv1.CameraStream{}}, nil
	}
	return &customerbllv1.ListCameraStreamsResponse{Streams: res.Streams}, nil
}

func (s *Server) CompleteBacktestRun(ctx context.Context, in *customerbllv1.CompleteBacktestRunRequest) (*customerbllv1.CompleteBacktestRunResponse, error) {
	if s.ciClient == nil {
		return nil, status.Errorf(codes.Unavailable, "core invest domain client is not initialized")
	}
	res, err := s.ciClient.CompleteBacktestRun(ctx, &coreinvestv1.CompleteBacktestRunRequest{
		RunID:   in.RunID,
		Status:  in.Status,
		Results: in.Results,
	})
	if err != nil {
		return nil, err
	}
	return &customerbllv1.CompleteBacktestRunResponse{Run: res.Run}, nil
}

func (s *Server) StreamPortfolioValuation(in *coreinvestv1.StreamPortfolioValuationRequest, stream customerbllv1.CustomerBLLService_StreamPortfolioValuationServer) error {
	if in == nil || in.PortfolioID == "" {
		return status.Error(codes.InvalidArgument, "portfolio_id required")
	}
	if s.ciClient != nil {
		upstream, err := s.ciClient.StreamPortfolioValuation(stream.Context(), in)
		if err == nil {
			for {
				evt, err := upstream.Recv()
				if err != nil {
					break
				}
				if err := stream.Send(evt); err != nil {
					return err
				}
			}
			return nil
		}
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for i := 0; i < 5; i++ {
		select {
		case <-stream.Context().Done():
			return nil
		case t := <-ticker.C:
			val := 154200.50 + float64(i)*2.75
			if err := stream.Send(&coreinvestv1.PortfolioValuationEvent{
				PortfolioID:        in.PortfolioID,
				TotalBalanceUSD:    val,
				CashBalanceUSD:     15000.00,
				InvestedBalanceUSD: val - 15000.00,
				ChangePct:          1.42,
				Timestamp:          t.UTC().Format(time.RFC3339),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
