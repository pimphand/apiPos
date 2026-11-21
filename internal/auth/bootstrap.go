package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	gormdb "apiPos/internal/database/gorm"
)

// BootstrapSuperadmin provisions an explicitly configured account without resetting existing credentials.
func BootstrapSuperadmin(ctx context.Context, db *gorm.DB, username, email, password string) error {
	if username == "" && email == "" && password == "" {
		return nil
	}
	address, err := mail.ParseAddress(email)
	if username == "" || username != strings.TrimSpace(username) || len(username) > 50 ||
		strings.EqualFold(username, "superadmin") || err != nil || address.Address != email ||
		len(email) > 100 || len(password) < 24 || len(password) > 72 {
		return errors.New("set a new username, a valid email, and a unique random password of 24–72 bytes")
	}

	db = db.WithContext(ctx)
	var role gormdb.Role
	if err := db.Where("name = ?", "superadmin").First(&role).Error; err != nil {
		return errors.New("superadmin role is unavailable; apply database migrations")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}
	user := gormdb.User{
		Username: username, Email: email, PasswordHash: string(hash), FullName: "Super Admin",
		RoleID: role.ID, StoreID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), IsActive: true,
	}
	result := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "username"}}, DoNothing: true}).Create(&user)
	if result.Error != nil {
		return errors.New("could not provision superadmin; check that the email is unique")
	}
	if result.RowsAffected == 0 {
		var existing gormdb.User
		if err := db.Where("username = ?", username).First(&existing).Error; err != nil ||
			existing.RoleID != role.ID || !existing.IsActive || existing.Email != email {
			return errors.New("bootstrap username is already in use")
		}
	}
	return nil
}
