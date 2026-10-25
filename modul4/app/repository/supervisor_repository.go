package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"

	"latihan-fiber/modul4/app/model"
)

var (
	ErrSupervisorNotFound = errors.New("dosen pembimbing tidak ditemukan")
	ErrSupervisorFull     = errors.New("kuota dosen pembimbing penuh")
	ErrNoSupervision      = errors.New("mahasiswa belum memiliki pembimbing")
	ErrSameSupervisor     = errors.New("pembimbing baru sama dengan pembimbing lama")
	ErrSupervisorQuota    = errors.New("kuota lebih kecil dari jumlah mahasiswa bimbingan")
	ErrSupervisorVersion  = errors.New("data pembimbing sudah diubah pihak lain")
)

type SupervisorRepository interface {
	WithTx(tx DBTX) SupervisorRepository

	LockStudent(ctx context.Context, studentID int) error
	FindSupervision(ctx context.Context, studentID int) (int, error)
	LockSupervisor(ctx context.Context, supervisorID int) (model.Supervisor, error)
	IncrementIfAvailable(ctx context.Context, supervisorID int) error
	InsertSupervision(ctx context.Context, studentID, supervisorID int) error
	ChangeSupervision(ctx context.Context, studentID, supervisorID int) error
	DecrementSupervised(ctx context.Context, supervisorID int) error
	UpdateQuota(ctx context.Context, id, quota, version int) (model.Supervisor, error)
	FindByID(ctx context.Context, id int) (model.Supervisor, error)
}

type supervisorPostgresRepository struct {
	db DBTX
}

func NewSupervisorRepository(db DBTX) SupervisorRepository {
	return &supervisorPostgresRepository{db: db}
}

func (r *supervisorPostgresRepository) WithTx(tx DBTX) SupervisorRepository {
	return &supervisorPostgresRepository{db: tx}
}

func scanSupervisor(row pgx.Row) (model.Supervisor, error) {
	var s model.Supervisor
	err := row.Scan(
		&s.ID,
		&s.Name,
		&s.Quota,
		&s.SupervisedCount,
		&s.Version,
	)
	return s, err
}

const supervisorColumns = `
	id, name, quota, supervised_count, version
`

func (r *supervisorPostgresRepository) FindByID(
	ctx context.Context, id int,
) (model.Supervisor, error) {
	s, err := scanSupervisor(r.db.QueryRow(
		ctx,
		`SELECT `+supervisorColumns+` FROM supervisors WHERE id = $1`,
		id,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Supervisor{}, ErrSupervisorNotFound
	}
	if err != nil {
		return model.Supervisor{}, fmt.Errorf("mengambil dosen pembimbing: %w", err)
	}
	return s, nil
}

func (r *supervisorPostgresRepository) LockStudent(
	ctx context.Context, studentID int,
) error {
	var id int
	err := r.db.QueryRow(
		ctx,
		`SELECT id FROM students WHERE id = $1 FOR UPDATE`,
		studentID,
	).Scan(&id)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("mengunci mahasiswa: %w", err)
	}
	return nil
}

func (r *supervisorPostgresRepository) FindSupervision(
	ctx context.Context, studentID int,
) (int, error) {
	var supervisorID int
	err := r.db.QueryRow(
		ctx,
		`SELECT supervisor_id FROM supervisions WHERE student_id = $1`,
		studentID,
	).Scan(&supervisorID)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNoSupervision
	}
	if err != nil {
		return 0, fmt.Errorf("mengambil pembimbing mahasiswa: %w", err)
	}
	return supervisorID, nil
}

func (r *supervisorPostgresRepository) LockSupervisor(
	ctx context.Context, supervisorID int,
) (model.Supervisor, error) {
	s, err := scanSupervisor(r.db.QueryRow(
		ctx,
		`SELECT `+supervisorColumns+`
		 FROM supervisors
		 WHERE id = $1
		 FOR UPDATE`,
		supervisorID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Supervisor{}, ErrSupervisorNotFound
	}
	if err != nil {
		return model.Supervisor{}, fmt.Errorf("mengunci dosen pembimbing: %w", err)
	}
	return s, nil
}

func (r *supervisorPostgresRepository) IncrementIfAvailable(
	ctx context.Context, supervisorID int,
) error {
	tag, err := r.db.Exec(
		ctx,
		`UPDATE supervisors
		 SET supervised_count = supervised_count + 1
		 WHERE id = $1 AND supervised_count < quota`,
		supervisorID,
	)
	if err != nil {
		return fmt.Errorf("menambah jumlah bimbingan: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *supervisorPostgresRepository) InsertSupervision(
	ctx context.Context, studentID, supervisorID int,
) error {
	_, err := r.db.Exec(
		ctx,
		`INSERT INTO supervisions (student_id, supervisor_id)
		 VALUES ($1, $2)`,
		studentID,
		supervisorID,
	)
	if err != nil {
		return fmt.Errorf("menyimpan pembimbing mahasiswa: %w", err)
	}
	return nil
}

func (r *supervisorPostgresRepository) ChangeSupervision(
	ctx context.Context, studentID, supervisorID int,
) error {
	tag, err := r.db.Exec(
		ctx,
		`UPDATE supervisions
		 SET supervisor_id = $1
		 WHERE student_id = $2`,
		supervisorID,
		studentID,
	)
	if err != nil {
		return fmt.Errorf("mengganti pembimbing mahasiswa: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNoSupervision
	}
	return nil
}

func (r *supervisorPostgresRepository) DecrementSupervised(
	ctx context.Context, supervisorID int,
) error {
	tag, err := r.db.Exec(
		ctx,
		`UPDATE supervisors
		 SET supervised_count = supervised_count - 1
		 WHERE id = $1 AND supervised_count > 0`,
		supervisorID,
	)
	if err != nil {
		return fmt.Errorf("mengurangi jumlah bimbingan: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("jumlah bimbingan tidak konsisten untuk supervisor %d", supervisorID)
	}
	return nil
}

func (r *supervisorPostgresRepository) UpdateQuota(
	ctx context.Context, id, quota, version int,
) (model.Supervisor, error) {
	s, err := scanSupervisor(r.db.QueryRow(
		ctx,
		`UPDATE supervisors
		 SET quota = $1, version = version + 1
		 WHERE id = $2
		   AND version = $3
		   AND $1 >= supervised_count
		 RETURNING `+supervisorColumns,
		quota, id, version,
	))
	if err == nil {
		return s, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Supervisor{}, fmt.Errorf("memperbarui kuota pembimbing: %w", err)
	}

	current, findErr := r.FindByID(ctx, id)
	if findErr != nil {
		return model.Supervisor{}, findErr
	}
	if current.Version != version {
		return model.Supervisor{}, ErrSupervisorVersion
	}
	if quota < current.SupervisedCount {
		return model.Supervisor{}, ErrSupervisorQuota
	}
	return model.Supervisor{}, ErrSupervisorVersion
}

// LockSupervisors mengunci semua dosen menurut urutan ID menaik.
// Semua proses switch wajib memakai urutan yang sama.
func LockSupervisors(
	ctx context.Context,
	repo SupervisorRepository,
	ids ...int,
) error {
	unique := make(map[int]bool)
	ordered := make([]int, 0, len(ids))
	for _, id := range ids {
		if !unique[id] {
			unique[id] = true
			ordered = append(ordered, id)
		}
	}
	sort.Ints(ordered)

	for _, id := range ordered {
		if _, err := repo.LockSupervisor(ctx, id); err != nil {
			return err
		}
	}
	return nil
}