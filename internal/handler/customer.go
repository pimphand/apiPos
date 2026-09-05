package handler

import (
	"context"

	customerv1 "apiPos/gen/customer/v1"
	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/customer"
	"apiPos/internal/response"
)

// CustomerHandler implements the CustomerService gRPC server
type CustomerHandler struct {
	customerv1.UnimplementedCustomerServiceServer
	service *customer.Service
	logs    *ActivityLogger
}

// NewCustomerHandler creates a new customer Handler
func NewCustomerHandler(service *customer.Service, logs *ActivityLogger) *CustomerHandler {
	return &CustomerHandler{service: service, logs: logs}
}

// CreateCustomer creates a new customer
func (h *CustomerHandler) CreateCustomer(ctx context.Context, req *customerv1.CreateCustomerRequest) (*customerv1.CustomerResponse, error) {
	c, err := h.service.CreateCustomer(ctx, customer.CreateCustomerRequest{
		Code:    req.Code,
		Name:    req.Name,
		Email:   req.Email,
		Phone:   req.Phone,
		Address: req.Address,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "CREATE", "customers", c.ID.String(), `{"name":"`+c.Name+`"}`)

	return &customerv1.CustomerResponse{
		Customer: toProtoCustomer(c),
	}, nil
}

// GetCustomer gets a customer by ID
func (h *CustomerHandler) GetCustomer(ctx context.Context, req *customerv1.GetCustomerRequest) (*customerv1.CustomerResponse, error) {
	c, err := h.service.GetCustomer(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	return &customerv1.CustomerResponse{
		Customer: toProtoCustomer(c),
	}, nil
}

// ListCustomers lists customers with filters
func (h *CustomerHandler) ListCustomers(ctx context.Context, req *customerv1.ListCustomersRequest) (*customerv1.ListCustomersResponse, error) {
	result, err := h.service.ListCustomers(ctx, req.IsActive, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	customers := make([]*customerv1.Customer, 0, len(result.Customers))
	for _, c := range result.Customers {
		customers = append(customers, toProtoCustomer(&c))
	}

	return &customerv1.ListCustomersResponse{
		Customers:   customers,
		TotalCount:  result.Pagination.TotalCount,
		Page:        result.Pagination.Page,
		PerPage:     result.Pagination.PerPage,
	}, nil
}

// SearchCustomers searches customers
func (h *CustomerHandler) SearchCustomers(ctx context.Context, req *customerv1.SearchCustomersRequest) (*customerv1.ListCustomersResponse, error) {
	result, err := h.service.SearchCustomers(ctx, req.Query, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	customers := make([]*customerv1.Customer, 0, len(result.Customers))
	for _, c := range result.Customers {
		customers = append(customers, toProtoCustomer(&c))
	}

	return &customerv1.ListCustomersResponse{
		Customers:   customers,
		TotalCount:  result.Pagination.TotalCount,
		Page:        result.Pagination.Page,
		PerPage:     result.Pagination.PerPage,
	}, nil
}

// UpdateCustomer updates a customer
func (h *CustomerHandler) UpdateCustomer(ctx context.Context, req *customerv1.UpdateCustomerRequest) (*customerv1.CustomerResponse, error) {
	c, err := h.service.UpdateCustomer(ctx, customer.UpdateCustomerRequest{
		ID:       req.Id,
		Code:     req.Code,
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		Address:  req.Address,
		IsActive: req.IsActive,
	})
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "UPDATE", "customers", c.ID.String(), `{"name":"`+c.Name+`"}`)

	return &customerv1.CustomerResponse{
		Customer: toProtoCustomer(c),
	}, nil
}

// DeleteCustomer deletes a customer
func (h *CustomerHandler) DeleteCustomer(ctx context.Context, req *customerv1.DeleteCustomerRequest) (*customerv1.DeleteCustomerResponse, error) {
	var meta interface{}
	if c, gerr := h.service.GetCustomer(ctx, req.Id); gerr == nil {
		meta = c
	}

	err := h.service.DeleteCustomer(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	if meta != nil {
		h.logs.LogJSON(ctx, "DELETE", "customers", req.Id, meta)
	} else {
		h.logs.Log(ctx, "DELETE", "customers", req.Id, `{}`)
	}

	return &customerv1.DeleteCustomerResponse{
		Success: true,
		Message: response.MsgCustomerDeleted,
	}, nil
}

// DeactivateCustomer deactivates a customer
func (h *CustomerHandler) DeactivateCustomer(ctx context.Context, req *customerv1.DeactivateCustomerRequest) (*customerv1.DeleteCustomerResponse, error) {
	err := h.service.DeactivateCustomer(ctx, req.Id)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	h.logs.Log(ctx, "UPDATE", "customers", req.Id, `{"is_active":false}`)

	return &customerv1.DeleteCustomerResponse{
		Success: true,
		Message: "Customer deactivated successfully",
	}, nil
}

// GetPurchaseHistory gets a customer's purchase history
func (h *CustomerHandler) GetPurchaseHistory(ctx context.Context, req *customerv1.GetPurchaseHistoryRequest) (*customerv1.PurchaseHistoryResponse, error) {
	sales, pagination, err := h.service.GetPurchaseHistory(ctx, req.Id, req.Page, req.PerPage)
	if err != nil {
		return nil, apperror.MapError(err).GRPCStatus().Err()
	}

	protoSales := make([]*customerv1.Sale, 0, len(sales))
	for _, s := range sales {
		protoSales = append(protoSales, toProtoSaleForCustomer(&s))
	}

	return &customerv1.PurchaseHistoryResponse{
		Purchases:   protoSales,
		TotalCount:  pagination.TotalCount,
		Page:        pagination.Page,
		PerPage:     pagination.PerPage,
	}, nil
}

// toProtoCustomer converts a GORM Customer to proto Customer
func toProtoCustomer(c *gormdb.Customer) *customerv1.Customer {
	return &customerv1.Customer{
		Id:        c.ID.String(),
		Code:      c.Code,
		Name:      c.Name,
		Email:     c.Email,
		Phone:     c.Phone,
		Address:   c.Address,
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// toProtoSaleForCustomer converts a GORM Sale to proto Sale (for customer purchase history)
func toProtoSaleForCustomer(s *gormdb.Sale) *customerv1.Sale {
	return &customerv1.Sale{
		Id:            s.ID.String(),
		InvoiceNumber: s.InvoiceNumber,
		StoreId:       s.StoreID.String(),
		CashierId:     s.CashierID.String(),
		CustomerId:    customerIDStr(s.CustomerID),
		Subtotal:      s.Subtotal,
		Discount:      s.Discount,
		Tax:           s.Tax,
		GrandTotal:    s.GrandTotal,
		Status:        s.Status,
		CreatedAt:     s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
