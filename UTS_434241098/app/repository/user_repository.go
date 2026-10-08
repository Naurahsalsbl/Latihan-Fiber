package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"UTS_434241098/app/model"
)

type UserRepository struct {
	DB *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*model.User, error) {
	var user model.User

	err := r.DB.QueryRow(ctx, `
		SELECT id, email, password, role
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Role,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) FindByID(
	ctx context.Context,
	id int,
) (*model.User, error) {
	var user model.User

	err := r.DB.QueryRow(ctx, `
		SELECT id, email, password, role
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Role,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) Create(
	ctx context.Context,
	email string,
	password string,
	role string,
) (*model.User, error) {
	var user model.User

	err := r.DB.QueryRow(ctx, `
		INSERT INTO users (email, password, role)
		VALUES ($1, $2, $3)
		RETURNING id, email, password, role
	`,
		email,
		password,
		role,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Role,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}