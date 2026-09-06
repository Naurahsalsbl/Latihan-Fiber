package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/modul4/app/model"
	"latihan-fiber/modul4/app/repository"
	"latihan-fiber/modul4/helper"
)

// StudentService memegang dua tanggung jawab sekaligus pada struktur baku
// mata kuliah ini: menerima *fiber.Ctx (peran controller) dan mengoordinasi
// pemanggilan business rules murni di student_rules.go (peran use case).
// Business rules itu sendiri TIDAK ditulis ulang di sini, hanya dipanggil.
type StudentService struct {
	repo repository.StudentRepository
}

func NewStudentService(repo repository.StudentRepository) *StudentService {
	return &StudentService{repo: repo}
}

// translateError memetakan error milik repository menjadi status HTTP.
// Pesan dipertahankan persis sama seperti pada handler lama.
func translateError(c *fiber.Ctx, err error, generalMessage string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.Fail(c, fiber.StatusNotFound, "data mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Fail(c, fiber.StatusConflict, "NIM sudah terdaftar")
	default:
		return helper.Fail(c, fiber.StatusInternalServerError, generalMessage)
	}
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q, err := helper.ParseListQuery(c)
	if err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "parameter query tidak valid")
	}

	students, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return translateError(c, err, "gagal mengambil daftar mahasiswa")
	}

	meta := model.Meta{
		Page:      q.Page,
		Limit:     q.Limit,
		TotalData: total,
		TotalPage: CountTotalPage(total, q.Limit),
	}

	return helper.SuccessList(c, "berhasil mengambil daftar mahasiswa", students, meta)
}

func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(c, err, "gagal mengambil data mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "berhasil mengambil data mahasiswa", student)
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.Student
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "request body tidak valid")
	}

	// Business rules-nya dipanggil, bukan ditulis ulang di sini.
	if msg := ValidateCreate(req); msg != "" {
		return helper.Fail(c, fiber.StatusBadRequest, msg)
	}

	student, err := s.repo.Create(ctx, req)
	if err != nil {
		return translateError(c, err, "gagal menambahkan mahasiswa")
	}

	return helper.Success(c, fiber.StatusCreated, "berhasil menambahkan mahasiswa", student)
}

func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req model.Student
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "request body tidak valid")
	}
	req.ID = id

	if msg := ValidateReplace(req); msg != "" {
		return helper.Fail(c, fiber.StatusBadRequest, msg)
	}

	student, err := s.repo.Update(ctx, req)
	if err != nil {
		return translateError(c, err, "gagal memperbarui data mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "berhasil memperbarui data mahasiswa", student)
}

func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "request body tidak valid")
	}

	if msg := ValidatePatch(req); msg != "" {
		return helper.Fail(c, fiber.StatusBadRequest, msg)
	}

	student, err := s.repo.Patch(ctx, id, req)
	if err != nil {
		return translateError(c, err, "gagal memperbarui sebagian data mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "berhasil memperbarui sebagian data mahasiswa", student)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(c, err, "gagal menghapus mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "berhasil menghapus mahasiswa", nil)
}