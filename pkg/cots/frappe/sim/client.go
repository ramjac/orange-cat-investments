package sim

import (
	simfr "github.com/orange-cat-investments/oci/pkg/cots/sim/frappe"
)

// Client re-exports the simulation client from pkg/cots/sim/frappe.
type Client = simfr.Client

// Config re-exports Config.
type Config = simfr.Config

// NewClient initializes a simulation Frappe client.
func NewClient(cfg Config) *Client {
	return simfr.NewClient(cfg)
}
