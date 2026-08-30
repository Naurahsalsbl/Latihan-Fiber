# API Students

REST API untuk mengelola data mahasiswa menggunakan Go, Fiber v2, dan PostgreSQL.

Data mahasiswa disimpan secara permanen di PostgreSQL dan akses database dikelola menggunakan Repository Pattern.

## Fitur

- Menampilkan seluruh data mahasiswa
- Menampilkan data mahasiswa berdasarkan ID
- Menambahkan data mahasiswa
- Mengubah seluruh data mahasiswa
- Mengubah sebagian data mahasiswa
- Menghapus data mahasiswa
- Pencarian mahasiswa berdasarkan nama atau NIM
- Filter berdasarkan status aktif
- Sorting data mahasiswa
- Pagination
- Pemeriksaan koneksi server dan database

## Cara Menjalankan

Pastikan PostgreSQL sudah berjalan dan konfigurasi database pada file `.env` sudah sesuai.

go mod tidy
go run .

Server berjalan pada:

http://localhost:3000

## Struktur Berkas

| Berkas | Fungsi |
|---------|---------|
| main.go | Konfigurasi aplikasi, koneksi database, dan routing |
| handler.go | Menangani request dan response endpoint API |
| helper.go | Fungsi helper untuk response API dan context request |
| app/model/student.go | Struct Student, response, metadata, dan query parameter |
| app/repository/student_repository.go | Proses CRUD dan query data mahasiswa ke PostgreSQL |
| config/env.go | Mengatur konfigurasi environment aplikasi |
| database/postgres.go | Membuat koneksi dan connection pool PostgreSQL |
| migrations/001_create_students.sql | Membuat tabel students pada database |

## Struktur Data Mahasiswa

{
  "id": 1,
  "nim": "20260001",
  "name": "Naura",
  "grade": 92,
  "is_active": true,
  "created_at": "2026-08-25T10:00:00Z"
}

## Database

Database yang digunakan:

pbl_prak

Tabel yang digunakan:

students

Struktur tabel:

| Kolom | Tipe Data | Keterangan |
|----------|----------|----------|
| id | SERIAL | ID mahasiswa |
| nim | VARCHAR | Nomor Induk Mahasiswa |
| name | VARCHAR | Nama mahasiswa |
| grade | DOUBLE PRECISION | Nilai mahasiswa |
| is_active | BOOLEAN | Status aktif mahasiswa |
| created_at | TIMESTAMP | Waktu data dibuat |

## Endpoint API

| Method | Endpoint | Keterangan |
|----------|----------|----------|
| GET | /api/v1/health | Mengecek status server dan database |
| GET | /api/v1/students | Menampilkan daftar mahasiswa |
| GET | /api/v1/students/:id | Menampilkan mahasiswa berdasarkan ID |
| POST | /api/v1/students | Menambahkan mahasiswa |
| PUT | /api/v1/students/:id | Mengganti seluruh data mahasiswa |
| PATCH | /api/v1/students/:id | Mengubah sebagian data mahasiswa |
| DELETE | /api/v1/students/:id | Menghapus data mahasiswa |

## Query Parameter

Endpoint:

GET /api/v1/students

Mendukung parameter berikut:

| Parameter | Fungsi |
|------------|---------|
| page | Nomor halaman |
| limit | Jumlah data per halaman |
| search | Pencarian berdasarkan nama atau NIM |
| sort | Pengurutan data |
| order | Urutan data asc atau desc |
| is_active | Filter berdasarkan status aktif |

Contoh:

GET /api/v1/students?page=1&limit=5&search=naura&sort=grade&order=desc

## Contoh Request

### POST Mahasiswa

POST /api/v1/students

Body:

{
  "nim": "120210004",
  "name": "Naura",
  "grade": 90,
  "is_active": true
}

### PUT Mahasiswa

PUT /api/v1/students/1

Body:

{
  "nim": "20260003",
  "name": "Dina Ramadhan",
  "grade": 85,
  "is_active": false
}

### PATCH Mahasiswa

PATCH /api/v1/students/1

Body:

{
  "grade": 90
}

## Status HTTP

| Status | Keterangan |
|----------|----------|
| 200 OK | Request berhasil |
| 201 Created | Data berhasil ditambahkan |
| 204 No Content | Data berhasil dihapus |
| 400 Bad Request | Request tidak valid |
| 404 Not Found | Data tidak ditemukan |
| 409 Conflict | NIM sudah digunakan |
| 500 Internal Server Error | Terjadi kesalahan pada server |

## Catatan

- NIM digunakan sebagai identitas unik mahasiswa.
- Data mahasiswa disimpan di PostgreSQL sehingga tetap tersedia meskipun server dihentikan.
- Endpoint POST mengembalikan status 201 Created ketika data berhasil ditambahkan.
- Endpoint PUT digunakan untuk mengganti seluruh data mahasiswa.
- Endpoint PATCH hanya mengubah field yang dikirim oleh client.
- Endpoint GET mendukung pencarian, filter, sorting, dan pagination.
- Endpoint DELETE digunakan untuk menghapus data mahasiswa dari PostgreSQL.
- Endpoint /api/v1/health digunakan untuk memastikan server dan database dapat terhubung.
- File .env digunakan untuk menyimpan konfigurasi database dan tidak di-upload ke repository.
- File .env.example digunakan sebagai contoh konfigurasi environment.

## Repository Pattern

Pada Modul 3, proses akses database dipisahkan menggunakan Repository Pattern.

Alur aplikasi:

Client
↓
Handler
↓
Repository
↓
PostgreSQL
↓
Repository
↓
Handler
↓
Client

Repository bertanggung jawab terhadap proses pengambilan, penambahan, perubahan, dan penghapusan data mahasiswa pada PostgreSQL.

## Perbedaan Modul 2 dan Modul 3

| Modul 2 | Modul 3 |
|----------|----------|
| Data disimpan di memory | Data disimpan di PostgreSQL |
| Menggunakan slice | Menggunakan database |
| Tidak menggunakan Repository Pattern | Menggunakan Repository Pattern |
| Data hilang ketika server dihentikan | Data tetap tersimpan di database |
| CRUD dilakukan pada data di memory | CRUD dilakukan melalui query PostgreSQL |

## Teknologi

- Golang
- Fiber v2
- PostgreSQL
- pgx
- godotenv
- Repository Pattern
- REST API

