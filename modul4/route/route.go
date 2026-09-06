package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"latihan-fiber/modul4/app/service"
	"latihan-fiber/modul4/helper"
)

// Register memetakan URL ke method pada service.
//
// Perhatikan isi file ini: tidak ada logika bisnis, tidak ada query,
// tidak ada validasi. Hanya daftar alamat dan siapa yang melayaninya.
func Register(app *fiber.App, pool *pgxpool.Pool, studentService *service.StudentService) {
	api := app.Group("/api/v1")

	api.Get("/health", healthCheck(pool))

	s := api.Group("/students")
	s.Get("/", studentService.List)
	s.Get("/:id", studentService.Get)
	s.Post("/", studentService.Create)
	s.Put("/:id", studentService.Replace)
	s.Patch("/:id", studentService.Patch)
	s.Delete("/:id", studentService.Delete)
}

// healthCheck melaporkan kondisi layanan beserta databasenya.
// Sebelumnya berada inline di main.go; dipindah ke sini agar main.go
// hanya berisi urutan perakitan (lihat Modul 4 Langkah 6-7).
func healthCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.Fail(c, fiber.StatusServiceUnavailable, "database tidak dapat dihubungi")
		}
		return helper.Success(c, fiber.StatusOK, "server dan database berjalan", nil)
	}
}