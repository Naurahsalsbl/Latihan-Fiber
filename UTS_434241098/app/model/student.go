package model

import "time"

type Student struct {
	ID           int        `json:"id"`
	UserID       int        `json:"user_id"`
	NIM          string     `json:"nim"`
	Nama         string     `json:"nama"`
	Prodi        string     `json:"prodi"`
	Angkatan     int        `json:"angkatan"`
	IPKTerakhir  *float64   `json:"ipk_terakhir"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type CreateStudentRequest struct {
	NIM         string   `json:"nim" validate:"required,len=12,numeric"`
	Nama        string   `json:"nama" validate:"required"`
	Email       string   `json:"email" validate:"required,email"`
	Prodi       string   `json:"prodi" validate:"required"`
	Angkatan    int      `json:"angkatan" validate:"required,gte=2000,lte=2100"`
	IPKTerakhir *float64 `json:"ipk_terakhir" validate:"omitempty,gte=0,lte=4"`
}

type UpdateStudentRequest struct {
	Nama         string   `json:"nama" validate:"required"`
	Prodi        string   `json:"prodi" validate:"required"`
	Angkatan     int      `json:"angkatan" validate:"required,len=4"`
	IPKTerakhir  *float64 `json:"ipk_terakhir" validate:"omitempty,gte=0,lte=4"`
}

