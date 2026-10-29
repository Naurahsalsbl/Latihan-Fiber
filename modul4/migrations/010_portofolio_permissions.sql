-- Migration 008: permission pengelolaan portofolio
-- psql -U postgres -d praktikum_backend -f migrations/008_portfolio_permissions.sql
--
-- Portofolionya disimpan di MongoDB, tetapi hak aksesnya tetap dikelola di
-- PostgreSQL bersama permission lain. Satu sistem hak akses untuk seluruh
-- aplikasi, apa pun database tempat datanya disimpan.
--
-- Pemilik data mahasiswa (students.owner_id) boleh mengubah portofolionya
-- sendiri tanpa permission apa pun; permission ini untuk mengubah portofolio
-- mahasiswa MANA PUN.
INSERT INTO permissions (name, description) VALUES
    ('portfolio:manage', 'Mengubah portofolio mahasiswa mana pun')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'portfolio:manage')
ON CONFLICT DO NOTHING;