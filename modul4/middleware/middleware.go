package middleware

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"latihan-fiber/modul4/helper"
)

// Register memasang middleware yang berlaku untuk semua route.
// URUTAN PENTING: middleware dieksekusi sesuai urutan pemasangan.
//
// Middleware ini BARU ditambahkan saat restrukturisasi Pertemuan 4 dan
// tidak ada pada kode Pertemuan 3. Penambahan requestid, recover, dan cors
// tidak mengubah status maupun isi body JSON pada respons manapun — hanya
// menambah beberapa header (mis. X-Request-Id, Access-Control-*) dan
// menangkap panic agar server tidak mati. Karena itu penambahan ini aman
// terhadap syarat "perilaku HTTP tidak boleh berubah".
func Register(app *fiber.App, logger *slog.Logger, allowedOrigins string) {
	app.Use(requestid.New()) // 1. beri setiap request satu ID unik
	app.Use(recover.New())   // 2. tangkap panic agar server tidak mati
	app.Use(helmet.New())
	app.Use(corsPolicy(allowedOrigins))
	app.Use(RequestLogger(logger)) // 4. catat setiap request ke logger terstruktur
}

// RequestLogger mencatat setiap request ke log terstruktur (JSON).
func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		requestID, _ := c.Locals("requestid").(string)

		// Sejak handler mengembalikan error alih-alih menulis response sendiri,
		// status pada c.Response() BELUM terisi ketika baris ini dijalankan:
		// ErrorHandler baru berjalan setelah seluruh rangkaian middleware selesai.
		// Tanpa koreksi di bawah, setiap kegagalan tercatat sebagai 200.
		status := c.Response().StatusCode()
		if err != nil {
			var appErr *helper.AppError
			if errors.As(err, &appErr) {
				status = appErr.Status
			} else {
				status = fiber.StatusInternalServerError
			}
		}

		attrs := []any{
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.IP()),
		}
 
		if user, ok := helper.CurrentUser(c); ok {
			attrs = append(attrs,
				slog.Int("user_id", user.UserID),
				slog.String("role", user.Role))
		}
 
		logger.Info("http_request", attrs...)

		return err
	}
}

func corsPolicy(allowedOrigins string) fiber.Handler {
	if strings.TrimSpace(allowedOrigins) == "" {
		allowedOrigins = "http://localhost:5173"
	}
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	})
}

var methodsWithBody = map[string]bool{
	fiber.MethodPost:  true,
	fiber.MethodPut:   true,
	fiber.MethodPatch: true,
}

// RequireJSON menolak request berisi body yang Content-Type-nya bukan JSON.
//
// CATATAN PENTING: fungsi ini TERSEDIA tetapi SENGAJA BELUM dipasang pada
// grup /students di route/route.go. Kode Pertemuan 3 tidak memiliki
// pemeriksaan Content-Type sama sekali, sehingga memasangnya sekarang akan
// membuat request yang sebelumnya diterima (tanpa header Content-Type yang
// benar) menjadi ditolak dengan 415 — itu adalah perubahan perilaku yang
// dilarang oleh modul ini. Untuk mengaktifkannya, tinggal tambahkan
// middleware.RequireJSON sebagai argumen kedua pada api.Group("/students", ...)
// di route/route.go.
func RequireJSON(c *fiber.Ctx) error {
	if methodsWithBody[c.Method()] {
		ct := c.Get("Content-Type")
		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
			return helper.UnsupportedMediaType("Content-Type harus application/json")
		}
	}
	return c.Next()
}