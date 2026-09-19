package service

import (
	"latihan-fiber/modul4/app/model"
	"latihan-fiber/modul4/helper"
)

// CanAccessStudent memutuskan apakah seseorang boleh menyentuh satu baris
// data mahasiswa tertentu. Fungsi ini TIDAK mengimpor fiber maupun
// repository, sehingga dapat diuji hanya dengan memanggilnya langsung
// (lihat student_authz_rules_test.go).
//
// Dua jalur yang diizinkan:
//  1. Kepemilikan (ownership) — data itu didaftarkan oleh dirinya sendiri.
//  2. Permission — role-nya memang berhak atas data mahasiswa siapa pun.
//
// Urutannya disengaja: pemeriksaan kepemilikan didahulukan karena paling
// murah (tidak perlu membaca PermissionSet) dan paling sering benar pada
// kasus pemakaian yang wajar. Bila keduanya gagal, jawabannya false —
// fail closed, sama seperti PermissionSet.Can.
//
// ownerID bernilai 0 dipakai sebagai penanda "data ini tidak memiliki
// pemilik" (baris lama sebelum kolom owner_id ditambahkan, owner_id-nya
// NULL di database). Karena ID pengguna asli selalu dimulai dari 1,
// current.UserID tidak akan pernah cocok dengan 0, sehingga data tanpa
// pemilik otomatis hanya bisa diakses lewat permission :any — tidak
// pernah lewat jalur kepemilikan siapa pun.
func CanAccessStudent(
	current model.AuthUser,
	ownerID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	if ownerID != 0 && current.UserID == ownerID {
		return true
	}
	return perms.Can(current.Role, anyPermission)
}