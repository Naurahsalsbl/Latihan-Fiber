package service

import "latihan-fiber/modul4/app/model"

// IsEmptyPatch menandai permintaan PATCH yang tidak mengubah field apa pun.
func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.NIM == nil && 
		req.Name == nil && 
		req.Grade == nil && 
		req.IsActive == nil
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