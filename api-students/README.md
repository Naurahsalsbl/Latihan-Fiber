# API Students

REST API sederhana untuk mengelola data mahasiswa menggunakan Golang dan Fiber.

## Fitur

- Menampilkan seluruh data mahasiswa
- Menampilkan data mahasiswa berdasarkan ID
- Menambahkan data mahasiswa
- Mengubah seluruh data mahasiswa
- Mengubah sebagian data mahasiswa
- Menghapus data mahasiswa

## Struktur Data

```json
{
  "id": 1,
  "nim": "120210004",
  "name": "Naura",
  "grade": 92,
  "is_active": true
}
```

## Menjalankan Program

Clone repository:

```bash
git clone <url-repository>
```

Masuk ke folder project:

```bash
cd api-students
```

Install dependency:

```bash
go mod tidy
```

Jalankan aplikasi:

```bash
go run .
```

Server berjalan pada:

```text
http://localhost:3000
```

## Endpoint API

### GET Semua Mahasiswa

```http
GET /api/v1/students
```

Response:

```json
[
  {
    "id": 1,
    "nim": "120210004",
    "name": "Naura",
    "grade": 92,
    "is_active": true
  }
]
```

---

### GET Mahasiswa Berdasarkan ID

```http
GET /api/v1/students/:id
```

Contoh:

```http
GET /api/v1/students/1
```

---

### POST Mahasiswa

```http
POST /api/v1/students
```

Body:

```json
{
  "nim": "120210006",
  "name": "Bima",
  "grade": 88
}
```

---

### PUT Mahasiswa

```http
PUT /api/v1/students/:id
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

---

### PATCH Mahasiswa

```http
PATCH /api/v1/students/:id
```

Body:

```json
{
  "grade": 90
}
```

---

### DELETE Mahasiswa

```http
DELETE /api/v1/students/:id
```

## Status Code

| Status | Keterangan |
|----------|----------|
| 200 OK | Request berhasil |
| 201 Created | Data berhasil dibuat |
| 204 No Content | Data berhasil dihapus |
| 400 Bad Request | Request tidak valid |
| 404 Not Found | Data tidak ditemukan |
| 409 Conflict | Data duplikat |
| 415 Unsupported Media Type | Format request tidak didukung |
| 422 Unprocessable Entity | Validasi gagal |

## Teknologi

- Golang
- Fiber Framework

## Referensi

- Modul Praktikum REST API dan HTTP Deep Dive
- Dokumentasi Fiber