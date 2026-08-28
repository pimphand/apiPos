package handler

import (
	"context"

	cashierpb "apiPos/gen/cashier/v1"
	"apiPos/internal/apperror"
	"apiPos/internal/cashier"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// CashierHandler implements the CashierService gRPC server
type CashierHandler struct {
	cashierpb.UnimplementedCashierServiceServer
	service *cashier.Service
	repos   *repository.Repositories
}

// NewCashierHandler creates a new cashier Handler
func NewCashierHandler(service *cashier.Service, repos *repository.Repositories) *CashierHandler {
	return &CashierHandler{service: service, repos: repos}
}

// OpenShift opens a new cashier shift
func (h *CashierHandler) OpenShift(ctx context.Context, req *cashierpb.OpenShiftRequest) (*cashierpb.ShiftResponse, error) {
	userID := getUserIDFromContext(ctx)

	shift, err := h.service.OpenShift(ctx, cashier.OpenShiftRequest{
		StoreID:     req.StoreId,
		OpeningCash: req.OpeningCash,
		UserID:      userID,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &cashierpb.ShiftResponse{
		Shift: toProtoShift(shift, h.repos),
	}, nil
}

// CloseShift closes the current shift
func (h *CashierHandler) CloseShift(ctx context.Context, req *cashierpb.CloseShiftRequest) (*cashierpb.ShiftResponse, error) {
	shift, err := h.service.CloseShift(ctx, cashier.CloseShiftRequest{
		ShiftID:     req.ShiftId,
		ClosingCash: req.ClosingCash,
		ActualCash:  req.ActualCash,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &cashierpb.ShiftResponse{
		Shift: toProtoShift(shift, h.repos),
	}, nil
}

// GetCurrentShift gets the current open shift for a user
func (h *CashierHandler) GetCurrentShift(ctx context.Context, req *cashierpb.GetCurrentShiftRequest) (*cashierpb.ShiftResponse, error) {
	userID := getUserIDFromContext(ctx)

	shift, err := h.service.GetCurrentShift(ctx, userID)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &cashierpb.ShiftResponse{
		Shift: toProtoShift(shift, h.repos),
	}, nil
}

// GetShift gets a shift by ID
func (h *CashierHandler) GetShift(ctx context.Context, req *cashierpb.GetShiftRequest) (*cashierpb.ShiftResponse, error) {
	shift, err := h.service.GetShift(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &cashierpb.ShiftResponse{
		Shift: toProtoShift(shift, h.repos),
	}, nil
}

// ListShifts lists shifts with filters
func (h *CashierHandler) ListShifts(ctx context.Context, req *cashierpb.ListShiftsRequest) (*cashierpb.ListShiftsResponse, error) {
	result, err := h.service.ListShifts(ctx, req.UserId, req.StoreId, req.Status, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	shifts := make([]*cashierpb.CashierShift, 0, len(result.Shifts))
	for _, s := range result.Shifts {
		shifts = append(shifts, toProtoShift(&s, h.repos))
	}

	return &cashierpb.ListShiftsResponse{
		Shifts:     shifts,
		TotalCount: result.Pagination.TotalCount,
		Page:       result.Pagination.Page,
		PerPage:    result.Pagination.PerPage,
	}, nil
}

// GetShiftSummary gets shift summary with sales and cash info
func (h *CashierHandler) GetShiftSummary(ctx context.Context, req *cashierpb.GetShiftSummaryRequest) (*cashierpb.ShiftSummaryResponse, error) {
	detail, err := h.service.GetShiftSummary(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	// Convert sales
	sales := make([]*cashierpb.SaleSummary, 0, len(detail.Sales))
	for _, s := range detail.Sales {
		sales = append(sales, &cashierpb.SaleSummary{
			Id:            s.ID,
			InvoiceNumber: s.InvoiceNumber,
			GrandTotal:    s.GrandTotal,
			PaymentMethod: s.PaymentMethod,
			CreatedAt:     s.CreatedAt,
		})
	}

	return &cashierpb.ShiftSummaryResponse{
		ShiftId:            detail.Shift.ID.String(),
		CashierName:        detail.UserName,
		StoreName:          detail.StoreName,
		OpeningCash:        detail.Shift.OpeningCash,
		CashSales:          detail.Shift.CashSales,
		CashRefund:         detail.Shift.CashRefund,
		ExpectedCash:       detail.Shift.ExpectedCash,
		ActualCash:         detail.Shift.ActualCash,
		CashDifference:     detail.Shift.CashDifference,
		TotalTransactions:  detail.TotalTransactions,
		TotalSalesAmount:   detail.TotalSalesAmount,
		Status:             detail.Shift.Status,
		OpenedAt:           detail.Shift.OpenedAt.Format("2006-01-02T15:04:05Z07:00"),
		Sales:              sales,
	}, nil
}

// toProtoShift converts a GORM CashierShift to proto CashierShift
func toProtoShift(s *gormdb.CashierShift, repos *repository.Repositories) *cashierpb.CashierShift {
	proto := &cashierpb.CashierShift{
		Id:             s.ID.String(),
		UserId:         s.UserID.String(),
		StoreId:        s.StoreID.String(),
		OpeningCash:    s.OpeningCash,
		ClosingCash:    s.ClosingCash,
		CashSales:      s.CashSales,
		CashRefund:     s.CashRefund,
		ExpectedCash:   s.ExpectedCash,
		ActualCash:     s.ActualCash,
		CashDifference: s.CashDifference,
		Status:         s.Status,
		OpenedAt:       s.OpenedAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt:      s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if s.ClosedAt != nil {
		proto.ClosedAt = s.ClosedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	// Load user name
	if repos != nil {
		var user gormdb.User
		if err := repos.DB().Where("id = ?", s.UserID).First(&user).Error; err == nil {
			proto.UserName = user.FullName
		}

		// Load store name
		var store gormdb.Store
		if err := repos.DB().Where("id = ?", s.StoreID).First(&store).Error; err == nil {
			proto.StoreName = store.Name
		}
	}

	return proto
}
