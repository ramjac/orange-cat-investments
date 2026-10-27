package core_invest

import (
	"context"
	"errors"

	"github.com/orange-cat-investments/oci/internal/repository/core_invest"
)

func (s *coreInvestService) GetPortfolio(ctx context.Context, customerOrPortfolioID string) (*core_invest.Portfolio, error) {
	if customerOrPortfolioID == "" {
		return nil, errors.New("customer_id or portfolio_id is required")
	}
	p, err := s.repo.GetPortfolioByCustomerID(ctx, customerOrPortfolioID)
	if err == nil && p != nil {
		return p, nil
	}
	return s.repo.GetPortfolioByID(ctx, customerOrPortfolioID)
}

func (s *coreInvestService) ListPortfolios(ctx context.Context) ([]*core_invest.Portfolio, error) {
	return s.repo.ListPortfolios(ctx)
}

func (s *coreInvestService) CreatePortfolio(ctx context.Context, p *core_invest.Portfolio) (*core_invest.Portfolio, error) {
	if p == nil || p.CustomerID == "" {
		return nil, errors.New("customer_id is required to create portfolio")
	}
	return s.repo.CreatePortfolio(ctx, p)
}

func (s *coreInvestService) CreateCustomerAPIKey(ctx context.Context, key *core_invest.CustomerAPIKey) (*core_invest.CustomerAPIKey, error) {
	if key == nil || key.CustomerID == "" {
		return nil, errors.New("customer_id is required to create api key")
	}
	return s.repo.CreateAPIKey(ctx, key)
}

func (s *coreInvestService) ListCustomerAPIKeys(ctx context.Context, customerID string) ([]*core_invest.CustomerAPIKey, error) {
	if customerID == "" {
		return nil, errors.New("customer_id is required")
	}
	return s.repo.ListAPIKeys(ctx, customerID)
}

func (s *coreInvestService) RevokeCustomerAPIKey(ctx context.Context, customerID, keyID string) error {
	if customerID == "" || keyID == "" {
		return errors.New("customer_id and key_id are required")
	}
	return s.repo.RevokeAPIKey(ctx, customerID, keyID)
}

func (s *coreInvestService) CreateWebhookSubscription(ctx context.Context, sub *core_invest.WebhookSubscription) (*core_invest.WebhookSubscription, error) {
	if sub == nil || sub.CustomerID == "" || sub.TargetURL == "" {
		return nil, errors.New("customer_id and target_url are required")
	}
	return s.repo.CreateWebhook(ctx, sub)
}

func (s *coreInvestService) ListWebhookSubscriptions(ctx context.Context, customerID string) ([]*core_invest.WebhookSubscription, error) {
	if customerID == "" {
		return nil, errors.New("customer_id is required")
	}
	return s.repo.ListWebhooks(ctx, customerID)
}

func (s *coreInvestService) DeleteWebhookSubscription(ctx context.Context, customerID, subscriptionID string) error {
	if customerID == "" || subscriptionID == "" {
		return errors.New("customer_id and subscription_id are required")
	}
	return s.repo.DeleteWebhook(ctx, customerID, subscriptionID)
}
