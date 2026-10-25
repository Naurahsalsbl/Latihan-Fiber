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
	Permissions    *helper.PermissionSet
	StudentService *service.StudentService
	AuthService    *service.AuthService
	CourseService  *service.CourseService
	SupervisionService *service.SupervisionService
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
		perms := deps.Permissions

		// Hak dapat diputuskan tanpa melihat data -> middleware.
		s.Get("/",
			middleware.RequirePermission(perms, "student:list"),
			deps.StudentService.List)
		s.Post("/",
			middleware.RequirePermission(perms, "student:create"),
			deps.StudentService.Create)
		s.Delete("/:id",
			middleware.RequirePermission(perms, "student:delete"),
			deps.StudentService.Delete)

		// Hak bergantung pada kepemilikan data -> diperiksa di service.
		s.Get("/:id", deps.StudentService.Get)
		s.Put("/:id", deps.StudentService.Replace)
		s.Patch("/:id", deps.StudentService.Patch)

		
		// --- kelas dan KRS ---
		courses := api.Group("/courses",
			middleware.RequireJSON,
			middleware.RequireAuth(deps.JWT),
		)

		// /mine harus didaftarkan sebelum /:id
		courses.Get("/", deps.CourseService.List)
		courses.Get("/mine", deps.CourseService.Mine)
		courses.Get("/:id", deps.CourseService.Get)
		courses.Post("/:id/enroll", deps.CourseService.Enroll)
		courses.Delete("/:id/enroll", deps.CourseService.Drop)
		courses.Patch("/:id", middleware.RequirePermission(perms, "course:manage"),
	deps.CourseService.UpdateQuota)

		enrollments := api.Group("/enrollments", 
			middleware.RequireJSON, 
			middleware.RequireAuth(deps.JWT))
		enrollments.Post("/swap", deps.CourseService.Swap)

		// Pengelolaan dosen pembimbing hanya untuk pengguna
		// yang memiliki permission supervision:manage.
		supervisors := api.Group(
			"/supervisors",
			middleware.RequireJSON,
			middleware.RequireAuth(deps.JWT),
			middleware.RequirePermission(perms, "supervision:manage"),
		)
		supervisors.Post("/:id/assign", deps.SupervisionService.Assign)
		supervisors.Patch("/:id", deps.SupervisionService.UpdateQuota)

		supervisions := api.Group(
			"/supervisions",
			middleware.RequireJSON,
			middleware.RequireAuth(deps.JWT),
			middleware.RequirePermission(perms, "supervision:manage"),
		)
		supervisions.Post("/switch", deps.SupervisionService.Switch)
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