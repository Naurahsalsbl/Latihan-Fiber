package service

import (
	"context"
	"errors"
	
	"UTS_434241098/app/model"
	"UTS_434241098/app/repository"
	"UTS_434241098/helper"
)

var ErrStudentDuplicate = errors.New("NIM atau email sudah terdaftar")

type StudentService struct {
	Repo *repository.StudentRepository
	EnrollmentRepo *repository.EnrollmentRepository
}

func NewStudentService(
	repo *repository.StudentRepository,
	enrollmentRepo *repository.EnrollmentRepository,
) *StudentService {
	return &StudentService{
		Repo:           repo,
		EnrollmentRepo: enrollmentRepo,
	}
}

func (s *StudentService) FindByID(
	ctx context.Context,
	id int,
) (*model.Student, error) {
	return s.Repo.FindByID(ctx, id)
}

func (s *StudentService) FindByUserID(
	ctx context.Context,
	userID int,
) (*model.Student, error) {
	return s.Repo.FindByUserID(ctx, userID)
}

func (s *StudentService) List(
	ctx context.Context,
	page int,
	perPage int,
	search string,
	prodi string,
	angkatan int,
	sort string,
) ([]model.Student, int, int, error) {

	if page < 1 {
		page = 1
	}

	if perPage < 1 {
		perPage = 10
	}

	if perPage > 50 {
		perPage = 50
	}

	students, total, err := s.Repo.List(
		ctx,
		page,
		perPage,
		search,
		prodi,
		angkatan,
		sort,
	)

	if err != nil {
		return nil, 0, 0, err
	}

	lastPage := 1

	if total > 0 {
		lastPage = (total + perPage - 1) / perPage
	}

	return students, total, lastPage, nil
}

func (s *StudentService) GetDetail(
	ctx context.Context,
	id int,
) (*model.Student, []model.StudentCourse, int, int, error) {

	student, err := s.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, nil, 0, 0, err
	}

	courses, err := s.EnrollmentRepo.ListByStudent(ctx, id)
	if err != nil {
		return nil, nil, 0, 0, err
	}

	totalSKS := 0
	for _, course := range courses {
		totalSKS += course.SKS
	}

	batasSKS := 18

	if student.IPKTerakhir != nil {
		if *student.IPKTerakhir >= 3.00 {
			batasSKS = 24
		} else if *student.IPKTerakhir >= 2.50 {
			batasSKS = 21
		}
	}

	return student, courses, totalSKS, batasSKS, nil
}

func (s *StudentService) Create(
	ctx context.Context,
	req model.CreateStudentRequest,
) (*model.Student, error) {

	passwordHash, err := helper.HashPassword(req.NIM)
	if err != nil {
		return nil, err
	}

	student := &model.Student{
		NIM:         req.NIM,
		Nama:        req.Nama,
		Prodi:       req.Prodi,
		Angkatan:    req.Angkatan,
		IPKTerakhir: req.IPKTerakhir,
	}

	result, err := s.Repo.Create(
		ctx,
		student,
		req.Email,
		passwordHash,
	)

	if err != nil {
		// Error duplicate dari database
		if repository.IsDuplicateError(err) {
			return nil, ErrStudentDuplicate
		}

		return nil, err
	}

	return result, nil
}

func (s *StudentService) Update(
	ctx context.Context,
	id int,
	req model.UpdateStudentRequest,
) (*model.Student, error) {
	return s.Repo.Update(ctx, id, req)
}

func (s *StudentService) Delete(
	ctx context.Context,
	id int,
) error {
	return s.Repo.Delete(ctx, id)
}