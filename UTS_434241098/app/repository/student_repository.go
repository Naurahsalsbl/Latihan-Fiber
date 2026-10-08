package repository

import (
	"context"
	"fmt"
	"strings"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"UTS_434241098/app/model"
)

type StudentRepository struct {
	DB *pgxpool.Pool
}

func IsDuplicateError(err error) bool {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return strings.Contains(
		err.Error(),
		"duplicate key",
	)
}

func NewStudentRepository(db *pgxpool.Pool) *StudentRepository {
	return &StudentRepository{DB: db}
}

func (r *StudentRepository) FindByID(
	ctx context.Context,
	id int,
) (*model.Student, error) {
	var student model.Student

	err := r.DB.QueryRow(ctx, `
		SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at
		FROM students
		WHERE id = $1
		AND deleted_at IS NULL
	`, id).Scan(
		&student.ID,
		&student.UserID,
		&student.NIM,
		&student.Nama,
		&student.Prodi,
		&student.Angkatan,
		&student.IPKTerakhir,
		&student.DeletedAt,
	)

	if err != nil {
		return nil, err
	}

	return &student, nil
}

func (r *StudentRepository) FindByUserID(
	ctx context.Context,
	userID int,
) (*model.Student, error) {
	var student model.Student

	err := r.DB.QueryRow(ctx, `
		SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at
		FROM students
		WHERE user_id = $1
		AND deleted_at IS NULL
	`, userID).Scan(
		&student.ID,
		&student.UserID,
		&student.NIM,
		&student.Nama,
		&student.Prodi,
		&student.Angkatan,
		&student.IPKTerakhir,
		&student.DeletedAt,
	)

	if err != nil {
		return nil, err
	}

	return &student, nil
}

func (r *StudentRepository) List(
	ctx context.Context,
	page int,
	perPage int,
	search string,
	prodi string,
	angkatan int,
	sort string,
) ([]model.Student, int, error) {

	offset := (page - 1) * perPage

	baseQuery := `
		FROM students
		WHERE deleted_at IS NULL
	`

	args := []any{}
	argIndex := 1

	if search != "" {
		baseQuery += fmt.Sprintf(
			` AND (nim ILIKE $%d OR nama ILIKE $%d)`,
			argIndex,
			argIndex,
		)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if prodi != "" {
		baseQuery += fmt.Sprintf(
			` AND prodi = $%d`,
			argIndex,
		)
		args = append(args, prodi)
		argIndex++
	}

	if angkatan > 0 {
		baseQuery += fmt.Sprintf(
			` AND angkatan = $%d`,
			argIndex,
		)
		args = append(args, angkatan)
		argIndex++
	}

	var total int

	err := r.DB.QueryRow(
		ctx,
		`SELECT COUNT(*) `+baseQuery,
		args...,
	).Scan(&total)

	if err != nil {
		return nil, 0, err
	}

	// Sorting hanya boleh menggunakan pilihan yang ditentukan
	orderBy := "nama ASC"

	switch sort {
	case "nama":
		orderBy = "nama ASC"
	case "-ipk_terakhir":
		orderBy = "ipk_terakhir DESC NULLS LAST"
	}

	query := `
		SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at
	` + baseQuery + `
		ORDER BY ` + orderBy + fmt.Sprintf(
		` LIMIT $%d OFFSET $%d`,
		argIndex,
		argIndex+1,
	)

	args = append(args, perPage, offset)

	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	students := make([]model.Student, 0)

	for rows.Next() {
		var student model.Student

		err := rows.Scan(
			&student.ID,
			&student.UserID,
			&student.NIM,
			&student.Nama,
			&student.Prodi,
			&student.Angkatan,
			&student.IPKTerakhir,
			&student.DeletedAt,
		)

		if err != nil {
			return nil, 0, err
		}

		students = append(students, student)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

func (r *StudentRepository) Create(
	ctx context.Context,
	student *model.Student,
	email string,
	passwordHash string,
) (*model.Student, error) {

	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Buat user mahasiswa
	var userID int

	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password, role)
		VALUES ($1, $2, $3)
		RETURNING id
	`,
		email,
		passwordHash,
		model.RoleMahasiswa,
	).Scan(&userID)

	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Buat data mahasiswa
	student.UserID = userID

	err = tx.QueryRow(ctx, `
		INSERT INTO students
			(user_id, nim, nama, prodi, angkatan, ipk_terakhir)
		VALUES
			($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at
	`,
		student.UserID,
		student.NIM,
		student.Nama,
		student.Prodi,
		student.Angkatan,
		student.IPKTerakhir,
	).Scan(
		&student.ID,
		&student.UserID,
		&student.NIM,
		&student.Nama,
		&student.Prodi,
		&student.Angkatan,
		&student.IPKTerakhir,
		&student.DeletedAt,
	)

	if err != nil {
		return nil, err
	}

	// Kalau semua berhasil, simpan permanen
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return student, nil
}

func (r *StudentRepository) Update(
	ctx context.Context,
	id int,
	req model.UpdateStudentRequest,
) (*model.Student, error) {

	var student model.Student

	err := r.DB.QueryRow(ctx, `
		UPDATE students
		SET nama = $1,
			prodi = $2,
			angkatan = $3,
			ipk_terakhir = $4
		WHERE id = $5
		AND deleted_at IS NULL
		RETURNING id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at
	`,
		req.Nama,
		req.Prodi,
		req.Angkatan,
		req.IPKTerakhir,
		id,
	).Scan(
		&student.ID,
		&student.UserID,
		&student.NIM,
		&student.Nama,
		&student.Prodi,
		&student.Angkatan,
		&student.IPKTerakhir,
		&student.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create student: %w", err)
	}

	return &student, nil
}

func (r *StudentRepository) Delete(
	ctx context.Context,
	id int,
) error {

	result, err := r.DB.Exec(ctx, `
		UPDATE students
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = $1
		AND deleted_at IS NULL
	`, id)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}