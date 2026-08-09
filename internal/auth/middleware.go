package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const (
	LocalUserIDKey   = "user_id"
	LocalEmailKey    = "email"
	LocalUserRoleKey = "role"
)

var (
	ErrMissingAuthHeader = errors.New("missing authorization header")
	ErrInvalidAuthFormat = errors.New("invalid authorization header format")
	ErrInvalidToken      = errors.New("invalid or expired token")
	ErrUserNotContext    = errors.New("user_id not found in context")
)

// RoleQuerier is a minimal interface for fetching user role from the database.
// This avoids circular imports between auth and repository packages.
type RoleQuerier interface {
	GetUserRoleByID(ctx context.Context, id string) (string, error)
}

type JWKSKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type JWKSResponse struct {
	Keys []JWKSKey `json:"keys"`
}

var (
	jwksCache     = make(map[string]*ecdsa.PublicKey)
	jwksMutex     sync.RWMutex
	lastJWKSFetch time.Time
)

func getECPublicKey(kid string) (*ecdsa.PublicKey, error) {
	jwksMutex.RLock()
	pubKey, exists := jwksCache[kid]
	fetchNeeded := !exists || time.Since(lastJWKSFetch) > 1*time.Hour
	jwksMutex.RUnlock()

	if !fetchNeeded && pubKey != nil {
		return pubKey, nil
	}

	jwksMutex.Lock()
	defer jwksMutex.Unlock()

	if pubKey, exists := jwksCache[kid]; exists && time.Since(lastJWKSFetch) <= 1*time.Hour {
		return pubKey, nil
	}

	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		return nil, fmt.Errorf("SUPABASE_URL environment variable is not set")
	}
	jwksURL := fmt.Sprintf("%s/auth/v1/.well-known/jwks.json", strings.TrimSuffix(supabaseURL, "/"))

	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	for _, key := range jwks.Keys {
		if key.Kty == "EC" && key.X != "" && key.Y != "" {
			xBytes, err := base64.RawURLEncoding.DecodeString(key.X)
			if err != nil {
				continue
			}
			yBytes, err := base64.RawURLEncoding.DecodeString(key.Y)
			if err != nil {
				continue
			}
			parsedKey := &ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     new(big.Int).SetBytes(xBytes),
				Y:     new(big.Int).SetBytes(yBytes),
			}
			jwksCache[key.Kid] = parsedKey
		}
	}
	lastJWKSFetch = time.Now()

	if targetKey, found := jwksCache[kid]; found {
		return targetKey, nil
	}
	for _, k := range jwksCache {
		return k, nil
	}

	return nil, fmt.Errorf("public key not found for kid %s", kid)
}

// SupabaseClaims represents standard claims embedded in Supabase Auth JWTs.
type SupabaseClaims struct {
	jwt.RegisteredClaims
	Email        string                 `json:"email"`
	AppMetadata  map[string]interface{} `json:"app_metadata"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
	Role         string                 `json:"role"`
}

// RequireAuth returns a Fiber middleware that validates Supabase JWT from Authorization header.
// If a RoleQuerier is provided, the user's role is fetched from the database (source of truth).
// Otherwise, the JWT claim's top-level role field is used as fallback.
func RequireAuth(roleQuerier ...RoleQuerier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		secret := os.Getenv("SUPABASE_JWT_SECRET")

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "Unauthorized: Missing Authorization header",
				},
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "Unauthorized: Invalid Authorization header format",
				},
			})
		}

		tokenString := parts[1]
		claims := &SupabaseClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			switch token.Method.(type) {
			case *jwt.SigningMethodHMAC:
				if secret == "" {
					return nil, errors.New("SUPABASE_JWT_SECRET is missing for HS256 verification")
				}
				return []byte(secret), nil
			case *jwt.SigningMethodECDSA:
				kid, _ := token.Header["kid"].(string)
				return getECPublicKey(kid)
			default:
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
		})

		if err != nil || !token.Valid {
			fmt.Printf("⚠️ JWT Auth Error: err=%v, token.Valid=%v\n", err, token != nil && token.Valid)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "Unauthorized: Invalid or expired token",
				},
			})
		}

		userID := claims.Subject
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "Unauthorized: Invalid token subject",
				},
			})
		}

		// Determine role: prefer DB lookup (source of truth) over JWT claim
		role := "user"
		if len(roleQuerier) > 0 && roleQuerier[0] != nil {
			dbRole, err := roleQuerier[0].GetUserRoleByID(c.Context(), userID)
			if err == nil && dbRole != "" {
				role = dbRole
			}
		}

		// Fallback check against ADMIN_EMAILS env var for claims.Email
		if role != "admin" && claims.Email != "" {
			adminEmailsEnv := os.Getenv("ADMIN_EMAILS")
			if adminEmailsEnv != "" {
				for _, email := range strings.Split(adminEmailsEnv, ",") {
					if strings.EqualFold(strings.TrimSpace(email), strings.TrimSpace(claims.Email)) {
						role = "admin"
						break
					}
				}
			}
		}

		// Save user information into Fiber context locals
		c.Locals(LocalUserIDKey, userID)
		c.Locals(LocalEmailKey, claims.Email)
		c.Locals(LocalUserRoleKey, role)

		return c.Next()
	}
}

// RequireAdmin returns a Fiber middleware ensuring the authenticated user has admin privileges.
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals(LocalUserRoleKey).(string)
		if !ok || role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "Forbidden: Admin access required",
				},
			})
		}
		return c.Next()
	}
}

// GetUserID retrieves the authenticated user's ID from Fiber context.
func GetUserID(c *fiber.Ctx) (string, error) {
	userID, ok := c.Locals(LocalUserIDKey).(string)
	if !ok || userID == "" {
		return "", ErrUserNotContext
	}
	return userID, nil
}

// GetUserEmail retrieves the authenticated user's email from Fiber context.
func GetUserEmail(c *fiber.Ctx) string {
	email, ok := c.Locals(LocalEmailKey).(string)
	if !ok {
		return ""
	}
	return email
}
