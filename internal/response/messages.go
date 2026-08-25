package response

// Success messages
const (
	// Auth
	MsgLoginSuccess       = "Login successful"
	MsgRegisterSuccess    = "Registration successful"
	MsgLogoutSuccess      = "Logout successful"
	MsgTokenRefreshed     = "Token refreshed successfully"

	// User
	MsgUserCreated        = "User created successfully"
	MsgUserUpdated        = "User updated successfully"
	MsgUserDeleted        = "User deleted successfully"
	MsgUserRoleUpdated    = "User role updated successfully"
	MsgUserDeactivated    = "User deactivated successfully"

	// Product
	MsgProductCreated     = "Product created successfully"
	MsgProductUpdated     = "Product updated successfully"
	MsgProductDeleted     = "Product deleted successfully"
	MsgProductActivated   = "Product activated successfully"
	MsgProductDeactivated = "Product deactivated successfully"

	// Category
	MsgCategoryCreated     = "Category created successfully"
	MsgCategoryUpdated     = "Category updated successfully"
	MsgCategoryDeleted     = "Category deleted successfully"
	MsgCategoryDeactivated = "Category deactivated successfully"

	// Customer
	MsgCustomerCreated     = "Customer created successfully"
	MsgCustomerUpdated     = "Customer updated successfully"
	MsgCustomerDeleted     = "Customer deleted successfully"
	MsgCustomerDeactivated = "Customer deactivated successfully"

	// Sale
	MsgSaleCreated    = "Sale created successfully"
	MsgSaleUpdated    = "Sale updated successfully"
	MsgSaleDeleted    = "Sale deleted successfully"
	MsgSaleCancelled  = "Sale cancelled successfully"

	// Payment
	MsgPaymentCreated = "Payment recorded successfully"
	MsgPaymentDeleted = "Payment deleted successfully"

	// Cashier
	MsgShiftOpened    = "Shift opened successfully"
	MsgShiftClosed    = "Shift closed successfully"

	// General
	MsgOperationSuccess = "Operation completed successfully"
)

// Error messages (user-facing, safe to expose)
const (
	// Auth
	ErrInvalidCredentials = "Invalid username or password"
	ErrInvalidToken       = "Invalid token"
	ErrTokenExpired       = "Token expired"
	ErrUnauthorized       = "Unauthorized access"
	ErrForbidden          = "Insufficient permissions"

	// User
	ErrUserNotFound      = "User not found"
	ErrUserAlreadyExists = "Username already exists"
	ErrEmailAlreadyExists = "Email already exists"

	// Product
	ErrProductNotFound      = "Product not found"
	ErrProductAlreadyExists = "Product already exists"
	ErrInsufficientStock    = "Insufficient stock"
	ErrInvalidSKU           = "Invalid SKU"

	// Category
	ErrCategoryNotFound      = "Category not found"
	ErrCategoryAlreadyExists = "Category already exists"
	ErrCategoryHasProducts   = "Cannot delete category with products"

	// Customer
	ErrCustomerNotFound      = "Customer not found"
	ErrCustomerAlreadyExists = "Customer already exists"

	// Sale
	ErrSaleNotFound          = "Sale not found"
	ErrSaleAlreadyExists     = "Invoice number already exists"
	ErrInvalidSaleStatus     = "Invalid sale status"
	ErrPaymentInsufficient   = "Payment amount insufficient"

	// Cashier
	ErrShiftNotFound    = "Shift not found"
	ErrShiftAlreadyOpen = "You already have an open shift"
	ErrShiftNotOpen     = "No open shift found"

	// General
	ErrInternal       = "Internal server error"
	ErrInvalidRequest = "Invalid request data"
	ErrDatabaseError  = "Database error occurred"
)
