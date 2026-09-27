package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
 
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
 
	"latihan-fiber/modul4/app/model"
)
 
// Sentinel error yang diterjemahkan menjadi status HTTP di layer service.
// Package ini TIDAK mengetahui apa pun tentang HTTP atau Fiber.
var (
	ErrNotFound  = errors.New("data mahasiswa tidak ditemukan")
	ErrDuplicate = errors.New("NIM mahasiswa sudah terdaftar")
)
 
// StudentRepository adalah interface (gateway) yang dipakai oleh service.
// Interface ini dideklarasikan bersama implementasinya di package yang sama
// sebagai penyederhanaan mata kuliah (lihat catatan Dependency Inversion
// pada Modul 4 bagian 4 dan 7).
type StudentRepository interface {
	FindAll(ctx context.Context, q model.CursorQuery) ([]model.Student, error)
	FindByID(ctx context.Context, id int) (model.Student, error)
	FindOwnerID(ctx context.Context, id int) (*int, error)
	Create(ctx context.Context, s model.Student) (model.Student, error)
	Update(ctx context.Context, s model.Student) (model.Student, error)
	Patch(ctx context.Context, id int, req model.PatchStudentRequest) (model.Student, error)
	Delete(ctx context.Context, id int) error
}
 
// Daftar putih kolom yang boleh dipakai untuk ORDER BY.
type studentPostgresRepository struct {
	pool *pgxpool.Pool
}
 
func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}
 
func buildStudentFilter(q model.CursorQuery) (string, []any) {
	where := " WHERE 1=1"
	args := []any{}
 
	if strings.TrimSpace(q.Search) != "" {
		where += fmt.Sprintf(" AND (name ILIKE $%d OR nim ILIKE $%d)", len(args)+1, len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}
 
	if q.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", len(args)+1)
		args = append(args, *q.IsActive)
	}

	if q.After != nil {
		where += fmt.Sprintf(
			" AND (created_at, id) < ($%d, $%d)",
			len(args)+1,
			len(args)+2,
		)

		args = append(args,
			q.After.CreatedAt,
			q.After.ID,
		)
	}
 
	return where, args
}
 
func (r *studentPostgresRepository) FindAll(ctx context.Context, q model.CursorQuery) ([]model.Student, error) {
	where, args := buildStudentFilter(q)
 
	sqlText := fmt.Sprintf(
		`SELECT id, nim, name, grade, is_active, owner_id, created_at
		 FROM students%s
		 ORDER BY created_at DESC, id DESC
		 LIMIT $%d`,
		where,
		len(args)+1,
	)

	args = append(args, q.Limit+1)

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar student: %w", err)
	}
	defer rows.Close()

	list := []model.Student{}

	for rows.Next() {
		var s model.Student

		if err := rows.Scan(
			&s.ID,
			&s.NIM,
			&s.Name,
			&s.Grade,
			&s.IsActive,
			&s.OwnerID,
			&s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("membaca baris student: %w", err)
		}

		list = append(list, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca daftar student: %w", err)
	}

	return list, nil
}
 
func (r *studentPostgresRepository) FindByID(ctx context.Context, id int) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx, `SELECT id, nim, name, grade, is_active, owner_id, created_at FROM students WHERE id = $1`, id).
		Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.OwnerID, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("mengambil student: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) FindOwnerID(ctx context.Context, id int) (*int, error) {
	var ownerID *int
	err := r.pool.QueryRow(ctx, `SELECT owner_id FROM students WHERE id = $1`, id).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("mengambil owner_id student: %w", err)
	}
	return ownerID, nil
}
 
func (r *studentPostgresRepository) Create(ctx context.Context, s model.Student) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO students (nim, name, grade, is_active, owner_id) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		s.NIM, s.Name, s.Grade, s.IsActive, s.OwnerID,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("menyimpan student: %w", err)
	}
	return s, nil
}
 
func (r *studentPostgresRepository) Update(ctx context.Context, s model.Student) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		`UPDATE students SET nim = $1, name = $2, grade = $3, is_active = $4 WHERE id = $5 RETURNING id, nim, name, grade, is_active, owner_id, created_at`,
		s.NIM, s.Name, s.Grade, s.IsActive, s.ID,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.OwnerID, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("memperbarui student: %w", err)
	}
	return s, nil
}
 
func (r *studentPostgresRepository) Patch(ctx context.Context, id int, req model.PatchStudentRequest) (model.Student, error) {
	var s model.Student
 
	err := r.pool.QueryRow(ctx,
		`UPDATE students
		 SET nim = COALESCE($1, nim),
		     name = COALESCE($2, name),
		     grade = COALESCE($3, grade),
		     is_active = COALESCE($4, is_active)
		 WHERE id = $5
		 RETURNING id, nim, name, grade, is_active, owner_id, created_at`,
		req.NIM,
		req.Name,
		req.Grade,
		req.IsActive,
		id,
	).Scan(
		&s.ID,
		&s.NIM,
		&s.Name,
		&s.Grade,
		&s.IsActive,
		&s.OwnerID,
		&s.CreatedAt,
	)
 
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
 
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
 
		return model.Student{}, fmt.Errorf("memperbarui sebagian student: %w", err)
	}
 
	return s, nil
}
 
func (r *studentPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM students WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
 
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}