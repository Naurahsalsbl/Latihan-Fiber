package service

import (
	"context"
	"errors"

	"UTS_434241098/app/model"
	"UTS_434241098/app/repository"
	"UTS_434241098/helper"
)

var ErrInvalidCredentials = errors.New("email atau password salah")

type AuthService struct {
	UserRepo *repository.UserRepository
}

func NewAuthService(
	userRepo *repository.UserRepository,
) *AuthService {
	return &AuthService{
		UserRepo: userRepo,
	}
}

func (s *AuthService) Login(
	ctx context.Context,
	req model.LoginRequest,
) (*model.LoginResponse, error) {

	user, err := s.UserRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !helper.CheckPassword(req.Password, user.Password) {
		return nil, ErrInvalidCredentials
	}

	token, expiresIn, err := helper.GenerateToken(
		user.ID,
		user.Email,
		user.Role,
	)
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		User:        *user,
	}, nil
}

func (s *AuthService) Me(
	ctx context.Context,
	userID int,
) (*model.User, error) {
	return s.UserRepo.FindByID(ctx, userID)
}