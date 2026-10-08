package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"UTS_434241098/app/model"
)

type CourseRepository struct {
	DB *pgxpool.Pool
}

func NewCourseRepository(db *pgxpool.Pool) *CourseRepository {
	return &CourseRepository{DB: db}
}

func (r *CourseRepository) List(
	ctx context.Context,
	semester int,
	search string,
	available bool,
) ([]model.Course, error) {

	query := `
		SELECT
			c.id,
			c.kode_mk,
			c.nama_mk,
			c.sks,
			c.semester,
			c.kuota,
			COUNT(e.id) AS terisi,
			c.kuota - COUNT(e.id) AS sisa_kuota
		FROM courses c
		LEFT JOIN enrollments e
			ON e.course_id = c.id
			AND e.tahun_akademik = '2026/2027-Ganjil'
		WHERE 1=1
	`

	args := []any{}
	arg := 1

	if semester > 0 {
		query += fmt.Sprintf(" AND c.semester = $%d", arg)
		args = append(args, semester)
		arg++
	}

	if search != "" {
		query += fmt.Sprintf(
			" AND (c.kode_mk ILIKE $%d OR c.nama_mk ILIKE $%d)",
			arg,
			arg,
		)
		args = append(args, "%"+search+"%")
		arg++
	}

	query += `
		GROUP BY
			c.id,
			c.kode_mk,
			c.nama_mk,
			c.sks,
			c.semester,
			c.kuota
	`

	if available {
		query += " HAVING COUNT(e.id) < c.kuota"
	}

	query += " ORDER BY c.id"

	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	courses := make([]model.Course, 0)

	for rows.Next() {
		var course model.Course

		if err := rows.Scan(
			&course.ID,
			&course.KodeMK,
			&course.NamaMK,
			&course.SKS,
			&course.Semester,
			&course.Kuota,
			&course.Terisi,
			&course.SisaKuota,
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

func (r *CourseRepository) FindByID(ctx context.Context, id int) (*model.Course, error) {
	var course model.Course

	err := r.DB.QueryRow(ctx, `
		SELECT
			c.id,
			c.kode_mk,
			c.nama_mk,
			c.sks,
			c.semester,
			c.kuota,
			COUNT(e.id) AS terisi,
			c.kuota - COUNT(e.id) AS sisa_kuota
		FROM courses c
		LEFT JOIN enrollments e ON e.course_id = c.id
		WHERE c.id = $1
		GROUP BY
			c.id,
			c.kode_mk,
			c.nama_mk,
			c.sks,
			c.semester,
			c.kuota
	`, id).Scan(
		&course.ID,
		&course.KodeMK,
		&course.NamaMK,
		&course.SKS,
		&course.Semester,
		&course.Kuota,
		&course.Terisi,
		&course.SisaKuota,
	)

	if err != nil {
		return nil, err
	}

	return &course, nil
}