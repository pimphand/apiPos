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
Es Kopi Susu,KOP-001,Minuman,8000,18000,100,10,pcs,8991234567890,Regular
Es Kopi Susu,KOP-002,Minuman,9000,21000,100,10,pcs,8991234567891,Large
Es Kopi Hitam,KPH-001,Minuman,4000,12000,100,10,pcs,8991234567892,Hot
Es Kopi Hitam,KPH-002,Minuman,5000,14000,100,10,pcs,8991234567893,Ice
Es Kopi Taro,TAR-001,Minuman,9000,22000,80,10,pcs,8991234567894,Hot
Es Kopi Taro,TAR-002,Minuman,10000,24000,80,10,pcs,8991234567895,Ice
Es Kopi Pandan,PDN-001,Minuman,9000,22000,80,10,pcs,8991234567896,Hot
Es Kopi Pandan,PDN-002,Minuman,10000,24000,80,10,pcs,8991234567897,Ice
Americano,AME-001,Minuman,5000,15000,120,15,pcs,8991234567901,Hot
Americano,AME-002,Minuman,6000,17000,120,15,pcs,8991234567902,Ice
Long Black,LBL-001,Minuman,5000,15000,100,10,pcs,8991234567903,Hot
Long Black,LBL-002,Minuman,6000,17000,100,10,pcs,8991234567904,Ice
Espresso,ESP-001,Minuman,4000,12000,150,15,pcs,8991234567905,Single
Espresso,ESP-002,Minuman,6000,18000,150,15,pcs,8991234567906,Double
Cappuccino,CAP-001,Minuman,7000,20000,100,10,pcs,8991234567911,Hot
Cappuccino,CAP-002,Minuman,8000,22000,100,10,pcs,8991234567912,Ice
Caffe Latte,LAT-001,Minuman,7000,20000,100,10,pcs,8991234567921,Hot
Caffe Latte,LAT-002,Minuman,8000,22000,100,10,pcs,8991234567922,Ice
Vanilla Latte,VNL-001,Minuman,9000,24000,80,10,pcs,8991234567923,Hot
Vanilla Latte,VNL-002,Minuman,10000,26000,80,10,pcs,8991234567924,Ice
Hazelnut Latte,HNL-001,Minuman,9000,24000,80,10,pcs,8991234567925,Hot
Hazelnut Latte,HNL-002,Minuman,10000,26000,80,10,pcs,8991234567926,Ice
Caramel Latte,CAR-001,Minuman,9000,24000,80,10,pcs,8991234567931,Hot
Caramel Latte,CAR-002,Minuman,10000,26000,80,10,pcs,8991234567932,Ice
Mocha,MOC-001,Minuman,9000,24000,80,10,pcs,8991234567941,Hot
Mocha,MOC-002,Minuman,10000,26000,80,10,pcs,8991234567942,Ice
White Chocolate Mocha,WCM-001,Minuman,10000,26000,80,10,pcs,8991234567943,Hot
White Chocolate Mocha,WCM-002,Minuman,11000,28000,80,10,pcs,8991234567944,Ice
Es Kopi Gula Aren,GAR-001,Minuman,8500,20000,100,10,pcs,8991234567951,Regular
Es Kopi Gula Aren,GAR-002,Minuman,9500,23000,100,10,pcs,8991234567952,Large
Kopi Susu Aren,KSA-001,Minuman,8500,20000,100,10,pcs,8991234567953,Hot
Kopi Susu Aren,KSA-002,Minuman,9500,23000,100,10,pcs,8991234567954,Ice
Salted Caramel Coffee,SCC-001,Minuman,9500,24000,80,10,pcs,8991234567955,Hot
Salted Caramel Coffee,SCC-002,Minuman,10500,26000,80,10,pcs,8991234567956,Ice
Dalgona Coffee,DAL-001,Minuman,9500,24000,80,10,pcs,8991234567957,Cold
Affogato,AFF-001,Minuman,10000,28000,60,5,pcs,8991234567958,Regular
Flat White,FLW-001,Minuman,8000,22000,80,10,pcs,8991234567959,Hot
Cortado,COR-001,Minuman,7000,19000,80,10,pcs,8991234567960,Hot
Cold Brew,CLB-001,Minuman,8000,20000,80,10,pcs,8991234567961,Regular
Cold Brew Latte,CBL-001,Minuman,9500,24000,80,10,pcs,8991234567962,Ice
Croissant,CRO-001,Makanan,5000,15000,50,5,pcs,8992234567890,Regular
Butter Croissant,BCR-001,Makanan,5500,17000,50,5,pcs,8992234567891,Regular
Chocolate Croissant,CCR-001,Makanan,6000,19000,50,5,pcs,8992234567892,Regular
Almond Croissant,ACR-001,Makanan,6500,21000,40,5,pcs,8992234567893,Regular
Roti Pisang,KUE-001,Makanan,3000,9000,60,10,pcs,8992234567894,Regular
Donat Coklat,DNT-001,Makanan,2500,8000,80,10,pcs,8992234567895,Coklat
Donat Gula,DNT-002,Makanan,2000,7000,80,10,pcs,8992234567896,Gula
Cheesecake Slice,CKE-001,Makanan,9000,25000,40,5,pcs,8992234567897,Regular
Brownies,BWN-001,Makanan,6000,15000,60,10,pcs,8992234567898,Regular
Muffin Coklat,MUF-001,Makanan,5000,14000,60,10,pcs,8992234567899,Coklat
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
