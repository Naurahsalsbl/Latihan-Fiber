package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"UTS_434241098/app/repository"
	"UTS_434241098/app/service"
	"UTS_434241098/config"
	"UTS_434241098/route"
)

func main() {
	// Load file .env
	if err := godotenv.Load(); err != nil {
		log.Println("file .env tidak ditemukan, menggunakan environment variable")
	}

	// Koneksi database
	db, err := config.NewDatabase()
	if err != nil {
		log.Fatal("gagal membuat koneksi database:", err)
	}
	defer db.Close()

	// Test koneksi database
	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("gagal konek database:", err)
	}

	log.Println("database berhasil terhubung")

	app := fiber.New(fiber.Config{
		AppName: "SIAKAD Mini",
	})

	app.Use(recover.New())
	app.Use(logger.New())

	// Repository
	userRepo := repository.NewUserRepository(db)
	studentRepo := repository.NewStudentRepository(db)
	courseRepo := repository.NewCourseRepository(db)
	enrollmentRepo := repository.NewEnrollmentRepository(db)

	// Service
	authService := service.NewAuthService(userRepo)
	studentService := service.NewStudentService(studentRepo, enrollmentRepo,
	)
	courseService := service.NewCourseService(courseRepo)
	enrollmentService := service.NewEnrollmentService(
		enrollmentRepo,
		courseRepo,
		studentRepo,
	)

	// Login limiter
	loginLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		SkipSuccessfulRequests: true,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"message": "Terlalu banyak percobaan login. Coba lagi dalam 1 menit",
			})
		},
	})

	route.Setup(
		app,
		authService,
		studentService,
		courseService,
		enrollmentService,
		loginLimiter,
	)

	// Routes akan kita pasang setelah handler selesai dibuat.
	// route.Setup(app, ...)
	_ = os.Getenv("PORT")

	log.Println("SIAKAD Mini siap dijalankan")
	log.Fatal(app.Listen(":3000"))
}