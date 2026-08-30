// Base types
export interface PaginationParams {
  page: number;
  per_page: number;
}

export interface PaginationResponse {
  total_count: number;
  page: number;
  per_page: number;
}

// Auth types
export interface User {
  id: string;
  username: string;
  email: string;
  full_name: string;
  role_id: string;
  role_name?: string;
  store_id: string;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_at: number;
  user: User;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
  full_name: string;
  store_name: string;
}

export interface RefreshTokenRequest {
  refresh_token: string;
}

export interface RefreshTokenResponse {
  access_token: string;
  refresh_token: string;
  expires_at: number;
}

export interface LogoutResponse {
  success: boolean;
  message: string;
}

// Category types
export interface Category {
  id: string;
  name: string;
  description: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateCategoryRequest {
  name: string;
  description?: string;
}

export interface UpdateCategoryRequest {
  name?: string;
  description?: string;
  is_active?: boolean;
}

export interface ListCategoriesResponse extends PaginationResponse {
  categories: Category[];
}

// Product types
export interface Product {
  id: string;
  name: string;
  description: string;
  category_id: string;
  category_name?: string;
  cost_price: number;
  selling_price: number;
  stock: number;
  minimum_stock: number;
  is_active: boolean;
  sku_count?: number;
  skus?: ProductSKU[];
  created_at: string;
  updated_at: string;
}

export interface ProductSKU {
  id: string;
  product_id: string;
  sku: string;
  barcode: string;
  variant: string;
  color: string;
  size: string;
  unit: string;
  cost_price: number;
  selling_price: number;
  stock: number;
  minimum_stock: number;
  weight: number;
  volume: number;
  expiry_date?: string;
  batch_number: string;
  location: string;
  image_url: string;
  is_active: boolean;
  metadata?: Record<string, string>;
  created_at: string;
  updated_at: string;
}

export interface CreateProductRequest {
  name: string;
  description?: string;
  category_id: string;
  cost_price: number;
  selling_price: number;
  stock: number;
  minimum_stock?: number;
  sku?: string;
  barcode?: string;
  variant?: string;
  color?: string;
  size?: string;
  unit?: string;
  weight?: number;
  volume?: number;
  batch_number?: string;
  location?: string;
}

export interface UpdateProductRequest {
  name?: string;
  description?: string;
  category_id?: string;
  cost_price?: number;
  selling_price?: number;
  minimum_stock?: number;
  is_active?: boolean;
  sku_updates?: ProductSKUUpdate[];
  sku_ids_to_delete?: string[];
}

export interface ProductSKUUpdate {
  id?: string;
  sku: string;
  barcode?: string;
  variant?: string;
  color?: string;
  size?: string;
  unit?: string;
  cost_price?: number;
  selling_price?: number;
  stock?: number;
  minimum_stock?: number;
  weight?: number;
  volume?: number;
  expiry_date?: string;
  batch_number?: string;
  location?: string;
  image_url?: string;
  is_active?: boolean;
  metadata?: Record<string, string>;
}

export interface ListProductsResponse extends PaginationResponse {
  products: Product[];
}

// Customer types
export interface Customer {
  id: string;
  code: string;
  name: string;
  email: string;
  phone: string;
  address: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateCustomerRequest {
  code: string;
  name: string;
  email?: string;
  phone?: string;
  address?: string;
}

export interface UpdateCustomerRequest {
  code?: string;
  name?: string;
  email?: string;
  phone?: string;
  address?: string;
  is_active?: boolean;
}

export interface ListCustomersResponse extends PaginationResponse {
  customers: Customer[];
}

// Sale types
export interface Sale {
  id: string;
  invoice_number: string;
  store_id: string;
  store_name?: string;
  cashier_id: string;
  cashier_name?: string;
  customer_id: string;
  customer_name?: string;
  subtotal: number;
  discount: number;
  tax: number;
  grand_total: number;
  status: 'PENDING' | 'PAID' | 'CANCELLED' | 'REFUNDED';
  item_count?: number;
  created_at: string;
  updated_at: string;
}

export interface SaleItem {
  id: string;
  sale_id: string;
  product_id: string;
  product_name?: string;
  sku_id?: string;
  sku_code?: string;
  quantity: number;
  unit_price: number;
  discount: number;
  subtotal: number;
  created_at: string;
}

export interface CreateSaleRequest {
  store_id: string;
  customer_id?: string;
  items: SaleItemInput[];
  discount?: number;
  tax_rate?: number;
  payment_method: string;
  amount_paid: number;
  payment_reference?: string;
}

export interface SaleItemInput {
  product_id: string;
  sku_id?: string;
  quantity: number;
  unit_price: number;
  discount?: number;
}

export interface SaleDetailResponse {
  sale: Sale;
  items: SaleItem[];
  payments: Payment[];
  amount_paid: number;
  change: number;
}

export interface ListSalesResponse extends PaginationResponse {
  sales: Sale[];
}

// Payment types
export interface Payment {
  id: string;
  sale_id: string;
  payment_method: 'CASH' | 'CARD' | 'BANK_TRANSFER' | 'EWALLET' | 'OTHER';
  amount: number;
  reference: string;
  paid_at: string;
  created_at: string;
}

export interface CreatePaymentRequest {
  sale_id: string;
  payment_method: string;
  amount: number;
  reference?: string;
}

export interface PaymentSummary {
  sale_id: string;
  grand_total: number;
  total_paid: number;
  remaining: number;
  change: number;
  status: 'PENDING' | 'PARTIAL' | 'PAID';
  payments: Payment[];
}

export interface ListPaymentsResponse extends PaginationResponse {
  payments: Payment[];
}

// Cashier Shift types
export interface CashierShift {
  id: string;
  user_id: string;
  user_name?: string;
  store_id: string;
  store_name?: string;
  opening_cash: number;
  closing_cash: number;
  cash_sales: number;
  cash_refund: number;
  expected_cash: number;
  actual_cash: number;
  cash_difference: number;
  status: 'OPEN' | 'CLOSED';
  opened_at: string;
  closed_at?: string;
  created_at: string;
}

export interface ShiftSummary {
  shift_id: string;
  cashier_name: string;
  store_name: string;
  opening_cash: number;
  cash_sales: number;
  cash_refund: number;
  expected_cash: number;
  actual_cash: number;
  cash_difference: number;
  total_transactions: number;
  total_sales_amount: number;
  status: string;
  opened_at: string;
  closed_at?: string;
  sales: ShiftSale[];
}

export interface ShiftSale {
  id: string;
  invoice_number: string;
  grand_total: number;
  payment_method: string;
  created_at: string;
}

export interface ListShiftsResponse extends PaginationResponse {
  shifts: CashierShift[];
}

// Inventory types
export interface InventoryTransaction {
  id: string;
  product_id: string;
  product_name?: string;
  store_id: string;
  store_name?: string;
  type: 'PURCHASE' | 'SALE' | 'RETURN' | 'ADJUSTMENT' | 'DAMAGE' | 'TRANSFER';
  quantity: number;
  reference_type: string;
  reference_id: string;
  before_stock: number;
  after_stock: number;
  description: string;
  created_by: string;
  creator_name?: string;
  created_at: string;
}

export interface ListTransactionsResponse extends PaginationResponse {
  transactions: InventoryTransaction[];
}

// Report types
export interface DailySalesReport {
  date: string;
  total_sales: number;
  total_amount: number;
}

export interface DailySalesReportResponse {
  reports: DailySalesReport[];
  total_sales: number;
  total_amount: number;
}

export interface SaleReport {
  id: string;
  invoice_number: string;
  store_name: string;
  cashier_name: string;
  customer_name: string;
  subtotal: number;
  discount: number;
  tax: number;
  grand_total: number;
  status: string;
  created_at: string;
}

export interface CashierSalesReport {
  cashier_id: string;
  cashier_name: string;
  total_transactions: number;
  total_amount: number;
}

export interface ProductSalesReport {
  product_id: string;
  product_name: string;
  quantity_sold: number;
  total_amount: number;
}

export interface CategorySalesReport {
  category_id: string;
  category_name: string;
  product_count: number;
  quantity_sold: number;
  total_amount: number;
}

export interface PaymentMethodReport {
  payment_method: string;
  count: number;
  total_amount: number;
}

export interface InventoryReport {
  category_name: string;
  product_count: number;
  total_stock: number;
  total_value: number;
}

export interface LowStockReport {
  product_id: string;
  product_name: string;
  category_name: string;
  current_stock: number;
  minimum_stock: number;
  deficit: number;
}

// Role types
export interface Role {
  id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface Permission {
  id: string;
  name: string;
  resource: string;
  action: string;
  description: string;
  created_at: string;
  updated_at: string;
}

// Activity Log types
export interface ActivityLog {
  id: string;
  user_id: string;
  username: string;
  action: string;
  resource: string;
  resource_id: string;
  description: string;
  ip_address: string;
  user_agent: string;
  metadata: string;
  created_at: string;
}

// Error types
export interface ApiError {
  code: string;
  message: string;
  status?: number;
}
