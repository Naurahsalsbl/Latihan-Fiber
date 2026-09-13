package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"latihan-fiber/modul4/app/service"
	"latihan-fiber/modul4/helper"
	"latihan-fiber/modul4/middleware"
)

// Dependencies mengumpulkan seluruh service jadi satu struct, supaya
// penambahan service berikutnya tidak mengubah tanda tangan Register.
type Dependencies struct {
	Pool           *pgxpool.Pool
	JWT            *helper.JWTManager
	StudentService *service.StudentService
	AuthService    *service.AuthService
}

// Register memetakan URL ke method pada service.
//
// Pembagian di bawah ini dapat dibaca sekilas: endpoint mana yang publik,
// mana yang untuk masuk, dan mana yang terkunci di belakang token.
func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	// --- publik, tidak perlu token ---
	api.Get("/health", healthCheck(deps.Pool))

	// --- autentikasi ---
	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", deps.AuthService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", deps.AuthService.Refresh)
	auth.Post("/logout", deps.AuthService.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	// --- wajib membawa access token yang sah ---
	s := api.Group("/students", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	s.Get("/", deps.StudentService.List)
	s.Get("/:id", deps.StudentService.Get)
	s.Post("/", deps.StudentService.Create)
	s.Put("/:id", deps.StudentService.Replace)
	s.Patch("/:id", deps.StudentService.Patch)
	s.Delete("/:id", deps.StudentService.Delete)
}

// healthCheck melaporkan kondisi layanan beserta databasenya. Tetap publik
// karena dipakai load balancer/monitoring yang tidak punya access token.
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