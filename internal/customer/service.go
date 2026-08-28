package customer

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"apiPos/internal/apperror"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// Service handles customer business logic
type Service struct {
	repos *repository.Repositories
}

// NewService creates a new customer Service
func NewService(repos *repository.Repositories) *Service {
	return &Service{repos: repos}
}

// CreateCustomerRequest represents the request to create a customer
type CreateCustomerRequest struct {
	Code    string
	Name    string
	Email   string
	Phone   string
	Address string
}

// UpdateCustomerRequest represents the request to update a customer
type UpdateCustomerRequest struct {
	ID        string
	Code      string
	Name      string
	Email     string
	Phone     string
	Address   string
	IsActive  bool
}

// PaginationResult holds pagination metadata
type PaginationResult struct {
	TotalCount int64
	Page       int32
	PerPage    int32
}

// ListResult holds the result of listing customers
type ListResult struct {
	Customers []gormdb.Customer
	Pagination PaginationResult
}

// CreateCustomer creates a new customer
func (s *Service) CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*gormdb.Customer, error) {
	if req.Code == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Customer code is required", nil)
	}
	if req.Name == "" {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Customer name is required", nil)
	}

	customer := &gormdb.Customer{
		Code:    req.Code,
		Name:    req.Name,
		Email:   req.Email,
		Phone:   req.Phone,
		Address: req.Address,
		IsActive: true,
	}

	if err := s.repos.Customer.Create(ctx, customer); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to create customer", err)
	}

	return customer, nil
}

// GetCustomer gets a customer by ID
func (s *Service) GetCustomer(ctx context.Context, id string) (*gormdb.Customer, error) {
	if !isValidUUID(id) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid customer ID", nil)
	}

	customer, err := s.repos.Customer.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Customer not found", err)
	}

	return customer, nil
}

// ListCustomers lists customers with filters
func (s *Service) ListCustomers(ctx context.Context, isActive string, page, perPage int32) (*ListResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	customers, totalCount, err := s.repos.Customer.List(ctx, repository.CustomerListParams{
		IsActive: isActive,
		Page:     page,
		PerPage:  perPage,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to list customers", err)
	}

	return &ListResult{
		Customers: customers,
		Pagination: PaginationResult{
			TotalCount: totalCount,
			Page:       page,
			PerPage:    perPage,
		},
	}, nil
}

// SearchCustomers searches customers by name, code, email, or phone
func (s *Service) SearchCustomers(ctx context.Context, query string, page, perPage int32) (*ListResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	customers, totalCount, err := s.repos.Customer.Search(ctx, repository.CustomerSearchParams{
		Query:   query,
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to search customers", err)
	}

	return &ListResult{
		Customers: customers,
		Pagination: PaginationResult{
			TotalCount: totalCount,
			Page:       page,
			PerPage:    perPage,
		},
	}, nil
}

// UpdateCustomer updates a customer
func (s *Service) UpdateCustomer(ctx context.Context, req UpdateCustomerRequest) (*gormdb.Customer, error) {
	if !isValidUUID(req.ID) {
		return nil, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid customer ID", nil)
	}

	customer, err := s.repos.Customer.GetByID(ctx, req.ID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrNotFound, "Customer not found", err)
	}

	if req.Code != "" {
		customer.Code = req.Code
	}
	if req.Name != "" {
		customer.Name = req.Name
	}
	if req.Email != "" {
		customer.Email = req.Email
	}
	if req.Phone != "" {
		customer.Phone = req.Phone
	}
	if req.Address != "" {
		customer.Address = req.Address
	}
	customer.IsActive = req.IsActive

	if err := s.repos.Customer.Update(ctx, customer); err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to update customer", err)
	}

	return customer, nil
}

// DeleteCustomer deletes a customer
func (s *Service) DeleteCustomer(ctx context.Context, id string) error {
	if !isValidUUID(id) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid customer ID", nil)
	}

	_, err := s.repos.Customer.GetByID(ctx, id)
	if err != nil {
		return apperror.Wrap(apperror.ErrNotFound, "Customer not found", err)
	}

	return s.repos.Customer.Delete(ctx, id)
}

// DeactivateCustomer deactivates a customer
func (s *Service) DeactivateCustomer(ctx context.Context, id string) error {
	if !isValidUUID(id) {
		return apperror.Wrap(apperror.ErrInvalidRequest, "Invalid customer ID", nil)
	}

	_, err := s.repos.Customer.GetByID(ctx, id)
	if err != nil {
		return apperror.Wrap(apperror.ErrNotFound, "Customer not found", err)
	}

	return s.repos.Customer.Deactivate(ctx, id)
}

// GetPurchaseHistory gets a customer's purchase history
func (s *Service) GetPurchaseHistory(ctx context.Context, customerID string, page, perPage int32) ([]gormdb.Sale, PaginationResult, error) {
	if !isValidUUID(customerID) {
		return nil, PaginationResult{}, apperror.Wrap(apperror.ErrInvalidRequest, "Invalid customer ID", nil)
	}

	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	sales, totalCount, err := s.repos.Customer.GetPurchaseHistory(ctx, customerID, repository.PaginationParams{
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return nil, PaginationResult{}, apperror.Wrap(apperror.ErrInternal, "Failed to get purchase history", err)
	}

	return sales, PaginationResult{
		TotalCount: totalCount,
		Page:       page,
		PerPage:    perPage,
	}, nil
}

// isValidUUID validates UUID format
func isValidUUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}

// Ensure fmt is used
var _ = fmt.Sprintf
