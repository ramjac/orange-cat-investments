package policy

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/open-policy-agent/opa/v1/rego"
)

//go:embed policies/*.rego
var embeddedPolicies embed.FS

// Actor describes the authenticated subject requesting an operation.
type Actor struct {
	ID         string `json:"id"`
	Role       string `json:"role"`
	Department string `json:"department,omitempty"`
}

// WorkforceResource describes the target entity for a workforce policy check.
type WorkforceResource struct {
	EmployeeID string `json:"employee_id,omitempty"`
	FelineID   string `json:"feline_id,omitempty"`
	ResourceID string `json:"resource_id,omitempty"`
}

// WorkforceInput is the structured OPA input for data.oci.authz.workforce.
type WorkforceInput struct {
	Action   string            `json:"action"`
	Actor    Actor             `json:"actor"`
	Resource WorkforceResource `json:"resource"`
}

// AccountState describes the customer account state for trading authorization.
type AccountState struct {
	Status         string  `json:"status"`
	BuyingPowerUSD float64 `json:"buying_power_usd"`
}

// FelineState describes the active feline executive safety state for trading authorization.
type FelineState struct {
	FelineID             string `json:"feline_id"`
	EmergencyMedicalHold bool   `json:"emergency_medical_hold"`
}

// OrderRequest describes the trade order parameters for policy evaluation.
type OrderRequest struct {
	Symbol               string  `json:"symbol"`
	Side                 string  `json:"side"`
	Quantity             float64 `json:"quantity"`
	OrderType            string  `json:"order_type"`
	EstimatedNotionalUSD float64 `json:"estimated_notional_usd"`
}

// TradingLimits configures policy thresholds for single-order notional caps.
type TradingLimits struct {
	MaxOrderNotionalUSD float64 `json:"max_order_notional_usd,omitempty"`
}

// TradingInput is the structured OPA input for data.oci.authz.trading.
type TradingInput struct {
	Actor   Actor         `json:"actor"`
	Account AccountState  `json:"account"`
	Feline  FelineState   `json:"feline"`
	Order   OrderRequest  `json:"order"`
	Limits  TradingLimits `json:"limits"`
}

// Decision captures the outcome of an embedded OPA policy evaluation.
type Decision struct {
	Allow   bool     `json:"allow"`
	Reasons []string `json:"reasons,omitempty"`
	Policy  string   `json:"policy"`
}

// ReasonSummary returns a single concise denial message suitable for HTTP 403 / gRPC PermissionDenied.
func (d Decision) ReasonSummary() string {
	if len(d.Reasons) == 0 {
		if d.Allow {
			return "allowed"
		}
		return "denied by authorization policy"
	}
	return strings.Join(d.Reasons, "; ")
}

// Engine embeds and pre-compiles OCI Rego policies for zero-hop in-process BLL evaluation.
type Engine struct {
	workforceQuery rego.PreparedEvalQuery
	tradingQuery   rego.PreparedEvalQuery
}

// NewEngine compiles all embedded .rego policy modules into prepared OPA evaluation queries.
func NewEngine(ctx context.Context) (*Engine, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	workforceBytes, err := embeddedPolicies.ReadFile("policies/workforce.rego")
	if err != nil {
		return nil, fmt.Errorf("read embedded workforce.rego: %w", err)
	}
	tradingBytes, err := embeddedPolicies.ReadFile("policies/trading.rego")
	if err != nil {
		return nil, fmt.Errorf("read embedded trading.rego: %w", err)
	}

	wfQuery, err := rego.New(
		rego.Query("result = data.oci.authz.workforce"),
		rego.Module("policies/workforce.rego", string(workforceBytes)),
	).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("compile workforce.rego policy: %w", err)
	}

	trQuery, err := rego.New(
		rego.Query("result = data.oci.authz.trading"),
		rego.Module("policies/trading.rego", string(tradingBytes)),
	).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("compile trading.rego policy: %w", err)
	}

	return &Engine{
		workforceQuery: wfQuery,
		tradingQuery:   trQuery,
	}, nil
}

// EvaluateWorkforce evaluates data.oci.authz.workforce and fails closed on any error.
func (e *Engine) EvaluateWorkforce(ctx context.Context, in WorkforceInput) (Decision, error) {
	if e == nil {
		return Decision{Allow: false, Policy: "oci.authz.workforce", Reasons: []string{"policy engine not initialized"}}, fmt.Errorf("policy engine is nil")
	}
	in.Actor.Role = strings.ToLower(strings.TrimSpace(in.Actor.Role))
	return evalPrepared(ctx, e.workforceQuery, "oci.authz.workforce", in)
}

// EvaluateTrading evaluates data.oci.authz.trading and fails closed on any error.
func (e *Engine) EvaluateTrading(ctx context.Context, in TradingInput) (Decision, error) {
	if e == nil {
		return Decision{Allow: false, Policy: "oci.authz.trading", Reasons: []string{"policy engine not initialized"}}, fmt.Errorf("policy engine is nil")
	}
	in.Actor.Role = strings.ToLower(strings.TrimSpace(in.Actor.Role))
	in.Account.Status = strings.ToLower(strings.TrimSpace(in.Account.Status))
	in.Order.Side = strings.ToLower(strings.TrimSpace(in.Order.Side))
	if in.Limits.MaxOrderNotionalUSD <= 0 {
		in.Limits.MaxOrderNotionalUSD = 250000
	}
	return evalPrepared(ctx, e.tradingQuery, "oci.authz.trading", in)
}

func evalPrepared(ctx context.Context, query rego.PreparedEvalQuery, policyName string, input any) (Decision, error) {
	rs, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return Decision{
			Allow:   false,
			Policy:  policyName,
			Reasons: []string{"policy evaluation error"},
		}, err
	}
	if len(rs) == 0 {
		return Decision{
			Allow:   false,
			Policy:  policyName,
			Reasons: []string{"empty policy result"},
		}, nil
	}
	raw, ok := rs[0].Bindings["result"].(map[string]any)
	if !ok {
		return Decision{
			Allow:   false,
			Policy:  policyName,
			Reasons: []string{"unexpected policy binding type"},
		}, nil
	}

	allow, _ := raw["allow"].(bool)
	var reasons []string
	if rawReasons, ok := raw["deny_reasons"].([]any); ok {
		for _, item := range rawReasons {
			if s, ok := item.(string); ok && s != "" {
				reasons = append(reasons, s)
			}
		}
		sort.Strings(reasons)
	}
	return Decision{
		Allow:   allow,
		Reasons: reasons,
		Policy:  policyName,
	}, nil
}
