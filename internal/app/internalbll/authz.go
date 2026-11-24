package internalbll

import (
	"context"

	"github.com/orange-cat-investments/oci/pkg/authz/policy"
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

	dec, err := s.policyEngine.EvaluateWorkforce(ctx, policy.WorkforceInput{
		Action: action,
		Actor: policy.Actor{
			ID:         actor.ActorID,
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
