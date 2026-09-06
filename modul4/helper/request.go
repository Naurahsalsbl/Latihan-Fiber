package helper

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/modul4/app/model"
)

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