package service

import (
	"testing"

	"latihan-fiber/modul4/app/model"
	"latihan-fiber/modul4/helper"
)

// Pengujian ini tidak menyalakan server, tidak menyentuh database,
// dan tidak membuat fiber.Ctx sama sekali — persis seperti pengujian
// business rules lain pada project ini.

func newTestPermissions() *helper.PermissionSet {
	return helper.NewPermissionSet(map[string][]string{
		"admin": {"student:read:any", "student:update:any"},
		"staff": {"student:read:any"},
		"user":  {},
	})
}

func TestCanAccessStudent(t *testing.T) {
	perms := newTestPermissions()

	cases := []struct {
		name          string
		current       model.AuthUser
		ownerID       int
		anyPermission string
		want          bool
	}{
		{
			name:          "pemilik data sendiri, role user tanpa permission apa pun",
			current:       model.AuthUser{UserID: 5, Role: "user"},
			ownerID:       5,
			anyPermission: "student:read:any",
			want:          true,
		},
		{
			name:          "bukan pemilik, role user tanpa permission",
			current:       model.AuthUser{UserID: 5, Role: "user"},
			ownerID:       9,
			anyPermission: "student:read:any",
			want:          false,
		},
		{
			name:          "bukan pemilik, tapi role staff punya permission :any",
			current:       model.AuthUser{UserID: 5, Role: "staff"},
			ownerID:       9,
			anyPermission: "student:read:any",
			want:          true,
		},
		{
			name:          "staff punya read:any tapi bukan update:any",
			current:       model.AuthUser{UserID: 5, Role: "staff"},
			ownerID:       9,
			anyPermission: "student:update:any",
			want:          false,
		},
		{
			name:          "data tanpa pemilik (ownerID 0), role user",
			current:       model.AuthUser{UserID: 5, Role: "user"},
			ownerID:       0,
			anyPermission: "student:read:any",
			want:          false,
		},
		{
			name:          "data tanpa pemilik (ownerID 0), role admin",
			current:       model.AuthUser{UserID: 5, Role: "admin"},
			ownerID:       0,
			anyPermission: "student:read:any",
			want:          true,
		},
		{
			name:          "role tidak dikenal selalu ditolak (fail closed)",
			current:       model.AuthUser{UserID: 5, Role: "siluman"},
			ownerID:       9,
			anyPermission: "student:read:any",
			want:          false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CanAccessStudent(tc.current, tc.ownerID, perms, tc.anyPermission)
			if got != tc.want {
				t.Errorf("CanAccessStudent(%+v, %d, _, %q) = %v, ingin %v",
					tc.current, tc.ownerID, tc.anyPermission, got, tc.want)
			}
		})
	}
}

func TestCanAccessStudent_PermissionSetNil(t *testing.T) {
	// PermissionSet nil harus tetap fail closed, bukan panic.
	current := model.AuthUser{UserID: 5, Role: "admin"}
	if CanAccessStudent(current, 9, nil, "student:read:any") {
		t.Error("PermissionSet nil seharusnya selalu menolak, bukan mengizinkan")
	}
}