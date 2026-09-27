package helper

import (
	"github.com/gofiber/fiber/v2"

	"latihan-fiber/modul4/app/model"
)

// Success mengirim respons sukses dengan status HTTP yang diberikan.
// Status dibuat sebagai parameter (bukan tetap 200) karena dipakai baik
// untuk 200 (get/list/replace/patch/delete) maupun 201 (create).
func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(model.WebResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// SuccessList mengirim respons sukses berupa daftar data disertai info paginasi.
func SuccessList(c *fiber.Ctx, message string, data any, meta model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Status:  "success",
		Message: message,
		Data:    data,
		Meta:    &meta,
	})
}

func SuccessCursor(
	c *fiber.Ctx,
	message string,
	data any,
	meta *model.CursorMeta,
) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Status:  "success",
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Created mengirim respons 201 disertai header Location.
// BARU pada Pertemuan 5, dipakai oleh AuthService.Register.
// Tidak dipakai oleh StudentService (perilakunya sengaja tidak diubah).
func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// Fail mengirim respons gagal dengan status HTTP yang sesuai situasi.
func Fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.WebResponse{
		Status:  "fail",
		Message: message,
	})
}

// FailValidation mengirim 422 disertai rincian error per field.
// BARU pada Pertemuan 5, dipakai oleh AuthService (register, login).
func FailValidation(c *fiber.Ctx, errs map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(model.WebResponse{
		Status:  "fail",
		Message: "validasi gagal",
		Errors:  errs,
	})
}