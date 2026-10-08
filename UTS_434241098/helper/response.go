package helper

import "github.com/gofiber/fiber/v2"

func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(fiber.Map{
		"success": true,
		"message": message,
		"data":    data,
	})
}

func Error(c *fiber.Ctx, status int, message string, errs any) error {
	response := fiber.Map{
		"success": false,
		"message": message,
	}

	if errs != nil {
		response["errors"] = errs
	}

	return c.Status(status).JSON(response)
}