package settings

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

	"apiPos/internal/apperror"
	"apiPos/internal/auth"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/repository"
)

const (
	testStoreID = "550e8400-e29b-41d4-a716-446655440000"
	testUserID  = "650e8400-e29b-41d4-a716-446655440001"
)

type fakeSettingsRepo struct {
	rows map[string]*gormdb.StoreSetting
}

func (f *fakeSettingsRepo) GetByStoreID(ctx context.Context, storeID string) (*gormdb.StoreSetting, error) {
	if s, ok := f.rows[storeID]; ok {
		return s, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeSettingsRepo) Upsert(ctx context.Context, setting *gormdb.StoreSetting) error {
	f.rows[setting.StoreID.String()] = setting
	return nil
}

func newFake() *fakeSettingsRepo {
	return &fakeSettingsRepo{rows: map[string]*gormdb.StoreSetting{}}
}

func claimsCtx(role string) context.Context {
	return auth.ContextWithClaims(context.Background(), &auth.Claims{
		UserID:   testUserID,
		Username: "tester",
		RoleName: role,
		StoreID:  testStoreID,
	})
}

func grpcCode(err error) codes.Code {
	mapped := apperror.MapError(err)
	if mapped == nil {
		return codes.OK
	}
	return mapped.GRPCStatus().Code()
}

func TestGetSettings_DefaultsWhenNoRow(t *testing.T) {
	repos := &repository.Repositories{Settings: newFake()}
	svc := NewService(repos)

	s, err := svc.GetSettings(claimsCtx("cashier"))
	assert.NoError(t, err)
	assert.Equal(t, DefaultTaxRate, s.TaxRate)
	assert.True(t, s.ShowStoreName)
	assert.True(t, s.ShowStoreContact)
	assert.True(t, s.ShowCashier)
	assert.False(t, s.ShowItemSKU)
	assert.False(t, s.ShowItemBarcode)
}

func TestGetSettings_Unauthenticated(t *testing.T) {
	repos := &repository.Repositories{Settings: newFake()}
	svc := NewService(repos)

	_, err := svc.GetSettings(context.Background())
	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, grpcCode(err))
}

func TestGetSettings_SuperadminWithoutStore(t *testing.T) {
	repos := &repository.Repositories{Settings: newFake()}
	svc := NewService(repos)

	ctx := auth.ContextWithClaims(context.Background(), &auth.Claims{
		UserID:   testUserID,
		RoleName: "superadmin",
	})
	_, err := svc.GetSettings(ctx)
	assert.Error(t, err)
}

func TestUpdateSettings_ForbiddenForCashier(t *testing.T) {
	repos := &repository.Repositories{Settings: newFake()}
	svc := NewService(repos)

	_, err := svc.UpdateSettings(claimsCtx("cashier"), UpdateParams{TaxRate: 11})
	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, grpcCode(err))
	assert.True(t, strings.Contains(err.Error(), "Only admin or manager"))
}

func TestUpdateSettings_InvalidTaxRate(t *testing.T) {
	repos := &repository.Repositories{Settings: newFake()}
	svc := NewService(repos)

	_, err := svc.UpdateSettings(claimsCtx("admin"), UpdateParams{TaxRate: 150})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcCode(err))
	assert.True(t, strings.Contains(err.Error(), "between 0 and 100"))
}

func TestUpdateSettings_AdminUpsertsAndReadsBack(t *testing.T) {
	fake := newFake()
	repos := &repository.Repositories{Settings: fake}
	svc := NewService(repos)

	_, err := svc.UpdateSettings(claimsCtx("admin"), UpdateParams{
		TaxRate:       0,
		ReceiptHeader: "TERIMA KASIH",
		ReceiptFooter: "Barang yang sudah dibeli tidak dapat ditukar",
		ShowStoreName: false,
		ShowTax:       false,
		ShowCashier:   true,
	})
	assert.NoError(t, err)

	s, err := svc.GetSettings(claimsCtx("admin"))
	assert.NoError(t, err)
	assert.Equal(t, 0.0, s.TaxRate)
	assert.Equal(t, "TERIMA KASIH", s.ReceiptHeader)
	assert.False(t, s.ShowStoreName)
	assert.False(t, s.ShowTax)

	// Second save = update path (row count stays at one).
	_, err = svc.UpdateSettings(claimsCtx("admin"), UpdateParams{TaxRate: 7.5})
	assert.NoError(t, err)
	assert.Len(t, fake.rows, 1)
	s, err = svc.GetSettings(claimsCtx("admin"))
	assert.NoError(t, err)
	assert.Equal(t, 7.5, s.TaxRate)
}

func TestUpdateSettings_ManagerAllowed(t *testing.T) {
	repos := &repository.Repositories{Settings: newFake()}
	svc := NewService(repos)

	_, err := svc.UpdateSettings(claimsCtx("manager"), UpdateParams{TaxRate: 5})
	assert.NoError(t, err)
}
