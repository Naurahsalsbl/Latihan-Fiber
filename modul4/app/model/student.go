package model

import "time"
 
// Student adalah entitas inti (Layer 1 - Entities). Tidak mengetahui
// apa pun tentang HTTP, database, atau framework yang dipakai.
type Student struct {
	ID        int       `json:"id"`
	NIM       string    `json:"nim"`
	Name      string    `json:"name"`
	Grade     float64   `json:"grade"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}
 
// PatchStudentRequest menampung body PATCH. Field pointer supaya
// "tidak dikirim" (nil) dapat dibedakan dari "dikirim nilai kosong".
type PatchStudentRequest struct {
	NIM      *string  `json:"nim,omitempty"`
	Name     *string  `json:"name,omitempty"`
	Grade    *float64 `json:"grade,omitempty"`
	IsActive *bool    `json:"is_active,omitempty"`
}
 
// WebResponse adalah amplop baku untuk seluruh respons API.
type WebResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}
 
type Meta struct {
	Page      int `json:"page"`
	Limit     int `json:"limit"`
	TotalData int `json:"total_data"`
	TotalPage int `json:"total_page"`
}
 
// ListQuery menampung hasil pembacaan query string pada endpoint daftar.
type ListQuery struct {
	Page     int    `query:"page"`
	Limit    int    `query:"limit"`
	Sort     string `query:"sort"`
	Order    string `query:"order"`
	Search   string `query:"search"`
	IsActive *bool  `query:"is_active"`
}
 
// Offset menghitung baris yang dilompati berdasarkan halaman dan limit.
func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}