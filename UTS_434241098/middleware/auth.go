package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"UTS_434241098/helper"
)

func Auth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	if authHeader == "" {
		return helper.Error(
			c,
			fiber.StatusUnauthorized,
			"Token tidak ditemukan",
			nil,
		)
	}

	parts := strings.SplitN(authHeader, " ", 2)

	if len(parts) != 2 || parts[0] != "Bearer" {
		return helper.Error(
			c,
			fiber.StatusUnauthorized,
			"Format token tidak valid",
			nil,
		)
	}

	claims, err := helper.ParseToken(parts[1])
	if err != nil {
		return helper.Error(
			c,
			fiber.StatusUnauthorized,
			"Token tidak valid atau sudah kedaluwarsa",
			nil,
		)
	}

	c.Locals("user_id", claims.UserID)
	c.Locals("email", claims.Email)
	c.Locals("role", claims.Role)

	return c.Next()
}

func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)

		if !ok {
			return helper.Error(
				c,
				fiber.StatusUnauthorized,
				"User belum terautentikasi",
				nil,
			)
		}

		for _, allowedRole := range roles {
			if role == allowedRole {
				return c.Next()
			}
		}

		return helper.Error(
			c,
			fiber.StatusForbidden,
			"Akses ditolak",
			nil,
		)
	}
}