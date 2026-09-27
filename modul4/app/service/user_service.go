
package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/modul4/app/model"
	"latihan-fiber/modul4/app/repository"
	"latihan-fiber/modul4/helper"
)

type UserService struct {
	repo  repository.UserRepository
	perms *helper.PermissionSet
}

func NewUserService(
	repo repository.UserRepository,
	perms *helper.PermissionSet,
) *UserService {
	return &UserService{
		repo:  repo,
		perms: perms,
	}
}

func CanAccessUser(
	current model.AuthUser,
	targetID int,
	perms *helper.PermissionSet,
	permission string,
) bool {
	if current.UserID == targetID {
		return true
	}

	return perms.Can(current.Role, permission)
}

func (s *UserService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	format, err := helper.Negotiate(
		c,
		helper.FormatJSON,
		helper.FormatCSV,
	)
	if err != nil {
		return err
	}

	q, err := helper.ParseCursorQuery(c)
	if err != nil {
		return err
	}

	rows, err := s.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	if format == helper.FormatCSV {
		return helper.WriteUsersCSV(c, rows)
	}

	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit]
	}

	meta := &model.CursorMeta{
		Limit:   q.Limit,
		HasMore: hasMore,
	}

	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(
			last.CreatedAt,
			last.ID,
		)
	}

	return helper.SuccessCursor(
		c,
		"daftar user berhasil diambil",
		rows,
		meta,
	)
}

func (s *UserService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if !CanAccessUser(
		current,
		id,
		s.perms,
		"user:read:any",
	) {
		return helper.Forbidden("tidak berhak mengakses data user lain")
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateUserError(err, "user")
	}

	return helper.Success(
		c,
		fiber.StatusOK,
		"user ditemukan",
		user,
	)
}

// translateError mengubah error milik repository menjadi AppError.
func translateUserError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("username sudah dipakai")
	default:
		return helper.Internal(err)
	}
}


