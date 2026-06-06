// Package auth implements authentication primitives: password
// hashing, JWT issue/verify, refresh-token sessions, 2FA TOTP
// (verification only in v1.0; full setup endpoint deferred to v1.1
// pending stable ACME/TOTP library).
//
// Token model:
//   - Access token: HS256 JWT, 15-min TTL, claims = uid/email/role/tid/sid
//   - Refresh token: 32-byte random, base64url, 30-day TTL, single-use
//                    rotation. Stored as SHA-256 hash in user_sessions.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/orvixpanel/orvixpanel/internal/config"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// -----------------------------------------------------------------------------
// Errors
// -----------------------------------------------------------------------------

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserSuspended      = errors.New("user suspended")
	ErrUserLocked         = errors.New("user locked")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("expired token")
	ErrSessionRevoked     = errors.New("session revoked")
	ErrWeakPassword       = errors.New("password too weak")
)

// -----------------------------------------------------------------------------
// Token + claims
// -----------------------------------------------------------------------------

// TokenPair is what Login / Refresh return.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// Claims is the JWT payload.
type Claims struct {
	UserID    string `json:"uid"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TenantID  string `json:"tid"`
	AccountID string `json:"aid,omitempty"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// -----------------------------------------------------------------------------
// Service
// -----------------------------------------------------------------------------

// Service is the entry point used by the API layer.
type Service struct {
	db         *gorm.DB
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	bcryptCost int
}

// New constructs a Service.
func New(db *gorm.DB, cfg *config.Config) (*Service, error) {
	if len(cfg.Server.SecretKey) < 32 {
		return nil, fmt.Errorf("server.secret_key must be at least 32 chars; got %d", len(cfg.Server.SecretKey))
	}
	cost := cfg.Auth.BcryptCost
	if cost <= 0 {
		cost = 12
	}
	return &Service{
		db:         db,
		secret:     []byte(cfg.Server.SecretKey),
		issuer:     "orvixpanel",
		accessTTL:  cfg.Auth.AccessTokenTTL,
		refreshTTL: cfg.Auth.RefreshTokenTTL,
		bcryptCost: cost,
	}, nil
}

// HashPassword bcrypts the given plaintext.
func (s *Service) HashPassword(plain string) (string, error) {
	if err := ValidatePassword(plain); err != nil {
		return "", err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), s.bcryptCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}
	return string(h), nil
}

// VerifyPassword constant-time compares the plaintext against the hash.
func (s *Service) VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// ValidatePassword enforces a minimum policy: 10 chars, must contain
// at least one letter and one digit.
func ValidatePassword(p string) error {
	if len(p) < 10 {
		return fmt.Errorf("%w: must be at least 10 characters", ErrWeakPassword)
	}
	hasLetter, hasDigit := false, false
	for _, r := range p {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			hasLetter = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("%w: must contain both letters and digits", ErrWeakPassword)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Login
// -----------------------------------------------------------------------------

// LoginResult carries the outcome of a Login.
type LoginResult struct {
	Tokens *TokenPair
	User   *models.User
}

// Login verifies email + password and issues tokens. TOTP 2FA in v1.0
// is a no-op stub (the schema is there; the verify path is added in
// v1.1 after the lego+otp library upgrade clears).
func (s *Service) Login(ctx context.Context, email, password, ip string) (*LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var user models.User
	if err := s.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Equalize timing on missing user.
			_ = bcrypt.CompareHashAndPassword(
				[]byte("$2a$12$dummy.hash.for.timing.equalization.only.xxxxxxxx"),
				[]byte(password),
			)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("lookup user: %w", err)
	}

	if user.Status != "active" {
		return nil, ErrUserSuspended
	}
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, ErrUserLocked
	}

	if !s.VerifyPassword(user.PasswordHash, password) {
		user.FailedLogins++
		if user.FailedLogins >= 5 {
			until := time.Now().Add(15 * time.Minute)
			user.LockedUntil = &until
		}
		_ = s.db.WithContext(ctx).Save(&user).Error
		return nil, ErrInvalidCredentials
	}

	// Success — reset counters.
	now := time.Now().UTC()
	user.FailedLogins = 0
	user.LockedUntil = nil
	user.LastLoginAt = &now
	user.LastLoginIP = ip
	_ = s.db.WithContext(ctx).Save(&user).Error

	pair, err := s.issueTokens(ctx, &user, ip, "")
	if err != nil {
		return nil, err
	}
	return &LoginResult{Tokens: pair, User: &user}, nil
}

// -----------------------------------------------------------------------------
// Refresh + Logout
// -----------------------------------------------------------------------------

// Refresh exchanges a refresh token for a new pair (rotation).
func (s *Service) Refresh(ctx context.Context, refreshToken, ip string) (*TokenPair, error) {
	if refreshToken == "" {
		return nil, ErrInvalidToken
	}
	hash := hashRefresh(refreshToken)

	var session models.UserSession
	if err := s.db.WithContext(ctx).
		Where("refresh_hash = ?", hash).
		First(&session).Error; err != nil {
		return nil, ErrInvalidToken
	}
	if !session.IsActive() {
		return nil, ErrSessionRevoked
	}

	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", session.UserID).Error; err != nil {
		return nil, ErrUserNotFound
	}
	if user.Status != "active" {
		return nil, ErrUserSuspended
	}

	// Rotate: revoke old, issue new.
	now := time.Now().UTC()
	session.RevokedAt = &now
	session.RevokeReason = "rotated"
	if err := s.db.WithContext(ctx).Save(&session).Error; err != nil {
		return nil, fmt.Errorf("revoke old session: %w", err)
	}
	return s.issueTokens(ctx, &user, ip, "")
}

// Logout revokes the session tied to the given claims.
func (s *Service) Logout(ctx context.Context, claims *Claims) error {
	if claims == nil || claims.SessionID == "" {
		return nil
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).
		Model(&models.UserSession{}).
		Where("session_id = ?", claims.SessionID).
		Updates(map[string]any{
			"revoked_at":    &now,
			"revoke_reason": "logout",
		}).Error
}

// -----------------------------------------------------------------------------
// JWT verify (used by middleware)
// -----------------------------------------------------------------------------

// Verify parses + validates an access token. Session-active check is
// the middleware's job (it needs ctx).
func (s *Service) Verify(tokenStr string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithLeeway(30*time.Second))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// VerifyAndCheckSession combines Verify with the session-active DB
// check. v1.0 always hits the DB; v1.1 adds a Redis cache.
func (s *Service) VerifyAndCheckSession(ctx context.Context, tokenStr string) (*Claims, error) {
	claims, err := s.Verify(tokenStr)
	if err != nil {
		return nil, err
	}
	var session models.UserSession
	if err := s.db.WithContext(ctx).
		Select("id, revoked_at, expires_at").
		Where("session_id = ?", claims.SessionID).
		First(&session).Error; err != nil {
		return nil, ErrSessionRevoked
	}
	if !session.IsActive() {
		return nil, ErrSessionRevoked
	}
	return claims, nil
}

// -----------------------------------------------------------------------------
// Internal — token issuance + helpers
// -----------------------------------------------------------------------------

func (s *Service) issueTokens(ctx context.Context, user *models.User, ip, userAgent string) (*TokenPair, error) {
	sessionID := uuid.NewString()

	refresh, err := newRefreshToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(s.refreshTTL)
	session := models.UserSession{
		Base:        models.Base{ID: newID()},
		UserID:      user.ID,
		SessionID:   sessionID,
		RefreshHash: hashRefresh(refresh),
		UserAgent:   userAgent,
		IP:          ip,
		ExpiresAt:   expiresAt,
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	access, exp, err := s.signAccessToken(user, sessionID)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    exp,
	}, nil
}

func (s *Service) signAccessToken(user *models.User, sessionID string) (string, int64, error) {
	exp := time.Now().Add(s.accessTTL)
	claims := &Claims{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		TenantID:  user.TenantID,
		AccountID: user.AccountID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(s.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign jwt: %w", err)
	}
	return signed, exp.Unix(), nil
}

func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashRefresh returns the lowercase hex SHA-256 of the refresh token.
// The raw token is never stored.
func hashRefresh(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
