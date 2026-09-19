-- =============================================================
-- Permission khusus entitas students
-- =============================================================
INSERT INTO permissions (name, description) VALUES
    ('student:list',        'Melihat daftar seluruh mahasiswa'),
    ('student:read:any',    'Melihat data mahasiswa mana pun, bukan hanya miliknya sendiri'),
    ('student:create',      'Menambahkan data mahasiswa baru'),
    ('student:update:any',  'Mengubah data mahasiswa mana pun'),
    ('student:delete',      'Menghapus data mahasiswa')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'student:list'),
    ('admin', 'student:read:any'),
    ('admin', 'student:create'),
    ('admin', 'student:update:any'),
    ('admin', 'student:delete'),
    ('staff', 'student:list'),
    ('staff', 'student:read:any'),
    ('staff', 'student:create')
ON CONFLICT DO NOTHING;
-- role 'user' sengaja tidak diberi permission student:* apa pun.
-- Haknya atas data mahasiswa mengalir dari kepemilikan (owner_id),
-- bukan dari permission — persis seperti pola user:* pada Modul 6.

-- =============================================================
-- Kolom owner_id — menandai user mana yang mendaftarkan data itu
-- =============================================================
-- Kolom ini SENGAJA dibiarkan boleh NULL (tidak NOT NULL). Tabel students
-- pada project ini sudah berisi data dari pengujian Modul 2-5, dan data
-- tersebut tidak punya pemilik yang bisa ditentukan secara valid — memaksa
-- backfill ke satu user tertentu (misalnya admin) akan memberi kesan palsu
-- bahwa admin "mendaftarkan" data yang sebenarnya dibuat sebelum sistem
-- akun ada sama sekali. Baris lama dibiarkan owner_id = NULL, yang berarti
-- data tersebut hanya dapat diakses lewat permission :any (admin/staff),
-- tidak lewat jalur kepemilikan siapa pun — pilihan yang aman (fail closed),
-- bukan kelalaian.
--
-- ON DELETE SET NULL dipakai (bukan CASCADE) supaya penghapusan akun user
-- tidak ikut menghapus data mahasiswa yang pernah didaftarkannya.
ALTER TABLE students
    ADD COLUMN IF NOT EXISTS owner_id INTEGER REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS students_owner_id_idx ON students (owner_id);