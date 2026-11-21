package integration

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"apiPos/internal/accounting"
	"apiPos/internal/apperror"
	"apiPos/internal/auth"
	gormdb "apiPos/internal/database/gorm"
	"apiPos/internal/payment"
	"apiPos/internal/permission"
	"apiPos/internal/repository"
	"apiPos/internal/role"
	"apiPos/internal/selforder"
	"apiPos/internal/user"
)

func TestStorePrivilegesAndSelfOrderConfirmation(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is not set")
	}
	db := newTestDB(t, dsn)
	repos := repository.New(db)
	tenantRole := gormdb.Role{Name: "admin"}
	systemRole := gormdb.Role{Name: "superadmin"}
	require.NoError(t, db.Create(&tenantRole).Error)
	require.NoError(t, db.Create(&systemRole).Error)
	tenant := seedTenant(t, db, tenantRole, "Security", 125)
	ctx := auth.ContextWithClaims(t.Context(), &auth.Claims{
		UserID: tenant.user.ID.String(), RoleName: tenantRole.Name,
		RoleID: tenantRole.ID.String(), StoreID: tenant.store.ID.String(),
	})

	t.Run("store admin cannot grant system privileges", func(t *testing.T) {
		svc := user.NewService(repos, nil)
		_, err := svc.UpdateUserRole(ctx, tenant.user.ID.String(), systemRole.ID.String(), tenant.store.ID.String())
		require.ErrorIs(t, err, apperror.ErrForbidden)
		_, err = svc.UpdateUser(ctx, tenant.user.ID.String(), "", "", systemRole.ID.String(), tenant.store.ID.String(), true)
		require.ErrorIs(t, err, apperror.ErrForbidden)
		_, err = role.NewService(repos).UpdateRole(ctx, tenantRole.ID.String(), "", "role description")
		require.ErrorIs(t, err, apperror.ErrForbidden)
		_, err = permission.NewService(repos).CreatePermission(ctx, "reports.read", "reports", "read", "")
		require.ErrorIs(t, err, apperror.ErrForbidden)
	})

	t.Run("staff payment errors keep their existing contract", func(t *testing.T) {
		payments := payment.NewService(repos)
		for _, tc := range []struct {
			status  string
			message string
		}{
			{"PAID", "Sale is already paid"},
			{"CANCELLED", "Cannot add payment to cancelled/refunded sale"},
			{"REFUNDED", "Cannot add payment to cancelled/refunded sale"},
			{"PENDING", "Payment amount (130.00) exceeds remaining (125.00)"},
		} {
			t.Run(tc.status, func(t *testing.T) {
				order := gormdb.Sale{StoreID: tenant.store.ID, InvoiceNumber: "INV-CONTRACT-" + tc.status,
					Subtotal: 125, GrandTotal: 125, Status: tc.status}
				require.NoError(t, db.Omit("CashierID", "CustomerID").Create(&order).Error)
				_, err := payments.CreatePayment(ctx, payment.CreatePaymentRequest{
					SaleID: order.ID.String(), PaymentMethod: payment.PaymentMethodCard, Amount: 130,
				})
				require.Error(t, err)
				status := apperror.MapError(err).GRPCStatus()
				require.Equal(t, codes.InvalidArgument, status.Code())
				require.Equal(t, tc.message, status.Message())
			})
		}
	})

	t.Run("customer cannot confirm payment", func(t *testing.T) {
		require.NoError(t, db.AutoMigrate(&gormdb.ChartOfAccount{}, &gormdb.AccountingPeriod{}, &gormdb.JournalEntry{}, &gormdb.JournalLine{}))
		require.NoError(t, accounting.EnsureCOAForStore(t.Context(), db, tenant.store.ID))
		orderToken := uuid.New()
		order := gormdb.Sale{StoreID: tenant.store.ID, InvoiceNumber: "INV-SECURITY-ORDER",
			Subtotal: 125, GrandTotal: 125, Status: "PENDING", OrderSource: "SELF_ORDER", OrderToken: &orderToken}
		require.NoError(t, db.Omit("CashierID", "CustomerID").Create(&order).Error)
		orders := selforder.NewService(repos)
		_, err := orders.PayPublicOrder(t.Context(), orderToken.String(), payment.PaymentMethodBankTransfer)
		require.Equal(t, codes.FailedPrecondition, apperror.MapError(err).GRPCStatus().Code())
		state, err := orders.GetOrderStatus(t.Context(), orderToken.String())
		require.NoError(t, err)
		require.Equal(t, "PENDING", state.Status)

		payments := payment.NewService(repos)
		req := payment.CreatePaymentRequest{SaleID: order.ID.String(), PaymentMethod: payment.PaymentMethodBankTransfer, Amount: 125}
		_, err = payments.CreatePayment(t.Context(), req)
		require.ErrorIs(t, err, apperror.ErrUnauthorized)
		outsideStore := auth.ContextWithClaims(t.Context(), &auth.Claims{RoleName: "cashier", StoreID: uuid.NewString()})
		_, err = payments.CreatePayment(outsideStore, req)
		requireDenied(t, err)

		p, err := payments.CreatePayment(ctx, req)
		require.NoError(t, err)
		require.Equal(t, tenant.store.ID, p.StoreID)
		_, err = payments.CreatePayment(ctx, req)
		require.Error(t, err)
		status := apperror.MapError(err).GRPCStatus()
		require.Equal(t, codes.InvalidArgument, status.Code())
		require.Equal(t, "Sale is already paid", status.Message())
		confirmed, err := orders.PayPublicOrder(t.Context(), orderToken.String(), payment.PaymentMethodBankTransfer)
		require.NoError(t, err)
		require.Equal(t, order.ID, confirmed.ID)
		require.Equal(t, "PAID", confirmed.Status)
		state, err = orders.GetOrderStatus(t.Context(), orderToken.String())
		require.NoError(t, err)
		require.Equal(t, "PAID", state.Status)
		records, err := payments.GetPaymentsBySale(ctx, order.ID.String())
		require.NoError(t, err)
		require.Len(t, records, 1)
	})
}
