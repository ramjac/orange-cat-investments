package sim

import (
	simsn "github.com/orange-cat-investments/oci/pkg/cots/sim/snipeit"
)

// Client re-exports the simulation client from pkg/cots/sim/snipeit.
type Client = simsn.Client

// Config re-exports Config.
type Config = simsn.Config

// NewClient initializes a simulation Snipe-IT client.
func NewClient(cfg Config) *Client {
	return simsn.NewClient(cfg)
}
