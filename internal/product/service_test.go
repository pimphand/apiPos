package product

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

// mockProductRepo is a mock for ProductRepository
type mockProductRepo struct {
	createFunc       func(ctx context.Context, product *gormdb.Product) error
	getByIDFunc      func(ctx context.Context, id string) (*gormdb.Product, error)
	listFunc         func(ctx context.Context, params repository.ProductListParams) ([]gormdb.Product, int64, error)
	searchByNameFunc func(ctx context.Context, params repository.ProductSearchParams) ([]gormdb.Product, int64, error)
	updateFunc       func(ctx context.Context, product *gormdb.Product) error
	deleteFunc       func(ctx context.Context, id string) error
	activateFunc     func(ctx context.Context, id string) error
	deactivateFunc   func(ctx context.Context, id string) error
	getLowStockFunc  func(ctx context.Context, params repository.PaginationParams) ([]gormdb.Product, int64, error)
}

func (m *mockProductRepo) Create(ctx context.Context, product *gormdb.Product) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, product)
	}
	return nil
}

func (m *mockProductRepo) GetByID(ctx context.Context, id string) (*gormdb.Product, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockProductRepo) GetBySKU(ctx context.Context, sku string) (*gormdb.Product, error) {
	return nil, gorm.ErrRecordNotFound
}

func (m *mockProductRepo) GetByBarcode(ctx context.Context, barcode string) (*gormdb.Product, error) {
	return nil, gorm.ErrRecordNotFound
}

func (m *mockProductRepo) List(ctx context.Context, params repository.ProductListParams) ([]gormdb.Product, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, params)
	}
	return []gormdb.Product{}, 0, nil
}

func (m *mockProductRepo) SearchByName(ctx context.Context, params repository.ProductSearchParams) ([]gormdb.Product, int64, error) {
	if m.searchByNameFunc != nil {
		return m.searchByNameFunc(ctx, params)
	}
	return []gormdb.Product{}, 0, nil
}

func (m *mockProductRepo) Update(ctx context.Context, product *gormdb.Product) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, product)
	}
	return nil
}

func (m *mockProductRepo) UpdateStock(ctx context.Context, id string, stock int32) error {
	return nil
}

func (m *mockProductRepo) DecreaseStock(ctx context.Context, id string, quantity int32) error {
	return nil
}

func (m *mockProductRepo) IncreaseStock(ctx context.Context, id string, quantity int32) error {
	return nil
}

func (m *mockProductRepo) Activate(ctx context.Context, id string) error {
	if m.activateFunc != nil {
		return m.activateFunc(ctx, id)
	}
	return nil
}

func (m *mockProductRepo) Deactivate(ctx context.Context, id string) error {
	if m.deactivateFunc != nil {
		return m.deactivateFunc(ctx, id)
	}
	return nil
}

func (m *mockProductRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockProductRepo) GetLowStock(ctx context.Context, params repository.PaginationParams) ([]gormdb.Product, int64, error) {
	if m.getLowStockFunc != nil {
		return m.getLowStockFunc(ctx, params)
	}
	return []gormdb.Product{}, 0, nil
}

func (m *mockProductRepo) GetStockForUpdate(ctx context.Context, id string) (*gormdb.Product, error) {
	return nil, nil
}

// mockProductSKURepo is a mock for ProductSKURepository
type mockProductSKURepo struct {
	createFunc       func(ctx context.Context, productSKU *gormdb.ProductSKU) error
	getByIDFunc      func(ctx context.Context, id uuid.UUID) (*gormdb.ProductSKU, error)
	getByBarcodeFunc func(ctx context.Context, barcode string) (*gormdb.ProductSKU, error)
	getBySKUFunc     func(ctx context.Context, sku string) (*gormdb.ProductSKU, error)
}

func (m *mockProductSKURepo) Create(ctx context.Context, productSKU *gormdb.ProductSKU) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, productSKU)
	}
	return nil
}

func (m *mockProductSKURepo) GetByID(ctx context.Context, id uuid.UUID) (*gormdb.ProductSKU, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockProductSKURepo) GetByBarcode(ctx context.Context, barcode string) (*gormdb.ProductSKU, error) {
	if m.getByBarcodeFunc != nil {
		return m.getByBarcodeFunc(ctx, barcode)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockProductSKURepo) GetBySKU(ctx context.Context, sku string) (*gormdb.ProductSKU, error) {
	if m.getBySKUFunc != nil {
		return m.getBySKUFunc(ctx, sku)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockProductSKURepo) List(ctx context.Context, productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
	return []gormdb.ProductSKU{}, 0, nil
}

func (m *mockProductSKURepo) ListAll(ctx context.Context, search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
	return []gormdb.ProductSKU{}, 0, nil
}

func (m *mockProductSKURepo) Update(ctx context.Context, productSKU *gormdb.ProductSKU) error {
	return nil
}

func (m *mockProductSKURepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockProductSKURepo) GetBySKUInStore(ctx context.Context, sku string, storeID string) (*gormdb.ProductSKU, error) {
	return nil, gorm.ErrRecordNotFound
}

func (m *mockProductSKURepo) CountByProductIDs(ctx context.Context, productIDs []uuid.UUID) (map[string]int32, error) {
	return make(map[string]int32), nil
}

func testProduct() *gormdb.Product {
	return &gormdb.Product{
		ID:           uuid.New(),
		Name:         "Test Product",
		Description:  "Test description",
		CategoryID:   uuid.New(),
		CostPrice:    5000,
		SellingPrice: 10000,
		Stock:        100,
		MinimumStock: 10,
		IsActive:     true,
	}
}

func TestProductService_CreateProduct(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			createFunc: func(ctx context.Context, product *gormdb.Product) error {
				product.ID = uuid.New()
				return nil
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		product, err := svc.CreateProduct(ctx, CreateProductRequest{
			Name:         "Test Product",
			Description:  "Test description",
			CategoryID:   uuid.New().String(),
			CostPrice:    5000,
			SellingPrice: 10000,
			Stock:        100,
			MinimumStock: 10,
		})

		require.NoError(t, err)
		assert.Equal(t, "Test Product", product.Name)
		assert.Equal(t, int32(100), product.Stock)
	})
}

func TestProductService_GetProduct(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Product, error) {
				return testProduct(), nil
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		product, err := svc.GetProduct(ctx, validID)
		require.NoError(t, err)
		assert.Equal(t, "Test Product", product.Name)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Product, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		_, err := svc.GetProduct(ctx, validID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProductService_GetProductByBarcode(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductRepo{}

		mockProductSKURepo := &mockProductSKURepo{
			getByBarcodeFunc: func(ctx context.Context, barcode string) (*gormdb.ProductSKU, error) {
				return &gormdb.ProductSKU{
					ID:        uuid.New(),
					ProductID: uuid.New(),
					SKU:       "SKU001",
					Barcode:   "1234567890",
				}, nil
			},
		}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		productSKU, err := svc.GetProductByBarcode(ctx, "1234567890")
		require.NoError(t, err)
		assert.Equal(t, "SKU001", productSKU.SKU)
	})

	t.Run("empty barcode", func(t *testing.T) {
		mockRepo := &mockProductRepo{}
		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		_, err := svc.GetProductByBarcode(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductRepo{}

		mockProductSKURepo := &mockProductSKURepo{
			getByBarcodeFunc: func(ctx context.Context, barcode string) (*gormdb.ProductSKU, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		_, err := svc.GetProductByBarcode(ctx, "0000000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProductService_GetProductBySKU(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductRepo{}

		mockProductSKURepo := &mockProductSKURepo{
			getBySKUFunc: func(ctx context.Context, sku string) (*gormdb.ProductSKU, error) {
				return &gormdb.ProductSKU{
					ID:        uuid.New(),
					ProductID: uuid.New(),
					SKU:       "SKU001",
					Barcode:   "1234567890",
				}, nil
			},
		}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		productSKU, err := svc.GetProductBySKU(ctx, "SKU001")
		require.NoError(t, err)
		assert.Equal(t, "SKU001", productSKU.SKU)
	})

	t.Run("empty sku", func(t *testing.T) {
		mockRepo := &mockProductRepo{}
		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		_, err := svc.GetProductBySKU(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductRepo{}

		mockProductSKURepo := &mockProductSKURepo{
			getBySKUFunc: func(ctx context.Context, sku string) (*gormdb.ProductSKU, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		_, err := svc.GetProductBySKU(ctx, "NONEXISTENT")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProductService_ListProducts(t *testing.T) {
	ctx := context.Background()

	t.Run("success with pagination", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			listFunc: func(ctx context.Context, params repository.ProductListParams) ([]gormdb.Product, int64, error) {
				return []gormdb.Product{
					{ID: uuid.New(), Name: "Product 1", Stock: 100},
					{ID: uuid.New(), Name: "Product 2", Stock: 50},
				}, 2, nil
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		result, err := svc.ListProducts(ctx, "", "", 1, 10)
		require.NoError(t, err)
		assert.Len(t, result.Products, 2)
		assert.Equal(t, int64(2), result.Pagination.TotalCount)
		assert.Equal(t, int32(1), result.Pagination.Page)
		assert.Equal(t, int32(10), result.Pagination.PerPage)
	})

	t.Run("empty list", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			listFunc: func(ctx context.Context, params repository.ProductListParams) ([]gormdb.Product, int64, error) {
				return []gormdb.Product{}, 0, nil
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		result, err := svc.ListProducts(ctx, "", "", 1, 10)
		require.NoError(t, err)
		assert.Empty(t, result.Products)
		assert.Equal(t, int64(0), result.Pagination.TotalCount)
	})
}

func TestProductService_SearchProducts(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			searchByNameFunc: func(ctx context.Context, params repository.ProductSearchParams) ([]gormdb.Product, int64, error) {
				return []gormdb.Product{
					{ID: uuid.New(), Name: "Coffee", Stock: 100},
				}, 1, nil
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		result, err := svc.SearchProducts(ctx, "Coffee", "", 1, 10)
		require.NoError(t, err)
		assert.Len(t, result.Products, 1)
		assert.Equal(t, "Coffee", result.Products[0].Name)
		assert.Equal(t, int64(1), result.Pagination.TotalCount)
	})
}

func TestProductService_UpdateProduct(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Product, error) {
				return testProduct(), nil
			},
			updateFunc: func(ctx context.Context, product *gormdb.Product) error {
				return nil
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		product, err := svc.UpdateProduct(ctx, UpdateProductRequest{
			ID:           validID,
			Name:         "Updated Product",
			SellingPrice: 15000,
			IsActive:     true,
		})

		require.NoError(t, err)
		assert.Equal(t, "Updated Product", product.Name)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Product, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		_, err := svc.UpdateProduct(ctx, UpdateProductRequest{
			ID:   validID,
			Name: "Updated",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProductService_DeleteProduct(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Product, error) {
				return testProduct(), nil
			},
			deleteFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		err := svc.DeleteProduct(ctx, validID)
		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Product, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		err := svc.DeleteProduct(ctx, validID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProductService_ActivateDeactivateProduct(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New().String()

	t.Run("activate success", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Product, error) {
				p := testProduct()
				p.IsActive = false
				return p, nil
			},
			activateFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		err := svc.ActivateProduct(ctx, validID)
		require.NoError(t, err)
	})

	t.Run("deactivate success", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Product, error) {
				return testProduct(), nil
			},
			deactivateFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		err := svc.DeactivateProduct(ctx, validID)
		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			getByIDFunc: func(ctx context.Context, id string) (*gormdb.Product, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		err := svc.DeactivateProduct(ctx, validID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProductService_GetLowStockProducts(t *testing.T) {
	ctx := context.Background()

	t.Run("success with pagination", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			getLowStockFunc: func(ctx context.Context, params repository.PaginationParams) ([]gormdb.Product, int64, error) {
				return []gormdb.Product{
					{ID: uuid.New(), Name: "Low Stock Product", Stock: 5, MinimumStock: 10},
				}, 1, nil
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		result, err := svc.GetLowStockProducts(ctx, 1, 10)
		require.NoError(t, err)
		assert.Len(t, result.Products, 1)
		assert.Equal(t, int64(1), result.Pagination.TotalCount)
		assert.LessOrEqual(t, result.Products[0].Stock, result.Products[0].MinimumStock)
	})
}

func TestProductService_CountProducts(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			listFunc: func(ctx context.Context, params repository.ProductListParams) ([]gormdb.Product, int64, error) {
				return []gormdb.Product{}, 25, nil
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		result, err := svc.ListProducts(ctx, "", "", 1, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(25), result.Pagination.TotalCount)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo := &mockProductRepo{
			listFunc: func(ctx context.Context, params repository.ProductListParams) ([]gormdb.Product, int64, error) {
				return nil, 0, fmt.Errorf("database error")
			},
		}

		mockProductSKURepo := &mockProductSKURepo{}

		repos := &repository.Repositories{
			Product:    mockRepo,
			ProductSKU: mockProductSKURepo,
		}
		svc := NewService(repos)

		_, err := svc.ListProducts(ctx, "", "", 1, 10)
		assert.Error(t, err)
	})
}
