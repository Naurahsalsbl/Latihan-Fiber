package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"UTS_434241098/app/model"
)

type EnrollmentRepository struct {
	DB *pgxpool.Pool
}

var (
	ErrDuplicateEnrollment = errors.New("duplicate enrollment")
	ErrQuotaFull           = errors.New("quota full")
	ErrSKSLimit            = errors.New("sks limit")
	ErrEnrollmentForbidden = errors.New("enrollment milik mahasiswa lain")
)

func NewEnrollmentRepository(db *pgxpool.Pool) *EnrollmentRepository {
	return &EnrollmentRepository{DB: db}
}

func (r *EnrollmentRepository) Create(
	ctx context.Context,
	studentID int,
	courseID int,
	tahunAkademik string,
) (*model.Enrollment, error) {

	var enrollment model.Enrollment

	err := r.DB.QueryRow(ctx, `
		INSERT INTO enrollments
			(student_id, course_id, tahun_akademik)
		VALUES
			($1, $2, $3)
		RETURNING id, student_id, course_id, tahun_akademik
	`,
		studentID,
		courseID,
		tahunAkademik,
	).Scan(
		&enrollment.ID,
		&enrollment.StudentID,
		&enrollment.CourseID,
		&enrollment.TahunAkademik,
	)

	if err != nil {
		return nil, err
	}

	return &enrollment, nil
}

func (r *EnrollmentRepository) Delete(
	ctx context.Context,
	id int,
	studentID int,
) error {

	var ownerStudentID int

	err := r.DB.QueryRow(ctx, `
		SELECT student_id
		FROM enrollments
		WHERE id = $1
	`, id).Scan(&ownerStudentID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return err
	}

	if ownerStudentID != studentID {
		return ErrEnrollmentForbidden
	}

	_, err = r.DB.Exec(ctx, `
		DELETE FROM enrollments
		WHERE id = $1
	`, id)

	return err
}

func (r *EnrollmentRepository) FindDuplicate(
	ctx context.Context,
	studentID int,
	courseID int,
	tahunAkademik string,
) (bool, error) {

	var exists bool

	err := r.DB.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM enrollments
			WHERE student_id = $1
			AND course_id = $2
			AND tahun_akademik = $3
		)
	`,
		studentID,
		courseID,
		tahunAkademik,
	).Scan(&exists)

	return exists, err
}

func (r *EnrollmentRepository) CountSKS(
	ctx context.Context,
	studentID int,
	tahunAkademik string,
) (int, error) {

	var total int

	err := r.DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(c.sks), 0)
		FROM enrollments e
		JOIN courses c ON c.id = e.course_id
		WHERE e.student_id = $1
		AND e.tahun_akademik = $2
	`,
		studentID,
		tahunAkademik,
	).Scan(&total)

	return total, err
}

func (r *EnrollmentRepository) ListByStudent(
	ctx context.Context,
	studentID int,
) ([]model.StudentCourse, error) {

	rows, err := r.DB.Query(ctx, `
		SELECT
			c.id,
			c.kode_mk,
			c.nama_mk,
			c.sks,
			e.tahun_akademik
		FROM enrollments e
		JOIN courses c ON c.id = e.course_id
		WHERE e.student_id = $1
		ORDER BY e.id
	`, studentID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	courses := make([]model.StudentCourse, 0)

	for rows.Next() {
		var course model.StudentCourse

		if err := rows.Scan(
			&course.ID,
			&course.KodeMK,
			&course.NamaMK,
			&course.SKS,
			&course.TahunAkademik,
		); err != nil {
			return nil, err
		}

		courses = append(courses, course)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return courses, nil
}

func (r *EnrollmentRepository) CountCourseEnrollment(
	ctx context.Context,
	courseID int,
	tahunAkademik string,
) (int, error) {

	var total int

	err := r.DB.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM enrollments
		WHERE course_id = $1
		AND tahun_akademik = $2
	`,
		courseID,
		tahunAkademik,
	).Scan(&total)

	return total, err
}

func (r *EnrollmentRepository) CreateWithTransaction(
	ctx context.Context,
	studentID int,
	courseID int,
	tahunAkademik string,
	maxSKS int,
) (*model.Enrollment, error) {

	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Lock baris course supaya pengecekan kuota aman
	var kuota int
	var sks int

	err = tx.QueryRow(ctx, `
		SELECT kuota, sks
		FROM courses
		WHERE id = $1
		FOR UPDATE
	`, courseID).Scan(&kuota, &sks)

	if err != nil {
		return nil, err
	}

	// Cek duplicate
	var exists bool

	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM enrollments
			WHERE student_id = $1
			AND course_id = $2
			AND tahun_akademik = $3
		)
	`,
		studentID,
		courseID,
		tahunAkademik,
	).Scan(&exists)

	if err != nil {
		return nil, err
	}

	if exists {
		return nil, ErrDuplicateEnrollment
	}

	// Cek kuota
	var terisi int

	err = tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM enrollments
		WHERE course_id = $1
		AND tahun_akademik = $2
	`,
		courseID,
		tahunAkademik,
	).Scan(&terisi)

	if err != nil {
		return nil, err
	}

	if terisi >= kuota {
		return nil, ErrQuotaFull
	}

	// Cek total SKS mahasiswa
	var currentSKS int

	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(c.sks), 0)
		FROM enrollments e
		JOIN courses c ON c.id = e.course_id
		WHERE e.student_id = $1
		AND e.tahun_akademik = $2
	`,
		studentID,
		tahunAkademik,
	).Scan(&currentSKS)

	if err != nil {
		return nil, err
	}

	if currentSKS+sks > maxSKS {
		sisaSKS := maxSKS - currentSKS

		if sisaSKS < 0 {
			sisaSKS = 0
		}

	return nil, &SKSLimitRepositoryError{
		SisaSKS: sisaSKS,
	}
}

	// Insert enrollment
	var enrollment model.Enrollment

	err = tx.QueryRow(ctx, `
		INSERT INTO enrollments
			(student_id, course_id, tahun_akademik)
		VALUES
			($1, $2, $3)
		RETURNING id, student_id, course_id, tahun_akademik
	`,
		studentID,
		courseID,
		tahunAkademik,
	).Scan(
		&enrollment.ID,
		&enrollment.StudentID,
		&enrollment.CourseID,
		&enrollment.TahunAkademik,
	)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &enrollment, nil
}

type SKSLimitRepositoryError struct {
	SisaSKS int
}

func (e *SKSLimitRepositoryError) Error() string {
	return fmt.Sprintf(
		"%s: %d",
		ErrSKSLimit.Error(),
		e.SisaSKS,
	)
}