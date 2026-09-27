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
	repo  repository.StudentRepository
	perms *helper.PermissionSet
}

func NewStudentService(repo repository.StudentRepository, perms *helper.PermissionSet) *StudentService {
	return &StudentService{repo: repo, perms: perms}
}

// translateError mengubah error milik repository menjadi AppError.
//
// Perhatikan tanda tangannya: tidak ada fiber.Ctx. Fungsi ini hanya
// menerjemahkan satu jenis error menjadi jenis lain, dan tidak tahu
// apa pun tentang HTTP. Yang tidak dikenali menjadi Internal — fail
// closed: lebih baik membalas 500 daripada menebak-nebak status.
func translateError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("NIM sudah terdaftar")
	default:
		return helper.Internal(err)
	}
}

func ownerIDValue(ownerID *int) int {
	if ownerID == nil {
		return 0
	}
	return *ownerID
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q, err := helper.ParseCursorQuery(c)
	if err != nil {
		return err
	}

	students, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	format, err := helper.Negotiate(
		c,
		helper.FormatJSON,
		helper.FormatCSV,
	)
	if err != nil {
		return err
	}

	hasMore := len(students) > q.Limit

	if hasMore {
		students = students[:q.Limit]
	}

	var nextCursor string

	if hasMore && len(students) > 0 {
		last := students[len(students)-1]
		nextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	meta := model.CursorMeta{
		Limit:      q.Limit,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}

	if format == helper.FormatCSV {
		return helper.WriteStudentsCSV(c, students)
	}

	return helper.SuccessCursor(
		c,
		"berhasil mengambil daftar mahasiswa",
		students,
		&meta,
	)
}

func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("ID tidak valid")
	}

	ownerID, err := s.repo.FindOwnerID(ctx, id)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	if !CanAccessStudent(current, ownerIDValue(ownerID), s.perms, "student:read:any") {
		return helper.Forbidden("tidak berhak mengakses data mahasiswa milik orang lain")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "berhasil mengambil data mahasiswa", student)
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var reqBody model.CreateStudentRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return helper.BadRequest("request body tidak valid")
	}

	validationErrors := helper.ValidateStruct(reqBody)
	if validationErrors != nil {
		return helper.Validation(validationErrors)
	}

	ownerID := current.UserID

	req := model.Student{
		NIM:      reqBody.NIM,
		Name:     reqBody.Name,
		Grade:    reqBody.Grade,
		OwnerID:  &ownerID,
	}

	student, err := s.repo.Create(ctx, req)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	return helper.Success(c, fiber.StatusCreated, "berhasil menambahkan mahasiswa", student)
}

func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("ID tidak valid")
	}

	ownerID, err := s.repo.FindOwnerID(ctx, id)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	if !CanAccessStudent(current, ownerIDValue(ownerID), s.perms, "student:update:any") {
		return helper.Forbidden("tidak berhak mengubah data mahasiswa milik orang lain")
	}

	var reqBody model.ReplaceStudentRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return helper.BadRequest("request body tidak valid")
	}
	
	validationErrors := helper.ValidateStruct(reqBody)
	if validationErrors != nil {
		return helper.Validation(validationErrors)
	}

	req := model.Student{
		ID:       id,
		NIM:      reqBody.NIM,
		Name:     reqBody.Name,
		Grade:    reqBody.Grade,
		IsActive: reqBody.IsActive,
	}

	student, err := s.repo.Update(ctx, req)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "berhasil memperbarui data mahasiswa", student)
}

func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("ID tidak valid")
	}

	ownerID, err := s.repo.FindOwnerID(ctx, id)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	if !CanAccessStudent(current, ownerIDValue(ownerID), s.perms, "student:update:any") {
		return helper.Forbidden("tidak berhak mengubah data mahasiswa milik orang lain")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("request body tidak valid")
	}

	validationErrors := helper.ValidateStruct(req)
	if validationErrors != nil {
		return helper.Validation(validationErrors)
	}

	student, err := s.repo.Patch(ctx, id, req)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "berhasil memperbarui sebagian data mahasiswa", student)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("ID tidak valid")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(err, "mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "berhasil menghapus mahasiswa", nil)
}