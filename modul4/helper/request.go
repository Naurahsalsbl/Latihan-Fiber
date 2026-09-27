package helper

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/modul4/app/model"
)

// RequestID membaca ID unik yang dipasang middleware requestid.New()
// pada setiap request. Dipusatkan di sini supaya ErrorHandler dan
// RequestLogger membaca sumber yang sama, bukan menyalin string literal
// "requestid" di beberapa tempat.
func RequestID(c *fiber.Ctx) string {
	id, _ := c.Locals("requestid").(string)
	return id
}

// RequestContext memberi timeout untuk setiap operasi database,
// supaya satu request yang lambat tidak menggantung selamanya.
func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

// ParamID membaca parameter :id dari jalur dan memastikan bentuknya benar.
func ParamID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// ParseListQuery membaca query string dan memberi nilai bawaan yang aman.
// Perilakunya dipertahankan persis sama seperti kode lama: memakai
// c.QueryParser, lalu membatasi page dan limit ke rentang yang wajar.
// Whitelist kolom sort sendiri ditegakkan di app/repository (lihat
// kolomUrutStudent), bukan di sini — sesuai kode aslinya.
func ParseListQuery(c *fiber.Ctx) (model.ListQuery, error) {
	var q model.ListQuery
	if err := c.QueryParser(&q); err != nil {
		return q, err
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 10
	}

	return q, nil
}

func ParseCursorQuery(c *fiber.Ctx) (model.CursorQuery, error) {
	var q model.CursorQuery

	limit := c.QueryInt("limit", 10)
	if limit < 1 || limit > 100 {
		return q, BadRequest("limit harus antara 1 dan 100")
	}
	q.Limit = limit

	q.Search = strings.TrimSpace(c.Query("search"))

	if active := c.Context().QueryArgs().Peek("is_active"); len(active) > 0 {
		value := c.Query("is_active")

		switch value {
		case "true":
			v := true
			q.IsActive = &v
		case "false":
			v := false
			q.IsActive = &v
		default:
			return q, BadRequest("is_active harus bernilai true atau false")
		}
	}

	if after := strings.TrimSpace(c.Query("after")); after != "" {
		cursor, err := DecodeCursor(after)
		if err != nil {
			return q, BadRequest("cursor tidak valid")
		}
		q.After = &cursor
	}

	return q, nil
}