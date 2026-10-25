-- Migration 007: 60 akun mahasiswa untuk uji beban
-- HANYA untuk database praktikum.
-- Jangan dijalankan di server sungguhan.

INSERT INTO users (username, email, password, role)
SELECT
    'mhs' || LPAD(g::TEXT, 3, '0'),
    'mhs' || LPAD(g::TEXT, 3, '0') || '@student.unair.ac.id',
    '!',
    'user'
FROM generate_series(1, 60) AS g
ON CONFLICT DO NOTHING;
