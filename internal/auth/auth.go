package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"apiPos/internal/apperror"
	"apiPos/internal/config"
	gormdb "apiPos/internal/database/gorm"
)

// Claims represents JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	RoleID   string `json:"role_id"`
	RoleName string `json:"role_name"`
	StoreID  string `json:"store_id"`
	jwt.RegisteredClaims
}

// TokenPair contains access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// UserLookup returns current account data for access-token validation.
type UserLookup interface {
	GetByID(context.Context, string) (*gormdb.User, error)
}

// AuthService handles authentication operations.
type AuthService struct {
	config  *config.Config
	lockout *LoginLockout
	users   UserLookup
}

// NewAuthService creates a new AuthService
func NewAuthService(cfg *config.Config, users UserLookup) *AuthService {
	return &AuthService{config: cfg, lockout: NewLoginLockout(), users: users}
}

// CheckLoginLockout reports apperror.ErrTooManyAttempts while the username is
// locked out after repeated failed logins.
func (s *AuthService) CheckLoginLockout(username string) error {
	return s.lockout.Check(username)
}

// RecordLoginFailure counts one failed login attempt for the username.
func (s *AuthService) RecordLoginFailure(username string) {
	s.lockout.RecordFailure(username)
}

// ResetLoginFailures clears the failure counter after a successful login.
func (s *AuthService) ResetLoginFailures(username string) {
	s.lockout.Reset(username)
}

// HashPassword hashes a password using bcrypt
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword compares a hashed password with a plain password
func (s *AuthService) CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// GenerateAccessToken issues a JWT access token and its expiry time.
func (s *AuthService) GenerateAccessToken(userID, username, email, roleID, roleName, storeID string) (string, time.Time, error) {
	return s.generateAccessToken(userID, username, email, roleID, roleName, storeID)
}

// GenerateRefreshToken generates a new opaque refresh token together with its
// SHA-256 hash. Only the hash is persisted; the raw token is returned to the
// client and must never be stored server-side.
func (s *AuthService) GenerateRefreshToken() (raw, hash string, expiresAt time.Time, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", time.Time{}, err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	expiresAt = time.Now().Add(s.config.JWT.RefreshTTL)
	return raw, s.HashToken(raw), expiresAt, nil
}

// HashToken returns the hex-encoded SHA-256 digest of a raw token.
func (s *AuthService) HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *AuthService) generateAccessToken(userID, username, email, roleID, roleName, storeID string) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.config.JWT.AccessTTL)

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RoleID:   roleID,
		RoleName: roleName,
		StoreID:  storeID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "pos-app",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// ValidateToken validates a JWT access token and returns claims
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWT.Secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, apperror.ErrTokenExpired
		}
		return nil, apperror.ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, apperror.ErrInvalidToken
}

// Context key for claims
type contextKey string

const ClaimsContextKey contextKey = "claims"

// ContextWithClaims adds claims to context
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ClaimsContextKey, claims)
}

// ClaimsFromContext extracts claims from context
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	return claims, ok
}

// Authenticate checks that the account and authorization in a signed token are still current.
func (s *AuthService) Authenticate(ctx context.Context, token string) (*Claims, error) {
	claims, err := s.ValidateToken(token)
	if err != nil {
		return nil, err
	}
	user, err := s.users.GetByID(ctx, claims.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.ErrInvalidToken
	}
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, "Failed to validate account", err)
	}
	if !user.IsActive || user.RoleID.String() != claims.RoleID ||
		user.Role.Name != claims.RoleName || user.StoreID.String() != claims.StoreID {
		return nil, apperror.ErrInvalidToken
	}
	return claims, nil
}

func (s *AuthService) ClaimsFromRequest(r *http.Request) (*Claims, error) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return nil, apperror.ErrInvalidToken
	}
	return s.Authenticate(r.Context(), token)
}

func RequireSuperadmin(ctx context.Context) error {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return apperror.ErrUnauthorized
	}
	if claims.RoleName != "superadmin" {
		return apperror.ErrForbidden
	}
	return nil
}
