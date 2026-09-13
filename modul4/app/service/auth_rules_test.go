package service

import (
	"testing"

	"latihan-fiber/modul4/app/model"
)

// Pengujian ini tidak menyalakan server, tidak menyentuh database,
// dan tidak membuat fiber.Ctx sama sekali.

func TestValidateRegister(t *testing.T) {
	cases := []struct {
		name       string
		req        model.RegisterRequest
		wantFields []string // field yang HARUS muncul di errs
		wantValid  bool
	}{
		{
			name:      "lengkap dan valid",
			req:       model.RegisterRequest{Username: "sari123", Email: "sari@example.com", Password: "rahasia123"},
			wantValid: true,
		},
		{
			name:       "username kosong",
			req:        model.RegisterRequest{Username: "", Email: "sari@example.com", Password: "rahasia123"},
			wantFields: []string{"username"},
		},
		{
			name:       "username kurang dari 3 karakter",
			req:        model.RegisterRequest{Username: "ab", Email: "sari@example.com", Password: "rahasia123"},
			wantFields: []string{"username"},
		},
		{
			name:       "username mengandung karakter tidak valid",
			req:        model.RegisterRequest{Username: "sari nama", Email: "sari@example.com", Password: "rahasia123"},
			wantFields: []string{"username"},
		},
		{
			name:       "email tanpa @",
			req:        model.RegisterRequest{Username: "sari123", Email: "sariexample.com", Password: "rahasia123"},
			wantFields: []string{"email"},
		},
		{
			name:       "password terlalu pendek",
			req:        model.RegisterRequest{Username: "sari123", Email: "sari@example.com", Password: "abc123"},
			wantFields: []string{"password"},
		},
		{
			name:       "password tanpa angka",
			req:        model.RegisterRequest{Username: "sari123", Email: "sari@example.com", Password: "rahasiaaa"},
			wantFields: []string{"password"},
		},
		{
			name:       "password terlalu umum",
			req:        model.RegisterRequest{Username: "sari123", Email: "sari@example.com", Password: "password1"},
			wantFields: []string{"password"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := ValidateRegister(tc.req)
			if tc.wantValid && len(errs) != 0 {
				t.Errorf("ValidateRegister(%+v) = %v, ingin kosong (valid)", tc.req, errs)
			}
			for _, field := range tc.wantFields {
				if _, ok := errs[field]; !ok {
					t.Errorf("ValidateRegister(%+v) = %v, ingin ada error pada field %q", tc.req, errs, field)
				}
			}
		})
	}
}

func TestValidateLogin(t *testing.T) {
	cases := []struct {
		name      string
		req       model.LoginRequest
		wantValid bool
	}{
		{"lengkap", model.LoginRequest{Username: "sari", Password: "apapun"}, true},
		{"username kosong", model.LoginRequest{Username: "", Password: "apapun"}, false},
		{"password kosong", model.LoginRequest{Username: "sari", Password: ""}, false},
		{"password lemah tetap valid di login", model.LoginRequest{Username: "sari", Password: "123"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := ValidateLogin(tc.req)
			gotValid := len(errs) == 0
			if gotValid != tc.wantValid {
				t.Errorf("ValidateLogin(%+v) valid=%v, ingin valid=%v (errs=%v)", tc.req, gotValid, tc.wantValid, errs)
			}
		})
	}
}

func TestCheckPasswordStrength(t *testing.T) {
	cases := []struct {
		password string
		wantMsg  string
	}{
		{"rahasia123", ""},
		{"pendek1", "minimal 8 karakter"},
		{"hanyahuruf", "harus memuat huruf dan angka"},
		{"12345678", "harus memuat huruf dan angka"},
		{"password1", "password terlalu umum"},
	}

	for _, tc := range cases {
		if got := checkPasswordStrength(tc.password); got != tc.wantMsg {
			t.Errorf("checkPasswordStrength(%q) = %q, ingin %q", tc.password, got, tc.wantMsg)
		}
	}
}

func TestIsValidUsername(t *testing.T) {
	cases := []struct {
		username string
		want     bool
	}{
		{"sari123", true},
		{"sari.budi_99", true},
		{"sari budi", false},
		{"sari@budi", false},
	}

	for _, tc := range cases {
		if got := isValidUsername(tc.username); got != tc.want {
			t.Errorf("isValidUsername(%q) = %v, ingin %v", tc.username, got, tc.want)
		}
	}
}

func TestIsValidEmail(t *testing.T) {
	cases := []struct {
		email string
		want  bool
	}{
		{"sari@example.com", true},
		{"sariexample.com", false},
		{"sari@@example.com", false},
		{"sari@example", false},
		{"sari@.com", false},
	}

	for _, tc := range cases {
		if got := isValidEmail(tc.email); got != tc.want {
			t.Errorf("isValidEmail(%q) = %v, ingin %v", tc.email, got, tc.want)
		}
	}
}