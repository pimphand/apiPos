package handler

import (
	"context"
	"log"

	"github.com/google/uuid"

	inventoryv1 "apiPos/gen/inventory/v1"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/inventory"
)

// InventoryHandler implements the InventoryService gRPC server
type InventoryHandler struct {
	inventoryv1.UnimplementedInventoryServiceServer
	service *inventory.Service
}

// NewInventoryHandler creates a new inventory Handler
func NewInventoryHandler(service *inventory.Service) *InventoryHandler {
	return &InventoryHandler{service: service}
}

// CreateTransaction creates a new inventory transaction
func (h *InventoryHandler) CreateTransaction(ctx context.Context, req *inventoryv1.CreateTransactionRequest) (*inventoryv1.TransactionResponse, error) {
	// Get user from context (set by auth middleware)
	userID := getUserIDFromContext(ctx)

	txn, err := h.service.CreateTransaction(ctx, inventory.CreateTransactionRequest{
		ProductID:     req.ProductId,
		StoreID:       req.StoreId,
		Type:          req.Type,
		Quantity:      req.Quantity,
		ReferenceType: req.ReferenceType,
		ReferenceID:   req.ReferenceId,
		Description:   req.Description,
		CreatedBy:     userID,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &inventoryv1.TransactionResponse{
		Transaction: toProtoTransaction(txn, h.service),
	}, nil
}

// GetTransaction gets an inventory transaction by ID
func (h *InventoryHandler) GetTransaction(ctx context.Context, req *inventoryv1.GetTransactionRequest) (*inventoryv1.TransactionResponse, error) {
	txn, err := h.service.GetTransaction(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &inventoryv1.TransactionResponse{
		Transaction: toProtoTransaction(txn, h.service),
	}, nil
}

// ListTransactions lists inventory transactions with filters
func (h *InventoryHandler) ListTransactions(ctx context.Context, req *inventoryv1.ListTransactionsRequest) (*inventoryv1.ListTransactionsResponse, error) {
	result, err := h.service.ListTransactions(ctx, req.ProductId, req.StoreId, req.Type, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoTxns := make([]*inventoryv1.InventoryTransaction, 0, len(result.Transactions))
	for _, txn := range result.Transactions {
		protoTxns = append(protoTxns, toProtoTransaction(&txn, h.service))
	}

	return &inventoryv1.ListTransactionsResponse{
		Transactions: protoTxns,
		TotalCount:   result.Pagination.TotalCount,
		Page:         result.Pagination.Page,
		PerPage:      result.Pagination.PerPage,
	}, nil
}

// AdjustStock adjusts stock for a product
func (h *InventoryHandler) AdjustStock(ctx context.Context, req *inventoryv1.AdjustStockRequest) (*inventoryv1.TransactionResponse, error) {
	userID := getUserIDFromContext(ctx)

	txn, err := h.service.AdjustStock(ctx, inventory.AdjustStockRequest{
		ProductID:   req.ProductId,
		StoreID:     req.StoreId,
		Direction:   req.Direction,
		Quantity:    req.Quantity,
		Reason:      req.Reason,
		Description: req.Description,
		CreatedBy:   userID,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &inventoryv1.TransactionResponse{
		Transaction: toProtoTransaction(txn, h.service),
	}, nil
}

// GetStockSummary gets stock summary for a product
func (h *InventoryHandler) GetStockSummary(ctx context.Context, req *inventoryv1.GetStockSummaryRequest) (*inventoryv1.StockSummaryResponse, error) {
	summary, err := h.service.GetStockSummary(ctx, req.ProductId)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	skus := make([]*inventoryv1.SKUStock, 0, len(summary.SKUs))
	for _, sku := range summary.SKUs {
		skus = append(skus, &inventoryv1.SKUStock{
			SkuId:        sku.SKUID.String(),
			SkuCode:      sku.SKUCode,
			Variant:      sku.Variant,
			Barcode:      sku.Barcode,
			Stock:        sku.Stock,
			MinimumStock: sku.MinimumStock,
			IsLowStock:   sku.IsLowStock,
		})
	}

	return &inventoryv1.StockSummaryResponse{
		ProductId:   summary.ProductID.String(),
		ProductName: summary.ProductName,
		TotalStock:  summary.TotalStock,
		Skus:        skus,
	}, nil
}

// GetLowStockAlert gets products with stock below minimum
func (h *InventoryHandler) GetLowStockAlert(ctx context.Context, req *inventoryv1.LowStockAlertRequest) (*inventoryv1.LowStockAlertResponse, error) {
	result, err := h.service.GetLowStockAlert(ctx, req.StoreId, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	products := make([]*inventoryv1.LowStockProduct, 0, len(result.Products))
	for _, p := range result.Products {
		products = append(products, &inventoryv1.LowStockProduct{
			ProductId:    p.ProductID.String(),
			ProductName:  p.ProductName,
			CategoryName: p.CategoryName,
			TotalStock:   p.TotalStock,
			MinimumStock: p.MinimumStock,
			Deficit:      p.Deficit,
		})
	}

	return &inventoryv1.LowStockAlertResponse{
		Products:    products,
		TotalCount:  result.Pagination.TotalCount,
		Page:        result.Pagination.Page,
		PerPage:     result.Pagination.PerPage,
	}, nil
}

// toProtoTransaction converts a GORM InventoryTransaction to proto
func toProtoTransaction(txn *gormdb.InventoryTransaction, svc *inventory.Service) *inventoryv1.InventoryTransaction {
	protoTxn := &inventoryv1.InventoryTransaction{
		Id:            txn.ID.String(),
		ProductId:     txn.ProductID.String(),
		StoreId:       txn.StoreID.String(),
		Type:          txn.Type,
		Quantity:      txn.Quantity,
		ReferenceType: txn.ReferenceType,
		BeforeStock:   txn.BeforeStock,
		AfterStock:    txn.AfterStock,
		CreatedBy:     txn.CreatedBy.String(),
		CreatedAt:     txn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if txn.ReferenceID != uuid.Nil {
		protoTxn.ReferenceId = txn.ReferenceID.String()
	}

	return protoTxn
}

// getUserIDFromContext extracts user ID from context (set by auth middleware)
func getUserIDFromContext(ctx context.Context) string {
	// Try to get from context - auth middleware sets this
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	// Fallback - return empty string (will be validated in service)
	log.Printf("Warning: user_id not found in context")
	return ""
}
