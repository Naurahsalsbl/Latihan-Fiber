package service

import "latihan-fiber/modul4/app/model"

// File ini berisi business rules MURNI: tidak mengimpor Fiber,
// tidak mengimpor repository, dan tidak mengetahui apa pun tentang HTTP.
// Karena itu seluruh fungsi di sini dapat diuji hanya dengan memanggilnya
// langsung, tanpa menyalakan server maupun database (lihat student_rules_test.go).
//
// Pesan dan urutan pemeriksaan di bawah ini SENGAJA dipertahankan persis
// sama seperti pada handler lama (Pertemuan 3), agar perilaku HTTP-nya
// tidak berubah setelah dipindahkan.

// ValidateCreate memeriksa isi permintaan POST.
// Mengembalikan string kosong bila lolos, atau pesan kesalahan pertama
// yang ditemukan (mengikuti urutan pemeriksaan pada kode asli).
func ValidateCreate(req model.Student) string {
	if req.NIM == "" || req.Name == "" {
		return "NIM, nama, dan grade wajib diisi"
	}
	if req.Grade < 0 || req.Grade > 100 {
		return "grade harus di antara 0 dan 100"
	}
	return ""
}

// ValidateReplace memeriksa isi permintaan PUT.
// Aturannya sama dengan ValidateCreate karena PUT mewajibkan
// seluruh field tetap terisi.
func ValidateReplace(req model.Student) string {
	if req.NIM == "" || req.Name == "" {
		return "NIM, nama, dan grade wajib diisi"
	}
	if req.Grade < 0 || req.Grade > 100 {
		return "grade harus di antara 0 dan 100"
	}
	return ""
}

// IsEmptyPatch menandai permintaan PATCH yang tidak mengubah field apa pun.
func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil
}

// ValidatePatch memeriksa isi permintaan PATCH. Hanya field yang dikirim
// (tidak nil) yang diperiksa; field yang tidak dikirim dilewati begitu saja.
// Penerapan nilainya sendiri dilakukan oleh repository lewat COALESCE di SQL,
// sehingga fungsi ini berperan sebagai gerbang validasi sebelum data
// diteruskan ke lapisan penyimpanan.
func ValidatePatch(req model.PatchStudentRequest) string {
	if IsEmptyPatch(req) {
		return "tidak ada field yang diubah"
	}
	if req.NIM != nil && *req.NIM == "" {
		return "NIM tidak boleh kosong"
	}
	if req.Name != nil && *req.Name == "" {
		return "nama tidak boleh kosong"
	}
	if req.Grade != nil && (*req.Grade < 0 || *req.Grade > 100) {
		return "grade harus di antara 0 dan 100"
	}
	return ""
}

// CountTotalPage membulatkan ke atas tanpa memakai bilangan pecahan.
// Mengembalikan 0 bila tidak ada data sama sekali.
func CountTotalPage(total, limit int) int {
	if total <= 0 || limit <= 0 {
		return 0
	}
	return (total + limit - 1) / limit
}