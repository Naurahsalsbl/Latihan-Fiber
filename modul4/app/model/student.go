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
	OwnerID	  *int		`json:"owner_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
 
// PatchStudentRequest menampung body PATCH. Field pointer supaya
// "tidak dikirim" (nil) dapat dibedakan dari "dikirim nilai kosong".
type PatchStudentRequest struct {
	NIM      *string  `json:"nim,omitempty" validate:"omitnil,required,nim"`
	Name     *string  `json:"name,omitempty" validate:"omitnil,min=1"`
	Grade    *float64 `json:"grade,omitempty" validate:"omitnil,gte=0,lte=100"`
	IsActive *bool    `json:"is_active,omitempty"`
}
 
// WebResponse adalah amplop baku untuk seluruh respons API.
type WebResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    any  `json:"meta,omitempty"`
	Errors	any	   `json:"errors,omitempty"`
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

// ErrorResponse adalah bentuk BAKU untuk seluruh respons kegagalan,
// dihasilkan hanya oleh ErrorHandler terpusat di config/app.go.
type ErrorResponse struct {
	Success   bool              `json:"success"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

// POST - semua field wajib
type CreateStudentRequest struct {
	NIM   string  `json:"nim" validate:"required,nim"`
	Name  string  `json:"name" validate:"required"`
	Grade float64 `json:"grade" validate:"required,gte=0,lte=100"`
}

// PUT - semua field wajib
type ReplaceStudentRequest struct {
	NIM      string  `json:"nim" validate:"required,nim"`
	Name     string  `json:"name" validate:"required"`
	Grade    float64 `json:"grade" validate:"required,gte=0,lte=100"`
	IsActive bool    `json:"is_active"`
}

