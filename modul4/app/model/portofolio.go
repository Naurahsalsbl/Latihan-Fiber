package model

import "time"

// Portfolio adalah bentuk portofolio yang dikenal seluruh aplikasi.
//
// Seperti model.Review pada modul, tidak ada tipe milik driver MongoDB di
// sini: ID disimpan sebagai string, bukan bson.ObjectID. Urusan menerjemahkan
// ke BSON adalah tugas repository.
type Portfolio struct {
	ID           string        `json:"id"`
	StudentID    int           `json:"student_id"`
	Headline     string        `json:"headline"`
	Bio          string        `json:"bio"`
	Projects     []Project     `json:"projects"`
	Certificates []Certificate `json:"certificates"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// Project disarangkan (embedded) di dalam dokumen portofolio.
type Project struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	URL          string   `json:"url"`
	Year         int      `json:"year"`
}

// Certificate juga disarangkan. Date hanya berisi tanggal (tanpa jam).
type Certificate struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	Issuer string    `json:"issuer"`
	Date   time.Time `json:"date"`
}

// ProjectInput dipakai oleh POST proyek dan oleh isi array projects pada PUT.
type ProjectInput struct {
	Title        string   `json:"title" validate:"required,min=2,max=100"`
	Technologies []string `json:"technologies" validate:"max=15,dive,min=1,max=30"`
	URL          string   `json:"url" validate:"omitempty,url,max=300"`
	Year         int      `json:"year" validate:"required,min=2000,max=2100"`
}

// CertificateInput menerima tanggal sebagai teks YYYY-MM-DD.
type CertificateInput struct {
	Name   string `json:"name" validate:"required,min=2,max=150"`
	Issuer string `json:"issuer" validate:"required,min=2,max=100"`
	Date   string `json:"date" validate:"required,datetime=2006-01-02"`
}

// PortfolioRequest adalah body PUT /students/:id/portfolio.
// Batas 20 proyek juga ditegakkan di sini agar PUT tidak menjadi jalan
// pintas untuk melewati batas yang dijaga POST proyek.
type PortfolioRequest struct {
	Headline     string             `json:"headline" validate:"required,min=3,max=120"`
	Bio          string             `json:"bio" validate:"max=1000"`
	Projects     []ProjectInput     `json:"projects" validate:"max=20,dive"`
	Certificates []CertificateInput `json:"certificates" validate:"max=30,dive"`
}

// TechnologyCount adalah satu baris hasil statistik teknologi.
type TechnologyCount struct {
	Technology string `json:"technology"`
	Total      int    `json:"total"`
}