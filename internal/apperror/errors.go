package apperror

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Sentinel errors for business logic
var (
	// Authentication errors
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")

	// User errors
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")

	// Product errors
	ErrProductNotFound      = errors.New("product not found")
	ErrProductAlreadyExists = errors.New("product already exists")
	ErrInsufficientStock    = errors.New("insufficient stock")

	// Category errors
	ErrCategoryNotFound      = errors.New("category not found")
	ErrCategoryAlreadyExists = errors.New("category already exists")

	// Customer errors
	ErrCustomerNotFound      = errors.New("customer not found")
	ErrCustomerAlreadyExists = errors.New("customer already exists")

	// Sale errors
	ErrSaleNotFound        = errors.New("sale not found")
	ErrSaleAlreadyExists   = errors.New("sale already exists")
	ErrInvalidSaleStatus   = errors.New("invalid sale status")
	ErrPaymentInsufficient = errors.New("payment amount insufficient")

	// Cashier errors
	ErrShiftNotFound    = errors.New("shift not found")
	ErrShiftAlreadyOpen = errors.New("shift already open")
	ErrShiftNotOpen     = errors.New("shift not open")

	// Reference errors
	ErrInvalidReference = errors.New("invalid reference")

	// General errors
	ErrNotFound           = errors.New("resource not found")
	ErrAlreadyExists      = errors.New("resource already exists")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInternal           = errors.New("internal server error")
	ErrFailedPrecondition = errors.New("failed precondition")
)

// AppError represents a structured application error
type AppError struct {
	Code    codes.Code
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// GRPCStatus converts the error to a gRPC status error
func (e *AppError) GRPCStatus() *status.Status {
	return status.New(e.Code, e.Message)
}

// New creates a new AppError with an explicit gRPC code
func New(code codes.Code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Wrap wraps a cause error with a context message, inferring the gRPC code from the sentinel error.
// If the cause is a PostgreSQL error, it uses the PG-specific message and code.
func Wrap(sentinelErr error, message string, cause error) *AppError {
	// First, check if the cause itself is a PostgreSQL error with a better message
	if pgMsg, pgCode := describePGError(cause); pgMsg != "" {
		return &AppError{
			Code:    pgCode,
			Message: pgMsg,
			Err:     cause,
		}
	}

	// Fall back to the sentinel error's code
	appErr := MapError(sentinelErr)
	if appErr != nil {
		return &AppError{
			Code:    appErr.Code,
			Message: message,
			Err:     cause,
		}
	}
	return &AppError{
		Code:    codes.Internal,
		Message: message,
		Err:     cause,
	}
}

// isPgUniqueViolation checks if error is a PostgreSQL unique constraint violation (23505)
func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// isPgForeignKeyViolation checks if error is a PostgreSQL foreign key violation (23503)
func isPgForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	return false
}

// getConstraintName extracts the constraint name from a PostgreSQL error message
func getConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

// describePGError converts a raw PostgreSQL error into a user-friendly message
func describePGError(err error) (string, codes.Code) {
	if isPgUniqueViolation(err) {
		constraint := getConstraintName(err)
		switch {
		case strings.Contains(constraint, "users_username"):
			return "Username already exists", codes.AlreadyExists
		case strings.Contains(constraint, "users_email"):
			return "Email already exists", codes.AlreadyExists
		case strings.Contains(constraint, "products_sku"):
			return "Product SKU already exists", codes.AlreadyExists
		case strings.Contains(constraint, "products_barcode"):
			return "Barcode already exists", codes.AlreadyExists
		case strings.Contains(constraint, "categories_name"):
			return "Category name already exists", codes.AlreadyExists
		case strings.Contains(constraint, "roles_name"):
			return "Role name already exists", codes.AlreadyExists
		case strings.Contains(constraint, "permissions_name"):
			return "Permission name already exists", codes.AlreadyExists
		case strings.Contains(constraint, "stores_name"):
			return "Store name already exists", codes.AlreadyExists
		case strings.Contains(constraint, "sales_invoice"):
			return "Invoice number already exists", codes.AlreadyExists
		default:
			return "Record already exists", codes.AlreadyExists
		}
	}

	if isPgForeignKeyViolation(err) {
		constraint := getConstraintName(err)
		// Also check the error message for table hints
		errMsg := err.Error()
		switch {
		case strings.Contains(constraint, "role") || strings.Contains(errMsg, "roles"):
			return "Invalid role reference", codes.InvalidArgument
		case strings.Contains(constraint, "store") || strings.Contains(errMsg, "stores"):
			return "Invalid store reference", codes.InvalidArgument
		case strings.Contains(constraint, "category") || strings.Contains(errMsg, "categories"):
			return "Invalid category reference", codes.InvalidArgument
		case strings.Contains(constraint, "product") || strings.Contains(errMsg, "products"):
			return "Invalid product reference", codes.InvalidArgument
		case strings.Contains(constraint, "customer") || strings.Contains(errMsg, "customers"):
			return "Invalid customer reference", codes.InvalidArgument
		case strings.Contains(constraint, "cashier") || strings.Contains(errMsg, "users"):
			return "Invalid user reference", codes.InvalidArgument
		default:
			return "Invalid reference to related record", codes.InvalidArgument
		}
	}

	return "", 0
}

// MapError maps an error to a proper AppError with gRPC status code
func MapError(err error) *AppError {
	if err == nil {
		return nil
	}

	// Check if it's already an AppError
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Handle PostgreSQL-specific errors
	if msg, code := describePGError(err); msg != "" {
		return New(code, msg, err)
	}

	// Map sentinel errors to gRPC status codes
	switch {
	// Auth errors
	case errors.Is(err, ErrInvalidCredentials):
		return New(codes.Unauthenticated, "Invalid username or password", err)
	case errors.Is(err, ErrInvalidToken):
		return New(codes.Unauthenticated, "Invalid token", err)
	case errors.Is(err, ErrTokenExpired):
		return New(codes.Unauthenticated, "Token expired", err)
	case errors.Is(err, ErrUnauthorized):
		return New(codes.PermissionDenied, "Unauthorized access", err)
	case errors.Is(err, ErrForbidden):
		return New(codes.PermissionDenied, "Insufficient permissions", err)

	// User errors
	case errors.Is(err, ErrUserNotFound):
		return New(codes.NotFound, "User not found", err)
	case errors.Is(err, ErrUserAlreadyExists):
		return New(codes.AlreadyExists, "User already exists", err)

	// Product errors
	case errors.Is(err, ErrProductNotFound):
		return New(codes.NotFound, "Product not found", err)
	case errors.Is(err, ErrProductAlreadyExists):
		return New(codes.AlreadyExists, "Product already exists", err)
	case errors.Is(err, ErrInsufficientStock):
		return New(codes.FailedPrecondition, "Insufficient stock", err)

	// Category errors
	case errors.Is(err, ErrCategoryNotFound):
		return New(codes.NotFound, "Category not found", err)
	case errors.Is(err, ErrCategoryAlreadyExists):
		return New(codes.AlreadyExists, "Category already exists", err)

	// Customer errors
	case errors.Is(err, ErrCustomerNotFound):
		return New(codes.NotFound, "Customer not found", err)
	case errors.Is(err, ErrCustomerAlreadyExists):
		return New(codes.AlreadyExists, "Customer already exists", err)

	// Sale errors
	case errors.Is(err, ErrSaleNotFound):
		return New(codes.NotFound, "Sale not found", err)
	case errors.Is(err, ErrSaleAlreadyExists):
		return New(codes.AlreadyExists, "Sale already exists", err)
	case errors.Is(err, ErrInvalidSaleStatus):
		return New(codes.InvalidArgument, "Invalid sale status", err)
	case errors.Is(err, ErrPaymentInsufficient):
		return New(codes.FailedPrecondition, "Payment amount insufficient", err)

	// Cashier errors
	case errors.Is(err, ErrShiftNotFound):
		return New(codes.NotFound, "Shift not found", err)
	case errors.Is(err, ErrShiftAlreadyOpen):
		return New(codes.AlreadyExists, "Shift already open", err)
	case errors.Is(err, ErrShiftNotOpen):
		return New(codes.FailedPrecondition, "Shift not open", err)

	// Reference errors
	case errors.Is(err, ErrInvalidReference):
		return New(codes.InvalidArgument, "Invalid reference", err)

	// General errors
	case errors.Is(err, ErrNotFound):
		return New(codes.NotFound, "Resource not found", err)
	case errors.Is(err, ErrAlreadyExists):
		return New(codes.AlreadyExists, "Resource already exists", err)
	case errors.Is(err, ErrInvalidRequest):
		return New(codes.InvalidArgument, "Invalid request", err)
	case errors.Is(err, ErrInternal):
		return New(codes.Internal, "Internal server error", err)
	case errors.Is(err, ErrFailedPrecondition):
		return New(codes.FailedPrecondition, "Failed precondition", err)

	default:
		return New(codes.Internal, "Internal server error", err)
	}
}
