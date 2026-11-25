package internalbll

import (
	"context"
	"errors"
	"strings"

	"github.com/orange-cat-investments/oci/pkg/authz/policy"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// authorizeWorkforce evaluates the embedded OPA workforce.rego policy for the incoming
// gRPC actor context. It fails closed and appends every policy denial to ops.audit_logs.
func (s *Server) authorizeWorkforce(ctx context.Context, action string, res policy.WorkforceResource) error {
	actor, _ := grpcutil.ActorFromIncomingContext(ctx)

	if s.policyEngine == nil {
		s.recordAudit(ctx, "authz.policy.denied", "policy", action, map[string]any{
			"policy":      "oci.authz.workforce",
			"action":      action,
			"actor_id":    actor.ActorID,
			"actor_role":  actor.Role,
			"resource_id": firstNonEmpty(res.ResourceID, res.FelineID, res.EmployeeID),
			"reasons":     []string{"policy engine not initialized"},
		})
		return status.Error(codes.PermissionDenied, "authorization policy engine not initialized")
	}

	// Owner-based rules (e.g. the self-approval restriction) compare the
	// actor against the resource owner's canonical workforce employee ID.
	// Forgejo-issued sessions carry emp-<username> actor IDs that can never
	// match a workforce UUID, so whenever the action targets an owned
	// resource the authenticated identity is correlated to its canonical
	// workforce employee ID. When the correlation cannot be established the
	// request is denied: an unresolved identity must never fall back to a
	// comparison that silently disables the self-approval restriction.
	policyActorID := actor.ActorID
	if res.EmployeeID != "" && actor.ActorID != "" && actor.Email != "" {
		canonical, corrErr := s.lookupWorkforceEmployeeByEmail(ctx, actor.Email)
		if corrErr != nil {
			s.recordAudit(ctx, "authz.policy.denied", "policy", action, map[string]any{
				"policy":      "oci.authz.workforce",
				"action":      action,
				"actor_id":    actor.ActorID,
				"actor_role":  actor.Role,
				"resource_id": firstNonEmpty(res.ResourceID, res.EmployeeID),
				"reasons":     []string{"unable to resolve authenticated actor to a canonical workforce identity: " + corrErr.Error()},
			})
			return status.Error(codes.PermissionDenied, "unable to resolve authenticated actor to a canonical workforce identity; owner-based authorization requires workforce correlation")
		}
		policyActorID = canonical
	}

	dec, err := s.policyEngine.EvaluateWorkforce(ctx, policy.WorkforceInput{
		Action: action,
		Actor: policy.Actor{
			ID:         policyActorID,
			Role:       actor.Role,
			Department: actor.Department,
		},
		Resource: res,
	})
	if err != nil || !dec.Allow {
		reason := dec.ReasonSummary()
		if err != nil && reason == "" {
			reason = err.Error()
		}
		if s.logger != nil {
			s.logger.Warn("OPA workforce policy denied request",
				"action", action,
				"actor_id", actor.ActorID,
				"actor_role", actor.Role,
				"resource_id", firstNonEmpty(res.ResourceID, res.FelineID, res.EmployeeID),
				"reason", reason,
			)
		}
		s.recordAudit(ctx, "authz.policy.denied", "policy", action, map[string]any{
			"policy":      dec.Policy,
			"action":      action,
			"actor_id":    actor.ActorID,
			"actor_role":  actor.Role,
			"resource_id": firstNonEmpty(res.ResourceID, res.FelineID, res.EmployeeID),
			"reasons":     dec.Reasons,
		})
		return status.Errorf(codes.PermissionDenied, "policy denied (%s): %s", action, reason)
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// lookupWorkforceEmployeeByEmail resolves an authenticated identity's email
// to its canonical workforce employee ID. Exactly one human employee must
// match; zero or multiple matches are treated as a correlation failure so
// owner-based policy checks fail closed.
func (s *Server) lookupWorkforceEmployeeByEmail(ctx context.Context, email string) (string, error) {
	if s.wfClient == nil {
		return "", errors.New("workforce client not configured")
	}
	resp, err := s.wfClient.ListEmployees(ctx, &workforcev1.ListEmployeesRequest{EmployeeType: "human"})
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", errors.New("empty workforce roster response")
	}
	var matchedID string
	matches := 0
	for _, emp := range resp.Employees {
		if emp == nil || emp.EmployeeID == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(emp.Email), strings.TrimSpace(email)) {
			matches++
			matchedID = emp.EmployeeID
			if matches > 1 {
				break
			}
		}
	}
	switch {
	case matches == 0:
		return "", errors.New("no workforce employee matches the authenticated identity")
	case matches > 1:
		return "", errors.New("ambiguous workforce identity match")
	default:
		return matchedID, nil
	}
}
