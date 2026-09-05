package productsku

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gormdb "apiPos/internal/database/gorm"
)

// mockProductSKURepo is a mock for ProductSKURepository
type mockProductSKURepo struct {
	createFunc       func(ctx context.Context, productSKU *gormdb.ProductSKU) error
	getByIDFunc      func(ctx context.Context, id uuid.UUID) (*gormdb.ProductSKU, error)
	getByBarcodeFunc func(ctx context.Context, barcode string) (*gormdb.ProductSKU, error)
	getBySKUFunc     func(ctx context.Context, sku string) (*gormdb.ProductSKU, error)
	listFunc         func(ctx context.Context, productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error)
	listAllFunc      func(ctx context.Context, search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error)
	updateFunc       func(ctx context.Context, productSKU *gormdb.ProductSKU) error
	deleteFunc       func(ctx context.Context, id uuid.UUID) error
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
	return nil, fmt.Errorf("not found")
}

func (m *mockProductSKURepo) GetByBarcode(ctx context.Context, barcode string) (*gormdb.ProductSKU, error) {
	if m.getByBarcodeFunc != nil {
		return m.getByBarcodeFunc(ctx, barcode)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockProductSKURepo) GetBySKU(ctx context.Context, sku string) (*gormdb.ProductSKU, error) {
	if m.getBySKUFunc != nil {
		return m.getBySKUFunc(ctx, sku)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockProductSKURepo) List(ctx context.Context, productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, productID, search, variant, color, size, isActive, page, perPage)
	}
	return []gormdb.ProductSKU{}, 0, nil
}

func (m *mockProductSKURepo) ListAll(ctx context.Context, search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
	if m.listAllFunc != nil {
		return m.listAllFunc(ctx, search, variant, color, size, isActive, productID, page, perPage)
	}
	return []gormdb.ProductSKU{}, 0, nil
}

func (m *mockProductSKURepo) Update(ctx context.Context, productSKU *gormdb.ProductSKU) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, productSKU)
	}
	return nil
}

func (m *mockProductSKURepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockProductSKURepo) CountByProductIDs(ctx context.Context, productIDs []uuid.UUID) (map[string]int32, error) {
	return make(map[string]int32), nil
}

func testProductSKU() *gormdb.ProductSKU {
	return &gormdb.ProductSKU{
		ID:           uuid.New(),
		ProductID:    uuid.New(),
		SKU:          "COF-L-001",
		Barcode:      "8991234567890",
		Variant:      "Large",
		Color:        "Brown",
		Size:         "L",
		Unit:         "pcs",
		CostPrice:    15000,
		SellingPrice: 25000,
		Stock:        100,
		MinimumStock: 10,
		Weight:       0.35,
		Volume:       0.5,
		BatchNumber:  "BATCH-2024-001",
		Location:     "A1-B2",
		ImageURL:     "https://example.com/image.jpg",
		IsActive:     true,
		Metadata:     map[string]any{"origin": "Indonesia"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func TestProductSKUService_CreateProductSKU(t *testing.T) {
	ctx := context.Background()
	_ = ctx

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			createFunc: func(ctx context.Context, productSKU *gormdb.ProductSKU) error {
				productSKU.ID = uuid.New()
				return nil
			},
		}
		svc := NewService(mockRepo)

		productSKU, err := svc.CreateProductSKU(ctx, 
			uuid.New().String(),
			"COF-L-001",
			"8991234567890",
			"Large",
			"Brown",
			"L",
			"pcs",
			15000,
			25000,
			100,
			10,
			0.35,
			0.5,
			nil,
			"BATCH-2024-001",
			"A1-B2",
			"https://example.com/image.jpg",
			true,
			map[string]string{"origin": "Indonesia"},
		)

		require.NoError(t, err)
		assert.Equal(t, "COF-L-001", productSKU.SKU)
		assert.Equal(t, "8991234567890", productSKU.Barcode)
		assert.Equal(t, "Large", productSKU.Variant)
		assert.Equal(t, "Brown", productSKU.Color)
		assert.Equal(t, "L", productSKU.Size)
		assert.Equal(t, float64(15000), productSKU.CostPrice)
		assert.Equal(t, float64(25000), productSKU.SellingPrice)
		assert.Equal(t, int32(100), productSKU.Stock)
		assert.True(t, productSKU.IsActive)
	})

	t.Run("create error", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			createFunc: func(ctx context.Context, productSKU *gormdb.ProductSKU) error {
				return fmt.Errorf("duplicate SKU")
			},
		}
		svc := NewService(mockRepo)

		_, err := svc.CreateProductSKU(ctx, 
			uuid.New().String(),
			"COF-L-001",
			"",
			"",
			"",
			"",
			"pcs",
			0,
			0,
			0,
			0,
			0,
			0,
			nil,
			"",
			"",
			"",
			true,
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate SKU")
	})
}

func TestProductSKUService_GetProductSKU(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*gormdb.ProductSKU, error) {
				return testProductSKU(), nil
			},
		}
		svc := NewService(mockRepo)

		productSKU, err := svc.GetProductSKU(ctx, validID.String())
		require.NoError(t, err)
		assert.Equal(t, "COF-L-001", productSKU.SKU)
	})

	t.Run("invalid id format", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{}
		svc := NewService(mockRepo)

		_, err := svc.GetProductSKU(ctx, "invalid-uuid")
		assert.Error(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*gormdb.ProductSKU, error) {
				return nil, fmt.Errorf("not found")
			},
		}
		svc := NewService(mockRepo)

		_, err := svc.GetProductSKU(ctx, validID.String())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProductSKUService_GetProductSKUByBarcode(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByBarcodeFunc: func(ctx context.Context, barcode string) (*gormdb.ProductSKU, error) {
				return testProductSKU(), nil
			},
		}
		svc := NewService(mockRepo)

		productSKU, err := svc.GetProductSKUByBarcode(ctx, "8991234567890")
		require.NoError(t, err)
		assert.Equal(t, "8991234567890", productSKU.Barcode)
	})

	t.Run("empty barcode", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{}
		svc := NewService(mockRepo)

		_, err := svc.GetProductSKUByBarcode(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByBarcodeFunc: func(ctx context.Context, barcode string) (*gormdb.ProductSKU, error) {
				return nil, fmt.Errorf("not found")
			},
		}
		svc := NewService(mockRepo)

		_, err := svc.GetProductSKUByBarcode(ctx, "0000000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProductSKUService_GetProductSKUBySKUCode(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getBySKUFunc: func(ctx context.Context, sku string) (*gormdb.ProductSKU, error) {
				return testProductSKU(), nil
			},
		}
		svc := NewService(mockRepo)

		productSKU, err := svc.GetProductSKUBySKUCode(ctx, "COF-L-001")
		require.NoError(t, err)
		assert.Equal(t, "COF-L-001", productSKU.SKU)
	})

	t.Run("empty sku", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{}
		svc := NewService(mockRepo)

		_, err := svc.GetProductSKUBySKUCode(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getBySKUFunc: func(ctx context.Context, sku string) (*gormdb.ProductSKU, error) {
				return nil, fmt.Errorf("not found")
			},
		}
		svc := NewService(mockRepo)

		_, err := svc.GetProductSKUBySKUCode(ctx, "NONEXISTENT")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProductSKUService_ListProductSKUs(t *testing.T) {
	ctx := context.Background()
	validProductID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			listFunc: func(ctx context.Context, productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
				return []gormdb.ProductSKU{
					*testProductSKU(),
					*testProductSKU(),
				}, 2, nil
			},
		}
		svc := NewService(mockRepo)

		result, totalCount, err := svc.ListProductSKUs(ctx, validProductID, "", "", "", "", nil, 1, 10)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(2), totalCount)
	})

	t.Run("invalid product id", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{}
		svc := NewService(mockRepo)

		_, _, err := svc.ListProductSKUs(ctx, "invalid-uuid", "", "", "", "", nil, 1, 10)
		assert.Error(t, err)
	})

	t.Run("empty list", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			listFunc: func(ctx context.Context, productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
				return []gormdb.ProductSKU{}, 0, nil
			},
		}
		svc := NewService(mockRepo)

		result, totalCount, err := svc.ListProductSKUs(ctx, validProductID, "", "", "", "", nil, 1, 10)
		require.NoError(t, err)
		assert.Empty(t, result)
		assert.Equal(t, int64(0), totalCount)
	})
}

func TestProductSKUService_ListAllProductSKUs(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			listAllFunc: func(ctx context.Context, search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
				return []gormdb.ProductSKU{
					*testProductSKU(),
				}, 1, nil
			},
		}
		svc := NewService(mockRepo)

		result, totalCount, err := svc.ListAllProductSKUs(ctx, "", "", "", "", nil, "", 1, 10)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, int64(1), totalCount)
	})

	t.Run("with filters", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			listAllFunc: func(ctx context.Context, search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
				return []gormdb.ProductSKU{}, 0, nil
			},
		}
		svc := NewService(mockRepo)

		result, _, err := svc.ListAllProductSKUs(ctx, "search", "variant", "color", "size", nil, uuid.New().String(), 1, 10)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestProductSKUService_UpdateProductSKU(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*gormdb.ProductSKU, error) {
				return testProductSKU(), nil
			},
			updateFunc: func(ctx context.Context, productSKU *gormdb.ProductSKU) error {
				return nil
			},
		}
		svc := NewService(mockRepo)

		productSKU, err := svc.UpdateProductSKU(ctx, 
			validID,
			"COF-L-002",
			"8991234567891",
			"Medium",
			"Black",
			"M",
			"pcs",
			20000,
			35000,
			50,
			5,
			0.25,
			0.3,
			nil,
			"BATCH-2024-002",
			"B1-C3",
			"https://example.com/image2.jpg",
			true,
			nil,
		)

		require.NoError(t, err)
		assert.Equal(t, "COF-L-002", productSKU.SKU)
		assert.Equal(t, "8991234567891", productSKU.Barcode)
		assert.Equal(t, "Medium", productSKU.Variant)
		assert.Equal(t, "Black", productSKU.Color)
		assert.Equal(t, "M", productSKU.Size)
		assert.Equal(t, float64(20000), productSKU.CostPrice)
		assert.Equal(t, float64(35000), productSKU.SellingPrice)
	})

	t.Run("invalid id", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{}
		svc := NewService(mockRepo)

		_, err := svc.UpdateProductSKU(ctx, "invalid-uuid", "", "", "", "", "", "", 0, 0, 0, 0, 0, 0, nil, "", "", "", true, nil)
		assert.Error(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*gormdb.ProductSKU, error) {
				return nil, fmt.Errorf("not found")
			},
		}
		svc := NewService(mockRepo)

		_, err := svc.UpdateProductSKU(ctx, validID, "", "", "", "", "", "", 0, 0, 0, 0, 0, 0, nil, "", "", "", true, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("partial update", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*gormdb.ProductSKU, error) {
				return testProductSKU(), nil
			},
			updateFunc: func(ctx context.Context, productSKU *gormdb.ProductSKU) error {
				return nil
			},
		}
		svc := NewService(mockRepo)

		productSKU, err := svc.UpdateProductSKU(ctx, 
			validID,
			"",
			"",
			"",
			"",
			"",
			"",
			0,
			30000,
			0,
			0,
			0,
			0,
			nil,
			"",
			"",
			"",
			true,
			nil,
		)

		require.NoError(t, err)
		assert.Equal(t, "COF-L-001", productSKU.SKU)
		assert.Equal(t, float64(30000), productSKU.SellingPrice)
	})
}

func TestProductSKUService_DeleteProductSKU(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}
		svc := NewService(mockRepo)

		err := svc.DeleteProductSKU(ctx, validID)
		require.NoError(t, err)
	})

	t.Run("invalid id", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{}
		svc := NewService(mockRepo)

		err := svc.DeleteProductSKU(ctx, "invalid-uuid")
		assert.Error(t, err)
	})

	t.Run("delete error", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return fmt.Errorf("delete failed")
			},
		}
		svc := NewService(mockRepo)

		err := svc.DeleteProductSKU(ctx, validID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})
}

func TestProductSKUService_PaginationNormalization(t *testing.T) {
	ctx := context.Background()
	validProductID := uuid.New().String()

	t.Run("default pagination", func(t *testing.T) {
		var capturedPage, capturedPerPage int
		mockRepo := &mockProductSKURepo{
			listFunc: func(ctx context.Context, productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
				capturedPage = page
				capturedPerPage = perPage
				return []gormdb.ProductSKU{}, 0, nil
			},
		}
		svc := NewService(mockRepo)

		_, _, _ = svc.ListProductSKUs(ctx, validProductID, "", "", "", "", nil, 0, 0)
		assert.Equal(t, 1, capturedPage)
		assert.Equal(t, 20, capturedPerPage)
	})

	t.Run("max per page", func(t *testing.T) {
		var capturedPerPage int
		mockRepo := &mockProductSKURepo{
			listFunc: func(ctx context.Context, productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
				capturedPerPage = perPage
				return []gormdb.ProductSKU{}, 0, nil
			},
		}
		svc := NewService(mockRepo)

		_, _, _ = svc.ListProductSKUs(ctx, validProductID, "", "", "", "", nil, 1, 200)
		assert.Equal(t, 100, capturedPerPage)
	})
}
