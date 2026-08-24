# API Students

REST API sederhana untuk mengelola data mahasiswa menggunakan Go dan Fiber v2.

Data disimpan sementara di memori (slice), tanpa menggunakan database.

## Fitur

- Menampilkan seluruh data mahasiswa
- Menampilkan data mahasiswa berdasarkan ID
- Menambahkan data mahasiswa
- Mengubah seluruh data mahasiswa
- Mengubah sebagian data mahasiswa
- Menghapus data mahasiswa

## Cara Menjalankan

```bash
go mod tidy
go run .
```

Server berjalan pada:

```text
http://localhost:3000
```

## Struktur Berkas

| Berkas | Fungsi |
|---------|---------|
| main.go | Konfigurasi aplikasi, middleware, dan routing |
| model.go | Struct Student, request, response, dan metadata |
| helper.go | Fungsi helper untuk response dan query string |
| handler.go | Logika endpoint CRUD mahasiswa |

## Struktur Data Mahasiswa

```json
{
  "id": 1,
  "nim": "230411100001",
  "name": "Naura",
  "grade": 92,
  "is_active": true,
  "created_at": "2026-08-25T10:00:00Z"
}
```

## Endpoint API

| Method | Endpoint | Keterangan |
|----------|----------|----------|
| GET | `/api/v1/health` | Mengecek status server |
| GET | `/api/v1/students` | Menampilkan daftar mahasiswa |
| GET | `/api/v1/students/:id` | Menampilkan mahasiswa berdasarkan ID |
| POST | `/api/v1/students` | Menambahkan mahasiswa |
| PUT | `/api/v1/students/:id` | Mengganti seluruh data mahasiswa |
| PATCH | `/api/v1/students/:id` | Mengubah sebagian data mahasiswa |
| DELETE | `/api/v1/students/:id` | Menghapus mahasiswa |

## Query Parameter

Endpoint:

```http
GET /api/v1/students
```

Mendukung parameter berikut:

| Parameter | Fungsi |
|------------|---------|
| page | Nomor halaman |
| limit | Jumlah data per halaman |
| search | Pencarian berdasarkan nama atau NIM |
| sort | Pengurutan data |
| order | asc atau desc |
| is_active | Filter status aktif |
| min_grade | Nilai minimum |
| max_grade | Nilai maksimum |

Contoh:

```http
GET /api/v1/students?page=1&limit=5&search=bima&sort=grade&order=desc
```

## Contoh Request

### POST Mahasiswa

```http
POST /api/v1/students
```

Body:

```json
{
  "nim": "120210004",
  "name": "Naura",
  "grade": 90
}
```

### PUT Mahasiswa

```http
PUT /api/v1/students/1
```

Body:

```json
{
  "nim": "120210006",
  "name": "Bima Ramadhan",
  "grade": 80,
  "is_active": false
}
```

### PATCH Mahasiswa

```http
PATCH /api/v1/students/1
```

Body:

```json
{
  "grade": 90
}
```

## Status HTTP

| Status | Keterangan |
|----------|----------|
| 200 OK | Request berhasil |
| 201 Created | Data berhasil ditambahkan |
| 204 No Content | Data berhasil dihapus |
| 400 Bad Request | Request tidak valid |
| 404 Not Found | Data tidak ditemukan |
| 409 Conflict | NIM sudah digunakan |
| 415 Unsupported Media Type | Content-Type bukan JSON |
| 422 Unprocessable Entity | Validasi data gagal |

## Catatan

- NIM digunakan sebagai identitas unik mahasiswa.
- Data disimpan di memori sehingga akan hilang ketika server dihentikan.
- Endpoint POST mengembalikan header `Location` yang menunjuk ke data mahasiswa yang baru dibuat.
- Endpoint PATCH hanya mengubah field yang dikirim oleh client.
- Endpoint GET mendukung pencarian, filter, sorting, dan pagination.

## Teknologi

- Golang
- Fiber v2

## Referensi

- Modul Praktikum REST API dan HTTP Deep Dive
- Dokumentasi Fiber