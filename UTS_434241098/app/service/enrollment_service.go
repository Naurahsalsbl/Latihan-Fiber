package service

import (
	"context"
	"errors"
	"fmt"

	"UTS_434241098/app/model"
	"UTS_434241098/app/repository"
)

var (
	ErrDuplicateEnrollment = errors.New("mata kuliah sudah diambil")
	ErrQuotaFull           = errors.New("kuota mata kuliah sudah penuh")
	ErrSKSLimit            = errors.New("jumlah SKS melebihi batas")
	ErrEnrollmentForbidden = errors.New("enrollment milik mahasiswa lain")
)

type SKSLimitError struct {
	SisaSKS int
}

func (e *SKSLimitError) Error() string {
	return fmt.Sprintf(
		"jumlah SKS melebihi batas, sisa SKS yang dapat diambil: %d",
		e.SisaSKS,
	)
}

type EnrollmentService struct {
	Repo       *repository.EnrollmentRepository
	CourseRepo *repository.CourseRepository
	StudentRepo *repository.StudentRepository
}

func NewEnrollmentService(
	repo *repository.EnrollmentRepository,
	courseRepo *repository.CourseRepository,
	studentRepo *repository.StudentRepository,
) *EnrollmentService {
	return &EnrollmentService{
		Repo:       repo,
		CourseRepo: courseRepo,
		StudentRepo: studentRepo,
	}
}

func (s *EnrollmentService) Create(
	ctx context.Context,
	studentID int,
	req model.CreateEnrollmentRequest,
) (*model.Enrollment, error) {

	student, err := s.StudentRepo.FindByID(ctx, studentID)
	if err != nil {
		return nil, err
	}

	maxSKS := 18

	if student.IPKTerakhir != nil {
		ipk := *student.IPKTerakhir

		switch {
		case ipk >= 3.00:
			maxSKS = 24
		case ipk >= 2.50:
			maxSKS = 21
		}
	}

	enrollment, err := s.Repo.CreateWithTransaction(
		ctx,
		studentID,
		req.CourseID,
		req.TahunAkademik,
		maxSKS,
	)

	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEnrollment) {
			return nil, ErrDuplicateEnrollment
		}

		if errors.Is(err, repository.ErrQuotaFull) {
			return nil, ErrQuotaFull
		}

		var sksErr *repository.SKSLimitRepositoryError

		if errors.As(err, &sksErr) {
			return nil, &SKSLimitError{
				SisaSKS: sksErr.SisaSKS,
			}
		}

		return nil, err
}

return enrollment, nil
}

func (s *EnrollmentService) Delete(
	ctx context.Context,
	id int,
	studentID int,
) error {

	err := s.Repo.Delete(ctx, id, studentID)

	if errors.Is(err, repository.ErrEnrollmentForbidden) {
		return ErrEnrollmentForbidden
	}

	return err
}