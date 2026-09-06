package service

import (
	"testing"

	"latihan-fiber/modul4/app/model"
)

// Perhatikan: pengujian ini tidak menyalakan server, tidak menyentuh
// database, dan tidak membuat fiber.Ctx sama sekali.

func TestValidateCreate(t *testing.T) {
	cases := []struct {
		name string
		req  model.Student
		want string
	}{
		{"lengkap dan valid", model.Student{NIM: "12345", Name: "Sari", Grade: 88}, ""},
		{"nim kosong", model.Student{NIM: "", Name: "Sari", Grade: 88}, "NIM, nama, dan grade wajib diisi"},
		{"name kosong", model.Student{NIM: "12345", Name: "", Grade: 88}, "NIM, nama, dan grade wajib diisi"},
		{"grade negatif", model.Student{NIM: "12345", Name: "Sari", Grade: -1}, "grade harus di antara 0 dan 100"},
		{"grade lebih dari 100", model.Student{NIM: "12345", Name: "Sari", Grade: 101}, "grade harus di antara 0 dan 100"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateCreate(tc.req); got != tc.want {
				t.Errorf("ValidateCreate(%+v) = %q, ingin %q", tc.req, got, tc.want)
			}
		})
	}
}

func TestValidateReplace(t *testing.T) {
	cases := []struct {
		name string
		req  model.Student
		want string
	}{
		{"lengkap dan valid", model.Student{NIM: "12345", Name: "Sari", Grade: 90}, ""},
		{"name kosong", model.Student{NIM: "12345", Name: "", Grade: 90}, "NIM, nama, dan grade wajib diisi"},
		{"grade di luar rentang", model.Student{NIM: "12345", Name: "Sari", Grade: 150}, "grade harus di antara 0 dan 100"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateReplace(tc.req); got != tc.want {
				t.Errorf("ValidateReplace(%+v) = %q, ingin %q", tc.req, got, tc.want)
			}
		})
	}
}

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