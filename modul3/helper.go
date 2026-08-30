package main

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"latihan-fiber/modul3/app/model"
)

func reqCtx(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

func ok(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

func okWithMeta(c *fiber.Ctx, message string, data any, meta model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Status:  "success",
		Message: message,
		Data:    data,
		Meta:    &meta,
	})
}

func fail(c *fiber.Ctx, code int, message string) error {
	return c.Status(code).JSON(model.WebResponse{
		Status:  "fail",
		Message: message,
	})
}