package server

import (
	"context"

	healthv1 "apiPos/gen/health/v1"
	"apiPos/internal/database"
)

// HealthHandler implements the HealthService gRPC server.
type HealthHandler struct {
	healthv1.UnimplementedHealthServiceServer
	db database.Service
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db database.Service) *HealthHandler {
	return &HealthHandler{db: db}
}

// Check returns the health status of the service.
func (h *HealthHandler) Check(ctx context.Context, req *healthv1.CheckRequest) (*healthv1.CheckResponse, error) {
	dbHealth := h.db.Health()

	return &healthv1.CheckResponse{
		Status:  dbHealth["status"],
		Details: dbHealth,
	}, nil
}
