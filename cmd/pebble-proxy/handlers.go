package main

import (
	"log/slog"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
)

type GatewayHandler struct {
	bllClient internalbllv1.InternalBLLServiceClient
	logger    *slog.Logger
}

func NewGatewayHandler(bllClient internalbllv1.InternalBLLServiceClient, logger *slog.Logger) *GatewayHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &GatewayHandler{
		bllClient: bllClient,
		logger:    logger,
	}
}
