package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"apiPos/internal/auth"
	"apiPos/internal/category"
	"apiPos/internal/config"
	"apiPos/internal/product"
	"apiPos/internal/repository"
)

// ProductImportHandler handles CSV/Excel product import via plain HTTP
// (multipart form upload). Registered directly on the HTTP mux because
// grpc-gateway does not support multipart file uploads.
type ProductImportHandler struct {
	productService    *product.Service
	categoryService   *category.Service
	repos             *repository.Repositories
	authService       *auth.AuthService
	config            *config.Config
}

func NewProductImportHandler(
	productService *product.Service,
	categoryService *category.Service,
	repos *repository.Repositories,
	authService *auth.AuthService,
	cfg *config.Config,
) *ProductImportHandler {
	return &ProductImportHandler{
		productService:  productService,
		categoryService: categoryService,
		repos:           repos,
		authService:     authService,
		config:          cfg,
	}
}

// ImportResult summarizes a batch import.
type ImportResult struct {
	Total    int              `json:"total"`
	Imported int              `json:"imported"`
	Failed   int              `json:"failed"`
	Errors   []ImportRowError `json:"errors"`
}

type ImportRowError struct {
	Row int    `json:"row"`
	Msg string `json:"msg"`
}

// ServeHTTP handles POST /api/v1/products/import.
func (h *ProductImportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Authenticate via JWT.
	claims, ok := auth.ClaimsFromRequest(r, h.config.JWT.Secret)
	if !ok || claims == nil {
		http.Error(w, `{"error":"unauthenticated"}`, http.StatusUnauthorized)
		return
	}
	storeID, err := uuid.Parse(claims.StoreID)
	if err != nil {
		http.Error(w, `{"error":"no store"}`, http.StatusForbidden)
		return
	}
	userID, _ := uuid.Parse(claims.UserID)

	// Attach claims to context so store-scoped services/callbacks resolve the
	// importing store (e.g. category auto-create during import).
	r = r.WithContext(auth.ContextWithClaims(r.Context(), claims))

	// Parse multipart form (max 10 MB).
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid upload: " + err.Error()})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "file field required"})
		return
	}
	defer file.Close()

	// Detect format by extension.
	name := strings.ToLower(header.Filename)
	var rows [][]string
	if strings.HasSuffix(name, ".xlsx") || strings.HasSuffix(name, ".xls") {
		rows, err = parseExcel(file)
	} else {
		rows, err = parseCSV(file)
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "parse error: " + err.Error()})
		return
	}

	result := h.processRows(r, storeID, userID, rows)
	writeJSON(w, http.StatusOK, result)
}

// ServeSampleCSV handles GET /api/v1/products/import-sample.
func (h *ProductImportHandler) ServeSampleCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="product_import_template.csv"`)
	io.WriteString(w, "\ufeff") // BOM for Excel
	w.Write([]byte(`name,sku,category,cost_price,selling_price,stock,minimum_stock,unit,barcode,variant
Es Kopi Susu,KOP-001,Minuman,8000,18000,100,10,pcs,8991234567890,Large
Es Kopi Susu,KOP-002,Minuman,8000,15000,100,10,pcs,,Small
Croissant,CRO-001,Makanan,5000,15000,50,5,pcs,,Regular
`))
}

// parseCSV reads a CSV file and returns rows (header + data).
// The first cell may carry a UTF-8 BOM (U+FEFF); strip it so the header
// lookup works even when the file was saved from Excel.
func parseCSV(r io.Reader) ([][]string, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	rows, err := cr.ReadAll()
	if err != nil || len(rows) == 0 {
		return rows, err
	}
	// Strip BOM from the first cell of the header row.
	rows[0][0] = strings.TrimPrefix(rows[0][0], "\ufeff")
	return rows, nil
}

// parseExcel reads the first sheet of an .xlsx file and returns rows.
func parseExcel(r io.Reader) ([][]string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	if sheet == "" {
		return nil, fmt.Errorf("no sheets found")
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read sheet: %w", err)
	}
	return rows, nil
}

// processRows validates and creates products from parsed rows.
func (h *ProductImportHandler) processRows(r *http.Request, storeID, userID uuid.UUID, rows [][]string) ImportResult {
	result := ImportResult{Total: max(0, len(rows)-1)}
	if len(rows) < 2 {
		return result
	}

	// Build category lookup: lowercase name → id.
	catMap := h.loadCategoryMap(r, storeID)

	header := normalizeHeader(rows[0])
	for i, row := range rows[1:] {
		lineNum := i + 2
		if err := h.importRow(r, storeID, userID, header, row, catMap); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, ImportRowError{Row: lineNum, Msg: err.Error()})
		} else {
			result.Imported++
		}
	}
	return result
}

func (h *ProductImportHandler) importRow(r *http.Request, storeID, userID uuid.UUID, header map[string]int, row []string, catMap map[string]string) error {
	get := func(key string) string {
		if idx, ok := header[key]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	name := get("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}
	sku := get("sku")
	if sku == "" {
		sku = "SKU-" + strings.ToUpper(uuid.New().String()[:8])
	}

	costPrice := parseNum(get("cost_price"))
	sellPrice := parseNum(get("selling_price"))
	stock := parseInt(get("stock"))
	minStock := parseInt(get("minimum_stock"))

	// Resolve category by name.
	catName := get("category")
	catID := catMap[strings.ToLower(catName)]
	if catID == "" && catName != "" {
		// Auto-create category if not found.
		cat, err := h.categoryService.CreateCategory(r.Context(), catName, "")
		if err == nil {
			catID = cat.ID.String()
		}
	}

	// Check SKU uniqueness in this store.
	if existing, _ := h.repos.ProductSKU.GetBySKUInStore(r.Context(), sku, storeID.String()); existing != nil {
		return fmt.Errorf("SKU '%s' already exists", sku)
	}

	// Create product + default SKU via service.
	barcode := get("barcode")
	unit := get("unit")
	if unit == "" {
		unit = "pcs"
	}
	variant := get("variant")

	prod, err := h.productService.CreateProductFromImport(r.Context(), product.CreateProductFromImportParams{
		StoreID:      storeID,
		UserID:       userID,
		Name:         name,
		CategoryID:   catID,
		CostPrice:    costPrice,
		SellingPrice: sellPrice,
		Stock:        stock,
		MinimumStock: minStock,
		SKU:          sku,
		Barcode:      barcode,
		Unit:         unit,
		Variant:      variant,
	})
	if err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	_ = prod
	return nil
}

func (h *ProductImportHandler) loadCategoryMap(r *http.Request, storeID uuid.UUID) map[string]string {
	result, err := h.categoryService.ListCategories(r.Context(), "", 1, 1000)
	if err != nil {
		return map[string]string{}
	}
	m := map[string]string{}
	for _, c := range result.Categories {
		m[strings.ToLower(c.Name)] = c.ID.String()
	}
	return m
}

func normalizeHeader(row []string) map[string]int {
	m := map[string]int{}
	for i, h := range row {
		h = strings.ToLower(strings.TrimSpace(h))
		h = strings.ReplaceAll(h, " ", "_")
		m[h] = i
	}
	return m
}

func parseNum(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseInt(s string) int {
	s = strings.ReplaceAll(s, ",", "")
	v, _ := strconv.Atoi(s)
	return v
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.Encode(data)
}

// max returns the larger of two ints.
