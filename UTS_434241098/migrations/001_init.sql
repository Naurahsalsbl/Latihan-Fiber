CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'mahasiswa')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE students (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE REFERENCES users(id),
    nim VARCHAR(12) NOT NULL UNIQUE,
    nama VARCHAR(150) NOT NULL,
    prodi VARCHAR(100) NOT NULL,
    angkatan INTEGER NOT NULL,
    ipk_terakhir NUMERIC(3,2),
    deleted_at TIMESTAMP NULL,

    CONSTRAINT students_ipk_check
        CHECK (
            ipk_terakhir IS NULL
            OR (ipk_terakhir >= 0 AND ipk_terakhir <= 4)
        )
);

CREATE TABLE courses (
    id SERIAL PRIMARY KEY,
    kode_mk VARCHAR(20) NOT NULL UNIQUE,
    nama_mk VARCHAR(150) NOT NULL,
    sks INTEGER NOT NULL CHECK (sks > 0),
    semester INTEGER NOT NULL CHECK (semester > 0),
    kuota INTEGER NOT NULL CHECK (kuota >= 0)
);

CREATE TABLE enrollments (
    id SERIAL PRIMARY KEY,
    student_id INTEGER NOT NULL REFERENCES students(id),
    course_id INTEGER NOT NULL REFERENCES courses(id),
    tahun_akademik VARCHAR(30) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_enrollment
        UNIQUE (student_id, course_id, tahun_akademik)
);

CREATE INDEX idx_students_prodi
    ON students(prodi);

CREATE INDEX idx_students_angkatan
    ON students(angkatan);

CREATE INDEX idx_enrollments_student
    ON enrollments(student_id);

CREATE INDEX idx_enrollments_course
    ON enrollments(course_id);