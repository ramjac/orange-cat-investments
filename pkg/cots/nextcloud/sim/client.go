package sim

import (
	simnc "github.com/orange-cat-investments/oci/pkg/cots/sim/nextcloud"
)

// Client re-exports the simulation client from pkg/cots/sim/nextcloud.
type Client = simnc.Client

// Config re-exports Config.
type Config = simnc.Config

// CreateUserOption re-exports CreateUserOption.
type CreateUserOption = simnc.CreateUserOption

// NewClient initializes a simulation Nextcloud client.
func NewClient(cfg Config) *Client {
	return simnc.NewClient(cfg)
}
