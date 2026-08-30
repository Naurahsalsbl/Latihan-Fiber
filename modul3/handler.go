
package main

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"latihan-fiber/modul3/app/model"
	"latihan-fiber/modul3/app/repository"
)

type StudentHandler struct {
	repo repository.StudentRepository
}

func NewStudentHandler(repo repository.StudentRepository) *StudentHandler {
	return &StudentHandler{repo: repo}
}

func terjemahkanError(c *fiber.Ctx, err error, pesanUmum string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return fail(c, fiber.StatusNotFound, "data mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return fail(c, fiber.StatusConflict, "NIM sudah terdaftar")
	default:
		return fail(c, fiber.StatusInternalServerError, pesanUmum)
	}
}

func (h *StudentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	var q model.ListQuery
	if err := c.QueryParser(&q); err != nil {
		return fail(c, fiber.StatusBadRequest, "parameter query tidak valid")
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 10
	}

	students, total, err := h.repo.FindAll(ctx, q)
	if err != nil {
		return terjemahkanError(c, err, "gagal mengambil daftar mahasiswa")
	}

	totalPage := 0
	if total > 0 {
		totalPage = (total + q.Limit - 1) / q.Limit
	}

	meta := model.Meta{
		Page:      q.Page,
		Limit:     q.Limit,
		TotalData: total,
		TotalPage: totalPage,
	}

	return okWithMeta(c, "berhasil mengambil daftar mahasiswa", students, meta)
}

func (h *StudentHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return fail(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	student, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(c, err, "gagal mengambil data mahasiswa")
	}

	return ok(c, "berhasil mengambil data mahasiswa", student)
}

func (h *StudentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	var req model.Student
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "request body tidak valid")
	}

	if req.NIM == "" || req.Name == "" {
		return fail(c, fiber.StatusBadRequest, "NIM, nama, dan grade wajib diisi")
	}

	if req.Grade < 0 || req.Grade > 100 {
		return fail(c, fiber.StatusBadRequest, "grade harus di antara 0 dan 100")
	}

	student, err := h.repo.Create(ctx, req)
	if err != nil {
		return terjemahkanError(c, err, "gagal menambahkan mahasiswa")
	}

	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Status:  "success",
		Message: "berhasil menambahkan mahasiswa",
		Data:    student,
	})
}

func (h *StudentHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return fail(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req model.Student
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "request body tidak valid")
	}
	req.ID = id

	if req.NIM == "" || req.Name == "" {
		return fail(c, fiber.StatusBadRequest, "NIM, nama, dan grade wajib diisi")
	}

	if req.Grade < 0 || req.Grade > 100 {
		return fail(c, fiber.StatusBadRequest, "grade harus di antara 0 dan 100")
	}

	student, err := h.repo.Update(ctx, req)
	if err != nil {
		return terjemahkanError(c, err, "gagal memperbarui data mahasiswa")
	}

	return ok(c, "berhasil memperbarui data mahasiswa", student)
}

func (h *StudentHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return fail(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req model.PatchStudentRequest

	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "request body tidak valid")
	}

	// Minimal satu field harus dikirim
	if req.NIM == nil &&
		req.Name == nil &&
		req.Grade == nil &&
		req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

	// Validasi NIM
	if req.NIM != nil && *req.NIM == "" {
		return fail(c, fiber.StatusBadRequest, "NIM tidak boleh kosong")
	}

	// Validasi nama
	if req.Name != nil && *req.Name == "" {
		return fail(c, fiber.StatusBadRequest, "nama tidak boleh kosong")
	}

	// Validasi grade
	if req.Grade != nil && (*req.Grade < 0 || *req.Grade > 100) {
		return fail(c, fiber.StatusBadRequest, "grade harus di antara 0 dan 100")
	}

	student, err := h.repo.Patch(ctx, id, req)
	if err != nil {
		return terjemahkanError(
			c,
			err,
			"gagal memperbarui sebagian data mahasiswa",
		)
	}

	return ok(
		c,
		"berhasil memperbarui sebagian data mahasiswa",
		student,
	)
}

func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return fail(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		return terjemahkanError(c, err, "gagal menghapus mahasiswa")
	}

	return ok(c, "berhasil menghapus mahasiswa", nil)
}