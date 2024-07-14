package apis

import (
	"github.com/gofiber/fiber/v2"
)

func APIKeyAuth(allowedKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("x-api-key")

		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "API Key is required",
			})
		}

		validKey := false
		if apiKey == allowedKey {
			validKey = true
		}

		if !validKey {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid API Key",
			})
		}

		return c.Next()
	}
}
