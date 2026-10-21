package sim

import (
	simfj "github.com/orange-cat-investments/oci/pkg/cots/sim/forgejo"
)

// Client re-exports the simulation client from pkg/cots/sim/forgejo.
type Client = simfj.Client

// Config re-exports Config.
type Config = simfj.Config

// CreateUserOption re-exports CreateUserOption.
type CreateUserOption = simfj.CreateUserOption

// NewClient initializes a simulation Forgejo client.
func NewClient(cfg Config) *Client {
	return simfj.NewClient(cfg)
}
