package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/modul4/app/model"
	"latihan-fiber/modul4/app/repository"
	"latihan-fiber/modul4/helper"
)

var ErrAlreadySupervised = errors.New("mahasiswa sudah memiliki pembimbing")

type SupervisionService struct {
	supervisors repository.SupervisorRepository
	tx           *repository.TxManager
}

func NewSupervisionService(
	supervisors repository.SupervisorRepository,
	tx *repository.TxManager,
) *SupervisionService {
	return &SupervisionService{
		supervisors: supervisors,
		tx:           tx,
	}
}

// POST /supervisors/:id/assign
func (s *SupervisionService) Assign(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	supervisorID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id dosen pembimbing harus berupa angka positif")
	}

	var req model.AssignSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)

		// Kunci mahasiswa terlebih dahulu agar dua assign untuk
		// mahasiswa yang sama tidak berjalan bersamaan.
		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		_, err := repo.FindSupervision(ctx, req.StudentID)
		if err == nil {
			return ErrAlreadySupervised
		}
		if !errors.Is(err, repository.ErrNoSupervision) {
			return err
		}

		if _, err := repo.LockSupervisor(ctx, supervisorID); err != nil {
			return err
		}
		if err := repo.IncrementIfAvailable(ctx, supervisorID); err != nil {
			return err
		}

		return repo.InsertSupervision(ctx, req.StudentID, supervisorID)
	})
	if err != nil {
		return translateSupervisionError(err)
	}

	return helper.Success(c, fiber.StatusCreated, "dosen pembimbing berhasil ditetapkan",
		fiber.Map{
			"student_id":   req.StudentID,
			"supervisor_id": supervisorID,
		})
}

// POST /supervisions/switch
func (s *SupervisionService) Switch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.SwitchSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)

		// Kunci mahasiswa sebelum membaca relasi pembimbing.
		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		oldSupervisorID, err := repo.FindSupervision(ctx, req.StudentID)
		if err != nil {
			return err
		}
		if oldSupervisorID == req.SupervisorID {
			return repository.ErrSameSupervisor
		}

		// Kunci pembimbing lama dan baru berdasarkan ID menaik.
		if err := repository.LockSupervisors(
			ctx, repo, oldSupervisorID, req.SupervisorID,
		); err != nil {
			return err
		}

		// Tambah kuota pembimbing baru terlebih dahulu.
		// Jika penuh, transaksi batal dan pembimbing lama tetap tercatat.
		if err := repo.IncrementIfAvailable(ctx, req.SupervisorID); err != nil {
			return err
		}

		if err := repo.ChangeSupervision(ctx, req.StudentID, req.SupervisorID); err != nil {
			return err
		}

		return repo.DecrementSupervised(ctx, oldSupervisorID)
	})
	if err != nil {
		return translateSupervisionError(err)
	}

	return helper.Success(c, fiber.StatusOK, "dosen pembimbing berhasil diganti",
		fiber.Map{
			"student_id":    req.StudentID,
			"supervisor_id": req.SupervisorID,
		})
}

// PATCH /supervisors/:id
func (s *SupervisionService) UpdateQuota(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id dosen pembimbing harus berupa angka positif")
	}

	var req model.UpdateSupervisorQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.supervisors.UpdateQuota(ctx, id, req.Quota, req.Version)
	if err != nil {
		return translateSupervisionError(err)
	}

	return helper.Success(c, fiber.StatusOK, "kuota dosen pembimbing berhasil diubah", updated)
}

func translateSupervisionError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrSupervisorNotFound):
		return helper.NotFound("dosen pembimbing tidak ditemukan")
	case errors.Is(err, repository.ErrNoSupervision):
		return helper.Conflict("mahasiswa belum memiliki pembimbing")
	case errors.Is(err, ErrAlreadySupervised):
		return helper.NewError(fiber.StatusConflict, "ALREADY_SUPERVISED",
			"mahasiswa sudah memiliki pembimbing")
	case errors.Is(err, repository.ErrSupervisorFull):
		return helper.NewError(fiber.StatusConflict, "SUPERVISOR_FULL",
			"kuota dosen pembimbing sudah penuh")
	case errors.Is(err, repository.ErrSameSupervisor):
		return helper.Validation(map[string]string{
			"supervisor_id": "harus berbeda dari pembimbing saat ini",
		})
	case errors.Is(err, repository.ErrSupervisorQuota):
		return helper.NewError(fiber.StatusConflict, "QUOTA_BELOW_SUPERVISED",
			"kuota tidak boleh lebih kecil dari jumlah mahasiswa bimbingan")
	case errors.Is(err, repository.ErrSupervisorVersion):
		return helper.NewError(fiber.StatusConflict, "VERSION_CONFLICT",
			"data pembimbing sudah diubah pihak lain, muat ulang lalu coba lagi")
	default:
		_ = strconv.Itoa(0) // placeholder tidak diperlukan; hapus jika tidak dipakai
		return helper.Internal(err)
	}
}