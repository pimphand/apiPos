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
  /** Actual permissions granted to this user's role from the backend. */
  permissions?: string[];
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

// POS hold order types (docs/pos-hold-order.md)
/** One snapshot cart line at the moment the order was held. */
export interface HeldOrderItem {
  product_id: string;
  sku_id: string;
  name: string; // product name snapshot (display only)
  sku_code: string; // SKU/variant code snapshot (display only)
  unit_price: number; // selling price AT hold time (R7)
  quantity: number;
  stock: number; // stock at hold time (snapshot reference)
  original_stock: number; // same as stock — kept to match CartItem shape
}

export interface HeldOrder {
  id: string;
  store_id: string; // local tenancy scope (R11)
  user_id: string; // cashier scope (R11)
  username: string;
  label: string; // e.g. "Hold #1 · Budi" (display in list)
  items: HeldOrderItem[];
  discount: number; // cart discount snapshot
  tax_rate: number; // cart tax-rate snapshot
  customer_id: string; // '' = walk-in
  customer_name?: string;
  created_at: string;
  updated_at: string;
}

/** Shape persisted in localStorage (pos.held_orders.v1). */
export interface HeldOrderStore {
  version: number;
  holds: HeldOrder[];
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
  /** Grand total per payment method, e.g. { CASH: 50000, CARD: 0 }. */
  revenue_by_method?: Record<string, number>;
  items_out_total?: number;
  items_in_total?: number;
}

export interface ShiftSale {
  id: string;
  invoice_number: string;
  grand_total: number;
  payment_method: string;
  created_at: string;
}

/** One row of the insert-only per-item shift ledger (goods OUT/IN). */
export interface ShiftItem {
  id: string;
  shift_id: string;
  sale_id?: string;
  direction: 'OUT' | 'IN';
  item_type: 'PRODUCT' | 'SKU';
  product_id: string;
  product_sku_id?: string;
  product_name: string;
  sku_code?: string;
  variant_label?: string;
  quantity: number;
  unit_price: number;
  discount_amount: number;
  line_amount: number;
  payment_method?: string;
  reference_no?: string;
  created_at: string;
}

export interface ListShiftItemsResponse extends PaginationResponse {
  items: ShiftItem[];
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
  /** Tenant store the activity belongs to (empty for system-level actions). */
  store_id?: string;
  store_name?: string;
}

// Store types (superadmin)
export interface Store {
  id: string;
  name: string;
  address: string;
  phone: string;
  is_active: string;   // "true" | "false" (string from proto)
  is_verified: string; // "true" | "false" (string from proto)
  created_at: string;
  updated_at: string;
}

// Per-store settings (docs/pos-store-settings.md)
export interface StoreSettings {
  store_id: string;
  tax_rate: number; // default tax % for new POS carts
  receipt_header: string;
  receipt_footer: string;
  show_store_name: boolean;
  show_store_contact: boolean;
  show_cashier: boolean;
  show_customer: boolean;
  show_item_sku: boolean;
  show_item_barcode: boolean;
  show_discount: boolean;
  show_tax: boolean;
  show_payment: boolean;
  show_change: boolean;
  updated_by: string;
  updated_at: string;
}

// Error types
export interface ApiError {
  code: string;
  message: string;
  status?: number;
}
