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
	createFunc       func(productSKU *gormdb.ProductSKU) error
	getByIDFunc      func(id uuid.UUID) (*gormdb.ProductSKU, error)
	getByBarcodeFunc func(barcode string) (*gormdb.ProductSKU, error)
	getBySKUFunc     func(sku string) (*gormdb.ProductSKU, error)
	listFunc         func(productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error)
	listAllFunc      func(search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error)
	updateFunc       func(productSKU *gormdb.ProductSKU) error
	deleteFunc       func(id uuid.UUID) error
}

func (m *mockProductSKURepo) Create(productSKU *gormdb.ProductSKU) error {
	if m.createFunc != nil {
		return m.createFunc(productSKU)
	}
	return nil
}

func (m *mockProductSKURepo) GetByID(id uuid.UUID) (*gormdb.ProductSKU, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockProductSKURepo) GetByBarcode(barcode string) (*gormdb.ProductSKU, error) {
	if m.getByBarcodeFunc != nil {
		return m.getByBarcodeFunc(barcode)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockProductSKURepo) GetBySKU(sku string) (*gormdb.ProductSKU, error) {
	if m.getBySKUFunc != nil {
		return m.getBySKUFunc(sku)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockProductSKURepo) List(productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(productID, search, variant, color, size, isActive, page, perPage)
	}
	return []gormdb.ProductSKU{}, 0, nil
}

func (m *mockProductSKURepo) ListAll(search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
	if m.listAllFunc != nil {
		return m.listAllFunc(search, variant, color, size, isActive, productID, page, perPage)
	}
	return []gormdb.ProductSKU{}, 0, nil
}

func (m *mockProductSKURepo) Update(productSKU *gormdb.ProductSKU) error {
	if m.updateFunc != nil {
		return m.updateFunc(productSKU)
	}
	return nil
}

func (m *mockProductSKURepo) Delete(id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *mockProductSKURepo) CountByProductIDs(productIDs []uuid.UUID) (map[string]int32, error) {
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
			createFunc: func(productSKU *gormdb.ProductSKU) error {
				productSKU.ID = uuid.New()
				return nil
			},
		}
		svc := NewService(mockRepo)

		productSKU, err := svc.CreateProductSKU(
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
			createFunc: func(productSKU *gormdb.ProductSKU) error {
				return fmt.Errorf("duplicate SKU")
			},
		}
		svc := NewService(mockRepo)

		_, err := svc.CreateProductSKU(
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
	validID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByIDFunc: func(id uuid.UUID) (*gormdb.ProductSKU, error) {
				return testProductSKU(), nil
			},
		}
		svc := NewService(mockRepo)

		productSKU, err := svc.GetProductSKU(validID.String())
		require.NoError(t, err)
		assert.Equal(t, "COF-L-001", productSKU.SKU)
	})

	t.Run("invalid id format", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{}
		svc := NewService(mockRepo)

		_, err := svc.GetProductSKU("invalid-uuid")
		assert.Error(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByIDFunc: func(id uuid.UUID) (*gormdb.ProductSKU, error) {
				return nil, fmt.Errorf("not found")
			},
		}
		svc := NewService(mockRepo)

		_, err := svc.GetProductSKU(validID.String())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProductSKUService_GetProductSKUByBarcode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByBarcodeFunc: func(barcode string) (*gormdb.ProductSKU, error) {
				return testProductSKU(), nil
			},
		}
		svc := NewService(mockRepo)

		productSKU, err := svc.GetProductSKUByBarcode("8991234567890")
		require.NoError(t, err)
		assert.Equal(t, "8991234567890", productSKU.Barcode)
	})

	t.Run("empty barcode", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{}
		svc := NewService(mockRepo)

		_, err := svc.GetProductSKUByBarcode("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByBarcodeFunc: func(barcode string) (*gormdb.ProductSKU, error) {
				return nil, fmt.Errorf("not found")
			},
		}
		svc := NewService(mockRepo)

		_, err := svc.GetProductSKUByBarcode("0000000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProductSKUService_GetProductSKUBySKUCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getBySKUFunc: func(sku string) (*gormdb.ProductSKU, error) {
				return testProductSKU(), nil
			},
		}
		svc := NewService(mockRepo)

		productSKU, err := svc.GetProductSKUBySKUCode("COF-L-001")
		require.NoError(t, err)
		assert.Equal(t, "COF-L-001", productSKU.SKU)
	})

	t.Run("empty sku", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{}
		svc := NewService(mockRepo)

		_, err := svc.GetProductSKUBySKUCode("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getBySKUFunc: func(sku string) (*gormdb.ProductSKU, error) {
				return nil, fmt.Errorf("not found")
			},
		}
		svc := NewService(mockRepo)

		_, err := svc.GetProductSKUBySKUCode("NONEXISTENT")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProductSKUService_ListProductSKUs(t *testing.T) {
	validProductID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			listFunc: func(productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
				return []gormdb.ProductSKU{
					*testProductSKU(),
					*testProductSKU(),
				}, 2, nil
			},
		}
		svc := NewService(mockRepo)

		result, totalCount, err := svc.ListProductSKUs(validProductID, "", "", "", "", nil, 1, 10)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(2), totalCount)
	})

	t.Run("invalid product id", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{}
		svc := NewService(mockRepo)

		_, _, err := svc.ListProductSKUs("invalid-uuid", "", "", "", "", nil, 1, 10)
		assert.Error(t, err)
	})

	t.Run("empty list", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			listFunc: func(productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
				return []gormdb.ProductSKU{}, 0, nil
			},
		}
		svc := NewService(mockRepo)

		result, totalCount, err := svc.ListProductSKUs(validProductID, "", "", "", "", nil, 1, 10)
		require.NoError(t, err)
		assert.Empty(t, result)
		assert.Equal(t, int64(0), totalCount)
	})
}

func TestProductSKUService_ListAllProductSKUs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			listAllFunc: func(search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
				return []gormdb.ProductSKU{
					*testProductSKU(),
				}, 1, nil
			},
		}
		svc := NewService(mockRepo)

		result, totalCount, err := svc.ListAllProductSKUs("", "", "", "", nil, "", 1, 10)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, int64(1), totalCount)
	})

	t.Run("with filters", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			listAllFunc: func(search, variant, color, size string, isActive *bool, productID uuid.UUID, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
				return []gormdb.ProductSKU{}, 0, nil
			},
		}
		svc := NewService(mockRepo)

		result, _, err := svc.ListAllProductSKUs("search", "variant", "color", "size", nil, uuid.New().String(), 1, 10)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestProductSKUService_UpdateProductSKU(t *testing.T) {
	validID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByIDFunc: func(id uuid.UUID) (*gormdb.ProductSKU, error) {
				return testProductSKU(), nil
			},
			updateFunc: func(productSKU *gormdb.ProductSKU) error {
				return nil
			},
		}
		svc := NewService(mockRepo)

		productSKU, err := svc.UpdateProductSKU(
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

		_, err := svc.UpdateProductSKU("invalid-uuid", "", "", "", "", "", "", 0, 0, 0, 0, 0, 0, nil, "", "", "", true, nil)
		assert.Error(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByIDFunc: func(id uuid.UUID) (*gormdb.ProductSKU, error) {
				return nil, fmt.Errorf("not found")
			},
		}
		svc := NewService(mockRepo)

		_, err := svc.UpdateProductSKU(validID, "", "", "", "", "", "", 0, 0, 0, 0, 0, 0, nil, "", "", "", true, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("partial update", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			getByIDFunc: func(id uuid.UUID) (*gormdb.ProductSKU, error) {
				return testProductSKU(), nil
			},
			updateFunc: func(productSKU *gormdb.ProductSKU) error {
				return nil
			},
		}
		svc := NewService(mockRepo)

		productSKU, err := svc.UpdateProductSKU(
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
	validID := uuid.New().String()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			deleteFunc: func(id uuid.UUID) error {
				return nil
			},
		}
		svc := NewService(mockRepo)

		err := svc.DeleteProductSKU(validID)
		require.NoError(t, err)
	})

	t.Run("invalid id", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{}
		svc := NewService(mockRepo)

		err := svc.DeleteProductSKU("invalid-uuid")
		assert.Error(t, err)
	})

	t.Run("delete error", func(t *testing.T) {
		mockRepo := &mockProductSKURepo{
			deleteFunc: func(id uuid.UUID) error {
				return fmt.Errorf("delete failed")
			},
		}
		svc := NewService(mockRepo)

		err := svc.DeleteProductSKU(validID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})
}

func TestProductSKUService_PaginationNormalization(t *testing.T) {
	validProductID := uuid.New().String()

	t.Run("default pagination", func(t *testing.T) {
		var capturedPage, capturedPerPage int
		mockRepo := &mockProductSKURepo{
			listFunc: func(productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
				capturedPage = page
				capturedPerPage = perPage
				return []gormdb.ProductSKU{}, 0, nil
			},
		}
		svc := NewService(mockRepo)

		_, _, _ = svc.ListProductSKUs(validProductID, "", "", "", "", nil, 0, 0)
		assert.Equal(t, 1, capturedPage)
		assert.Equal(t, 20, capturedPerPage)
	})

	t.Run("max per page", func(t *testing.T) {
		var capturedPerPage int
		mockRepo := &mockProductSKURepo{
			listFunc: func(productID uuid.UUID, search, variant, color, size string, isActive *bool, page, perPage int) ([]gormdb.ProductSKU, int64, error) {
				capturedPerPage = perPage
				return []gormdb.ProductSKU{}, 0, nil
			},
		}
		svc := NewService(mockRepo)

		_, _, _ = svc.ListProductSKUs(validProductID, "", "", "", "", nil, 1, 200)
		assert.Equal(t, 100, capturedPerPage)
	})
}
