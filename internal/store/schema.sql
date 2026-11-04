-- Comptes et sessions. Le rôle admin est prévu pour la gestion des utilisateurs,
-- l'inscription publique crée toujours un client.

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'client' CHECK (role IN ('client', 'admin')),
    confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    confirmation_code TEXT,
    confirmation_expires_at TIMESTAMPTZ,
    reset_code TEXT,
    reset_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions (user_id);
