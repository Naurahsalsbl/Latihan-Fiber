-- ADMIN
INSERT INTO users (email, password, role)
VALUES (
    'admin@siakad.local',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llCq3fQf6sR4G2R8H9P7G',
    'admin'
);

-- MAHASISWA
INSERT INTO users (email, password, role)
SELECT
    'mahasiswa' || i || '@siakad.local',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llCq3fQf6sR4G2R8H9P7G',
    'mahasiswa'
FROM generate_series(1, 20) AS i;

-- DATA MAHASISWA
INSERT INTO students
    (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
SELECT
    id,
    '187221000' || LPAD((id - 1)::text, 3, '0'),
    'Mahasiswa ' || (id - 1),
    CASE
        WHEN id % 3 = 0 THEN 'Teknik Informatika'
        WHEN id % 3 = 1 THEN 'Sistem Informasi'
        ELSE 'Teknik Komputer'
    END,
    2022,
    CASE
        WHEN id % 3 = 0 THEN 3.50
        WHEN id % 3 = 1 THEN 2.90
        ELSE 2.30
    END
FROM users
WHERE role = 'mahasiswa'
ORDER BY id;

-- MATA KULIAH
INSERT INTO courses
    (kode_mk, nama_mk, sks, semester, kuota)
VALUES
    ('IF101', 'Algoritma dan Pemrograman', 3, 1, 30),
    ('IF102', 'Basis Data', 3, 2, 30),
    ('IF103', 'Pemrograman Web', 3, 3, 25),
    ('IF104', 'Struktur Data', 3, 2, 30),
    ('IF105', 'Jaringan Komputer', 3, 4, 25),
    ('IF106', 'Sistem Operasi', 3, 4, 25),
    ('IF107', 'Kecerdasan Buatan', 3, 5, 20),
    ('IF108', 'Rekayasa Perangkat Lunak', 3, 5, 25),
    ('IF109', 'Keamanan Informasi', 3, 6, 20),
    ('IF110', 'Pemrograman Mobile', 3, 6, 20);