package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func JWT(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Missing Authorization header")
		}

		// Extract token from "Bearer <token>"
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid Authorization header format")
		}

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
		}

		if !token.Valid {
			return fiber.NewError(fiber.StatusUnauthorized, "Token is not valid")
		}

		// Extract claims
		claims, ok := token.Claims.(*UserClaims)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid token claims")
		}

		// Store user claims in context
		c.Locals("user", claims)
		return c.Next()
	}
}

func GetUserFromContext(c *fiber.Ctx) *UserClaims {
	user := c.Locals("user")
	if user == nil {
		return nil
	}

	claims, ok := user.(*UserClaims)
	if !ok {
		return nil
	}

	return claims
}

// extractTokenFromHeader extracts the JWT token from Authorization header
func extractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fiber.NewError(fiber.StatusUnauthorized, "Missing Authorization header")
	}

	// Check if it starts with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fiber.NewError(fiber.StatusUnauthorized, "Invalid Authorization header format")
	}

	// Extract token part after "Bearer "
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		return "", fiber.NewError(fiber.StatusUnauthorized, "Empty token")
	}

	return token, nil
}

// validateJWT validates a JWT token with the given secret
func validateJWT(tokenString, secret string) error {
	if tokenString == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Empty token")
	}

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Check signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid token: "+err.Error())
	}

	if !token.Valid {
		return fiber.NewError(fiber.StatusUnauthorized, "Token is not valid")
	}

	return nil
}
