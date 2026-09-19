-- =============================================================
-- roles — daftar role yang diakui sistem
-- =============================================================
CREATE TABLE IF NOT EXISTS roles (
    name        VARCHAR(20) PRIMARY KEY,
    description VARCHAR(150) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO roles (name, description) VALUES
    ('admin', 'Akses penuh terhadap seluruh data dan pengaturan'),
    ('staff', 'Boleh melihat dan menambah data, tetapi tidak semua boleh mengubah/menghapus'),
    ('user',  'Hanya boleh mengelola datanya sendiri')
ON CONFLICT (name) DO NOTHING;

-- =============================================================
-- permissions — daftar tindakan yang dapat diberikan kepada role
-- =============================================================
CREATE TABLE IF NOT EXISTS permissions (
    name        VARCHAR(50) PRIMARY KEY,
    description VARCHAR(150) NOT NULL
);

-- =============================================================
-- role_permissions — tabel penghubung, inti dari model RBAC
-- PRIMARY KEY gabungan mencegah satu role menerima permission yang
-- sama dua kali; aturan ini dijaga database, bukan diingat kode.
-- =============================================================
CREATE TABLE IF NOT EXISTS role_permissions (
    role_name       VARCHAR(20) NOT NULL REFERENCES roles(name) ON DELETE CASCADE,
    permission_name VARCHAR(50) NOT NULL REFERENCES permissions(name) ON DELETE CASCADE,
    PRIMARY KEY (role_name, permission_name)
);

-- Baris UPDATE ini wajib dijalankan SEBELUM foreign key di bawah dipasang,
-- karena kolom users.role pada project ini sebelumnya adalah VARCHAR bebas
-- (dari Modul 5), dan mungkin berisi nilai yang belum terdaftar di tabel
-- roles. Foreign key akan gagal dipasang bila masih ada baris yang melanggar.
UPDATE users SET role = 'user' WHERE role NOT IN (SELECT name FROM roles);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_fkey;
ALTER TABLE users
    ADD CONSTRAINT users_role_fkey
    FOREIGN KEY (role) REFERENCES roles(name) ON UPDATE CASCADE;

CREATE INDEX IF NOT EXISTS users_role_idx ON users (role);