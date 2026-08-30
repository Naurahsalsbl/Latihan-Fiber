package model

import "time"

type Student struct {
	ID        int       `json:"id"`
	NIM       string    `json:"nim"`
	Name      string    `json:"name"`
	Grade     float64   `json:"grade"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Request khusus untuk PATCH
type PatchStudentRequest struct {
	NIM      *string  `json:"nim,omitempty"`
	Name     *string  `json:"name,omitempty"`
	Grade    *float64 `json:"grade,omitempty"`
	IsActive *bool    `json:"is_active,omitempty"`
}

type WebResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta   `json:"meta,omitempty"`
}

type Meta struct {
	Page      int `json:"page"`
	Limit     int `json:"limit"`
	TotalData int `json:"total_data"`
	TotalPage int `json:"total_page"`
}

type ListQuery struct {
	Page     int    `query:"page"`
	Limit    int    `query:"limit"`
	Sort     string `query:"sort"`
	Order    string `query:"order"`
	Search   string `query:"search"`
	IsActive *bool  `query:"is_active"`
}

func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}