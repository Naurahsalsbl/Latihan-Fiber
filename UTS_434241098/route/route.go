package route

import (
	"strconv"
	"time"
	"strings"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"

	"UTS_434241098/app/model"
	"UTS_434241098/app/service"
	"UTS_434241098/helper"
	"UTS_434241098/middleware"
)

func Setup(
	app *fiber.App,
	authService *service.AuthService,
	studentService *service.StudentService,
	courseService *service.CourseService,
	enrollmentService *service.EnrollmentService,
	loginLimiter fiber.Handler,
) {
	api := app.Group("/api/v1")

	// =========================
	// AUTH
	// =========================
	api.Post("/auth/login", loginLimiter, loginHandler(authService))

	auth := api.Group("", middleware.Auth)

	auth.Get("/auth/me", meHandler(authService, studentService))

	// =========================
	// STUDENTS
	// =========================
	auth.Get("/students", studentListHandler(studentService))
	auth.Post(
		"/students",
		middleware.RequireRole(model.RoleAdmin),
		studentCreateHandler(studentService),
	)

	auth.Get("/students/:id", studentDetailHandler(studentService))

	auth.Put(
		"/students/:id",
		middleware.RequireRole(model.RoleAdmin),
		studentUpdateHandler(studentService),
	)

	auth.Delete(
		"/students/:id",
		middleware.RequireRole(model.RoleAdmin),
		studentDeleteHandler(studentService),
	)

	// =========================
	// COURSES
	// =========================
	auth.Get("/courses", courseListHandler(courseService))

	// =========================
	// ENROLLMENTS
	// =========================
	auth.Post(
		"/enrollments",
		middleware.RequireRole(model.RoleMahasiswa),
		enrollmentCreateHandler(enrollmentService, studentService),
	)

	auth.Delete(
		"/enrollments/:id",
		middleware.RequireRole(model.RoleMahasiswa),
		enrollmentDeleteHandler(enrollmentService, studentService),
	)
}

// =========================
// AUTH HANDLER
// =========================

func loginHandler(authService *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req model.LoginRequest

		if err := c.BodyParser(&req); err != nil {
			return helper.Error(
				c,
				fiber.StatusBadRequest,
				"Format request tidak valid",
				nil,
			)
		}

		validate := validator.New()

		if err := validate.Struct(req); err != nil {
			return helper.Error(
				c,
				fiber.StatusUnprocessableEntity,
				"validasi gagal",
				err.Error(),
			)
		}

		result, err := authService.Login(c.Context(), req)
		if err != nil {
			return helper.Error(
				c,
				fiber.StatusUnauthorized,
				err.Error(),
				nil,
			)
		}

		return helper.Success(
			c,
			fiber.StatusOK,
			"Login berhasil",
			result,
		)
	}
}

func meHandler(authService *service.AuthService, studentService *service.StudentService, ) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(int)
		if !ok {
			return helper.Error(
				c,
				fiber.StatusUnauthorized,
				"User belum terautentikasi",
				nil,
			)
		}

		user, err := authService.Me(c.Context(), userID)
		if err != nil {
			return helper.Error(
				c,
				fiber.StatusUnauthorized,
				"User tidak ditemukan",
				nil,
			)
		}

		response := model.MeResponse{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		}

		if user.Role == model.RoleMahasiswa {
			student, err := studentService.FindByUserID(c.Context(), userID)
			if err != nil {
				return helper.Error(
					c,
					fiber.StatusNotFound,
					"Data mahasiswa tidak ditemukan",
					nil,
				)
			}

			response.Student = student
		}

		return helper.Success(
			c,
			fiber.StatusOK,
			"Data user berhasil diambil",
			response,
		)
	}
}

// =========================
// STUDENT HANDLER
// =========================

func studentListHandler(studentService *service.StudentService) fiber.Handler {
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

		// Endpoint GET /students hanya untuk admin
		if role != model.RoleAdmin {
			return helper.Error(
				c,
				fiber.StatusForbidden,
				"Akses ditolak",
				nil,
			)
		}

		page, _ := strconv.Atoi(c.Query("page", "1"))
		perPage, _ := strconv.Atoi(c.Query("per_page", "10"))
		angkatan, _ := strconv.Atoi(c.Query("angkatan", "0"))

		search := c.Query("search")
		prodi := c.Query("prodi")
		sort := c.Query("sort", "nama")

		students, total, lastPage, err := studentService.List(
			c.Context(),
			page,
			perPage,
			search,
			prodi,
			angkatan,
			sort,
		)

		if err != nil {
			return helper.Error(
				c,
				fiber.StatusInternalServerError,
				"Gagal mengambil data mahasiswa",
				nil,
			)
		}

		return helper.Success(
			c,
			fiber.StatusOK,
			"Data mahasiswa berhasil diambil",
			fiber.Map{
				"data": students,
				"meta": fiber.Map{
					"current_page": page,
					"per_page":     perPage,
					"total":        total,
					"last_page":    lastPage,
				},
			},
		)
	}
}

func studentDetailHandler(studentService *service.StudentService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return helper.Error(c, 400, "ID mahasiswa tidak valid", nil)
		}

		student, courses, totalSKS, batasSKS, err := studentService.GetDetail(c.Context(), id)
		if err != nil {
			return helper.Error(c, 404, "Mahasiswa tidak ditemukan", nil)
		}

		role, _ := c.Locals("role").(string)

		// Mahasiswa hanya boleh melihat data dirinya sendiri
		if role == model.RoleMahasiswa {
			userID, _ := c.Locals("user_id").(int)

			if student.UserID != userID {
				return helper.Error(c, 403, "Akses ditolak", nil)
			}
		}

		data := fiber.Map{
			"id":            student.ID,
			"user_id":       student.UserID,
			"nim":           student.NIM,
			"nama":          student.Nama,
			"prodi":         student.Prodi,
			"angkatan":      student.Angkatan,
			"ipk_terakhir":  student.IPKTerakhir,
			"mata_kuliah":   courses,
			"total_sks":     totalSKS,
			"batas_sks":     batasSKS,
		}

		return helper.Success(
			c,
			200,
			"Data mahasiswa berhasil diambil",
			data,
		)
	}
}

func studentCreateHandler(studentService *service.StudentService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req model.CreateStudentRequest

		if err := c.BodyParser(&req); err != nil {
			return helper.Error(
				c,
				fiber.StatusBadRequest,
				"Format request tidak valid",
				nil,
			)
		}

		// Validasi field
		validate := validator.New()

		if err := validate.Struct(req); err != nil {
			return helper.Error(
				c,
				fiber.StatusUnprocessableEntity,
				"validasi gagal",
				err.Error(),
			)
		}

		// Angkatan tidak boleh lebih dari tahun berjalan
		currentYear := time.Now().Year()

		if req.Angkatan > currentYear {
			return helper.Error(
				c,
				fiber.StatusUnprocessableEntity,
				"angkatan tidak boleh lebih dari tahun berjalan",
				nil,
			)
		}

		student, err := studentService.Create(
			c.Context(),
			req,
		)

		if err != nil {
			errMessage := err.Error()

			if strings.Contains(err.Error(), "duplicate key") ||
				strings.Contains(errMessage, "NIM atau email sudah terdaftar") ||
				strings.Contains(err.Error(), "unique constraint") {
				return helper.Error(
					c,
					fiber.StatusUnprocessableEntity,
					"NIM atau email sudah terdaftar",
					nil,
				)
			}

			return helper.Error(
				c,
				fiber.StatusInternalServerError,
				"Gagal membuat data mahasiswa",
				err.Error(),
			)
		}

		return helper.Success(
			c,
			fiber.StatusCreated,
			"Data mahasiswa berhasil dibuat",
			student,
		)
	}
}

func studentUpdateHandler(studentService *service.StudentService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return helper.Error(
				c,
				fiber.StatusBadRequest,
				"ID mahasiswa tidak valid",
				nil,
			)
		}

		var req model.UpdateStudentRequest

		if err := c.BodyParser(&req); err != nil {
			return helper.Error(
				c,
				fiber.StatusBadRequest,
				"Format request tidak valid",
				nil,
			)
		}

		student, err := studentService.Update(
			c.Context(),
			id,
			req,
		)

		if err != nil {
			return helper.Error(
				c,
				fiber.StatusNotFound,
				"Mahasiswa tidak ditemukan",
				nil,
			)
		}

		return helper.Success(
			c,
			fiber.StatusOK,
			"Data mahasiswa berhasil diperbarui",
			student,
		)
	}
}

func studentDeleteHandler(studentService *service.StudentService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return helper.Error(
				c,
				fiber.StatusBadRequest,
				"ID mahasiswa tidak valid",
				nil,
			)
		}

		if err := studentService.Delete(c.Context(), id); err != nil {
			return helper.Error(
				c,
				fiber.StatusNotFound,
				"Mahasiswa tidak ditemukan",
				nil,
			)
		}

		return helper.Success(
			c,
			fiber.StatusOK,
			"Data mahasiswa berhasil dihapus",
			nil,
		)
	}
}

// =========================
// COURSE HANDLER
// =========================

func courseListHandler(courseService *service.CourseService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		semester := 0

		if value := c.Query("semester"); value != "" {
			var err error
			semester, err = strconv.Atoi(value)

			if err != nil || semester < 1 {
				return helper.Error(c, 422, "Semester tidak valid", nil)
			}
		}

		search := strings.TrimSpace(c.Query("search"))

		available := false
		if c.Query("available") == "true" {
			available = true
		}

		courses, err := courseService.List(
			c.Context(),
			semester,
			search,
			available,
		)
		if err != nil {
			return helper.Error(
				c,
				fiber.StatusInternalServerError,
				"Gagal mengambil data mata kuliah",
				nil,
			)
		}

		return helper.Success(
			c,
			fiber.StatusOK,
			"Data mata kuliah berhasil diambil",
			courses,
		)
	}
}

// =========================
// ENROLLMENT HANDLER
// =========================

func enrollmentCreateHandler(
	enrollmentService *service.EnrollmentService,
	studentService *service.StudentService,
) fiber.Handler {

	return func(c *fiber.Ctx) error {

		role, _ := c.Locals("role").(string)

		if role != model.RoleMahasiswa {
			return helper.Error(
				c,
				fiber.StatusForbidden,
				"Hanya mahasiswa yang dapat mengambil mata kuliah",
				nil,
			)
		}

		var req model.CreateEnrollmentRequest

		if err := c.BodyParser(&req); err != nil {
			return helper.Error(
				c,
				fiber.StatusBadRequest,
				"Format request tidak valid",
				nil,
			)
		}

		if req.CourseID <= 0 {
			return helper.Error(
				c,
				fiber.StatusUnprocessableEntity,
				"course_id wajib diisi",
				nil,
			)
		}

		if req.TahunAkademik == "" {
			return helper.Error(
				c,
				fiber.StatusUnprocessableEntity,
				"tahun_akademik wajib diisi",
				nil,
			)
		}

		userID, _ := c.Locals("user_id").(int)

		student, err := studentService.FindByUserID(
			c.Context(),
			userID,
		)

		if err != nil {
			return helper.Error(
				c,
				fiber.StatusNotFound,
				"Data mahasiswa tidak ditemukan",
				nil,
			)
		}

		enrollment, err := enrollmentService.Create(
			c.Context(),
			student.ID,
			req,
		)

		if err != nil {

			switch err {

			case service.ErrDuplicateEnrollment:
				return helper.Error(
					c,
					fiber.StatusConflict,
					"Mata kuliah sudah pernah diambil",
					nil,
				)

			case service.ErrQuotaFull:
				return helper.Error(
					c,
					fiber.StatusUnprocessableEntity,
					"Kuota mata kuliah sudah penuh",
					nil,
				)
			}

			var sksErr *service.SKSLimitError

			if errors.As(err, &sksErr) {
				return helper.Error(
					c,
					fiber.StatusUnprocessableEntity,
					sksErr.Error(),
					nil,
				)
			}

			return helper.Error(
				c,
				fiber.StatusInternalServerError,
				"Gagal mengambil mata kuliah",
				nil,
			)
		}

		return helper.Success(
			c,
			fiber.StatusCreated,
			"Mata kuliah berhasil diambil",
			enrollment,
		)
	}
}

func enrollmentDeleteHandler(enrollmentService *service.EnrollmentService, studentService *service.StudentService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		id, err := strconv.Atoi(c.Params("id"))
		if err != nil || id <= 0 {
			return helper.Error(
				c,
				fiber.StatusUnprocessableEntity,
				"ID enrollment tidak valid",
				nil,
			)
		}

		userID, ok := c.Locals("user_id").(int)
		if !ok {
			return helper.Error(
				c,
				fiber.StatusUnauthorized,
				"User belum terautentikasi",
				nil,
			)
		}

		student, err := studentService.FindByUserID(
			c.Context(),
			userID,
		)
		if err != nil {
			return helper.Error(
				c,
				fiber.StatusForbidden,
				"Data mahasiswa tidak ditemukan",
				nil,
			)
		}

		err = enrollmentService.Delete(
			c.Context(),
			id,
			student.ID,
		)

		if err != nil {

			if errors.Is(err, service.ErrEnrollmentForbidden) {
				return helper.Error(
					c,
					fiber.StatusForbidden,
					"Enrollment milik mahasiswa lain",
					nil,
				)
			}

			if errors.Is(err, pgx.ErrNoRows) {
				return helper.Error(
					c,
					fiber.StatusNotFound,
					"Enrollment tidak ditemukan",
					nil,
				)
			}

			return helper.Error(
				c,
				fiber.StatusInternalServerError,
				"Gagal menghapus enrollment",
				nil,
			)
		}

		return helper.Success(
			c,
			fiber.StatusOK,
			"Enrollment berhasil dihapus",
			nil,
		)
	}
}