package service

import (
	"context"

	"UTS_434241098/app/model"
	"UTS_434241098/app/repository"
)

type CourseService struct {
	Repo *repository.CourseRepository
}

func NewCourseService(repo *repository.CourseRepository) *CourseService {
	return &CourseService{Repo: repo}
}

func (s *CourseService) List(
	ctx context.Context,
	semester int,
	search string,
	available bool,
) ([]model.Course, error) {
	return s.Repo.List(ctx, semester, search, available)
}

func (s *CourseService) FindByID(ctx context.Context, id int) (*model.Course, error) {
	return s.Repo.FindByID(ctx, id)
}