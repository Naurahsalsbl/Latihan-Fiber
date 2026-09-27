package service

import (
	"testing"

	"latihan-fiber/modul4/app/model"
)

// Perhatikan: pengujian ini tidak menyalakan server, tidak menyentuh
// database, dan tidak membuat fiber.Ctx sama sekali.

func TestValidatePatch(t *testing.T) {
	nimKosong := ""
	nimIsi := "12345"
	gradeInvalid := 150.0
	gradeValid := 95.0

	cases := []struct {
		name string
		req  model.PatchStudentRequest
		want string
	}{
		{"tidak ada field", model.PatchStudentRequest{}, "tidak ada field yang diubah"},
		{"nim dikirim kosong", model.PatchStudentRequest{NIM: &nimKosong}, "NIM tidak boleh kosong"},
		{"grade di luar rentang", model.PatchStudentRequest{Grade: &gradeInvalid}, "grade harus di antara 0 dan 100"},
		{"hanya grade valid", model.PatchStudentRequest{Grade: &gradeValid}, ""},
		{"nim valid", model.PatchStudentRequest{NIM: &nimIsi}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidatePatch(tc.req); got != tc.want {
				t.Errorf("ValidatePatch(%+v) = %q, ingin %q", tc.req, got, tc.want)
			}
		})
	}
}

func TestIsEmptyPatch(t *testing.T) {
	grade := 90.0

	if !IsEmptyPatch(model.PatchStudentRequest{}) {
		t.Error("request kosong seharusnya dianggap empty patch")
	}
	if IsEmptyPatch(model.PatchStudentRequest{Grade: &grade}) {
		t.Error("request dengan satu field terisi seharusnya bukan empty patch")
	}
}

func TestCountTotalPage(t *testing.T) {
	cases := []struct{ total, limit, want int }{
		{0, 10, 0},
		{1, 10, 1},
		{10, 10, 1},
		{11, 10, 2},
		{137, 20, 7},
	}

	for _, tc := range cases {
		if got := CountTotalPage(tc.total, tc.limit); got != tc.want {
			t.Errorf("CountTotalPage(%d, %d) = %d, ingin %d", tc.total, tc.limit, got, tc.want)
		}
	}
}