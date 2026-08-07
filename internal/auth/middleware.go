package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"

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

// SupabaseClaims represents standard claims embedded in Supabase Auth JWTs.
type SupabaseClaims struct {
	jwt.RegisteredClaims
	Email       string                 `json:"email"`
	AppMetadata map[string]interface{} `json:"app_metadata"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
	Role        string                 `json:"role"`
}

// RequireAuth returns a Fiber middleware that validates Supabase JWT from Authorization header.
func RequireAuth(jwtSecret ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		secret := ""
		if len(jwtSecret) > 0 && jwtSecret[0] != "" {
			secret = jwtSecret[0]
		} else {
			secret = os.Getenv("SUPABASE_JWT_SECRET")
		}

		if secret == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "Server configuration error: SUPABASE_JWT_SECRET is missing",
				},
			})
		}

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
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
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

		// Save user information into Fiber context locals
		c.Locals(LocalUserIDKey, userID)
		c.Locals(LocalEmailKey, claims.Email)
		c.Locals(LocalUserRoleKey, claims.Role)

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
