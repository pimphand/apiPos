package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	gormdb "apiPos/internal/database/gorm"
)

// RefreshTokenRepository defines operations for device-bound refresh tokens.
type RefreshTokenRepository interface {
	// Create persists a new refresh token row.
	Create(ctx context.Context, tok *gormdb.RefreshToken) error
	// GetActiveByTokenHash returns a non-revoked, non-expired token whose
	// stored hash and device id match. A device mismatch is treated as no match
	// so a token copied to another device is rejected.
	GetActiveByTokenHash(ctx context.Context, tokenHash, deviceID string) (*gormdb.RefreshToken, error)
	// Revoke marks a single token as revoked by its ID.
	Revoke(ctx context.Context, id string) error
	// RevokeByTokenHash revokes tokens matching a token hash, optionally
	// scoped to a device. Used by logout.
	RevokeByTokenHash(ctx context.Context, tokenHash, deviceID string) error
	// RevokeByUser revokes all non-revoked tokens for a user, optionally scoped
	// to a single device.
	RevokeByUser(ctx context.Context, userID, deviceID string) error
}

// RefreshTokenRepo implements RefreshTokenRepository using GORM.
type RefreshTokenRepo struct {
	*GormBase
}

// NewRefreshTokenRepository creates a new RefreshTokenRepo.
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &RefreshTokenRepo{GormBase: NewGormBase(db)}
}

// Create persists a new refresh token row.
func (r *RefreshTokenRepo) Create(ctx context.Context, tok *gormdb.RefreshToken) error {
	return r.WithContext(ctx).Create(tok).Error
}

// GetActiveByTokenHash returns the matching non-revoked, non-expired token for
// a given hash + device id.
func (r *RefreshTokenRepo) GetActiveByTokenHash(ctx context.Context, tokenHash, deviceID string) (*gormdb.RefreshToken, error) {
	var tok gormdb.RefreshToken
	err := r.WithContext(ctx).
		Where("token = ? AND device_id = ? AND is_revoked = ? AND expires_at > ?",
			tokenHash, deviceID, false, time.Now()).
		First(&tok).Error
	if err != nil {
		return nil, err
	}
	return &tok, nil
}

// Revoke marks a single token as revoked by its ID.
func (r *RefreshTokenRepo) Revoke(ctx context.Context, id string) error {
	return r.WithContext(ctx).
		Model(&gormdb.RefreshToken{}).
		Where("id = ?", id).
		Update("is_revoked", true).Error
}

// RevokeByTokenHash revokes tokens matching a token hash (optionally per device).
func (r *RefreshTokenRepo) RevokeByTokenHash(ctx context.Context, tokenHash, deviceID string) error {
	q := r.WithContext(ctx).
		Model(&gormdb.RefreshToken{}).
		Where("token = ? AND is_revoked = ?", tokenHash, false)
	if deviceID != "" {
		q = q.Where("device_id = ?", deviceID)
	}
	return q.Update("is_revoked", true).Error
}

// RevokeByUser revokes all non-revoked tokens for a user (optionally per device).
func (r *RefreshTokenRepo) RevokeByUser(ctx context.Context, userID, deviceID string) error {
	q := r.WithContext(ctx).
		Model(&gormdb.RefreshToken{}).
		Where("user_id = ? AND is_revoked = ?", userID, false)
	if deviceID != "" {
		q = q.Where("device_id = ?", deviceID)
	}
	return q.Update("is_revoked", true).Error
}
