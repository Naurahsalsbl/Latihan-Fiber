-- Migration 008: penetapan dosen pembimbing tugas akhir

CREATE TABLE IF NOT EXISTS supervisors (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(120) NOT NULL,
    quota           INTEGER NOT NULL CHECK (quota > 0),
    supervised_count INTEGER NOT NULL DEFAULT 0
        CHECK (supervised_count >= 0 AND supervised_count <= quota),
    version         INTEGER NOT NULL DEFAULT 1 CHECK (version > 0)
);

CREATE TABLE IF NOT EXISTS supervisions (
    student_id   INTEGER PRIMARY KEY
        REFERENCES students(id) ON DELETE CASCADE,
    supervisor_id INTEGER NOT NULL
        REFERENCES supervisors(id) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS supervisions_supervisor_id_idx
    ON supervisions (supervisor_id);

-- Permission untuk mengelola penetapan pembimbing
INSERT INTO permissions (name, description)
VALUES (
    'supervision:manage',
    'Menetapkan dan mengganti dosen pembimbing serta mengubah kuota'
)
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name)
VALUES ('admin', 'supervision:manage')
ON CONFLICT DO NOTHING;