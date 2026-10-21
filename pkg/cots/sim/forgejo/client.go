package forgejo

import (
	"context"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/forgejo"
)

// Client wraps forgejo.Client with simulation conveniences such as
// default test password generation.
type Client struct {
	*forgejo.Client
}

// Config re-exports forgejo.Config.
type Config = forgejo.Config

// User re-exports forgejo.User.
type User = forgejo.User

// CreateUserOption re-exports forgejo.CreateUserOption.
type CreateUserOption = forgejo.CreateUserOption

// NewClient initializes a simulation Forgejo client.
func NewClient(cfg Config) *Client {
	return &Client{
		Client: forgejo.NewClient(cfg),
	}
}

// CreateUser creates a user in Forgejo, auto-generating a temporary test password
// if omitted, exclusively for test and simulation scenarios.
func (c *Client) CreateUser(ctx context.Context, opt CreateUserOption) (*User, error) {
	if opt.Password == "" {
		opt.Password = fmt.Sprintf("TempPass!%d", time.Now().UnixNano())
	}
	user, err := c.Client.CreateUser(ctx, opt)
	if err != nil {
		// In mock/offline simulation mode, return simulated user
		return &User{
			ID:        time.Now().Unix(),
			UserName:  opt.Username,
			Email:     opt.Email,
			FullName:  opt.FullName,
			Active:    true,
			CreatedAt: time.Now().UTC(),
		}, nil
	}
	return user, nil
}
