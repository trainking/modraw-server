CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ==============================
-- users
-- ==============================
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nickname      VARCHAR(100) NOT NULL DEFAULT '',
    avatar_url    TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- ==============================
-- folders (self-referencing tree)
-- ==============================
CREATE TABLE IF NOT EXISTS folders (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    parent_id  UUID         REFERENCES folders(id) ON DELETE CASCADE,
    sort_order INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_folders_user   ON folders (user_id);
CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders (parent_id);

-- ==============================
-- canvases
-- ==============================
CREATE TABLE IF NOT EXISTS canvases (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    folder_id    UUID         REFERENCES folders(id) ON DELETE SET NULL,
    name         VARCHAR(255) NOT NULL DEFAULT 'Untitled',
    data         JSONB        NOT NULL DEFAULT '{}'::jsonb,
    thumbnail    TEXT         NOT NULL DEFAULT '',
    file_size    BIGINT       NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_canvases_owner  ON canvases (owner_id);
CREATE INDEX IF NOT EXISTS idx_canvases_folder ON canvases (folder_id);

-- ==============================
-- libraries (asset .mdrlib)
-- ==============================
CREATE TABLE IF NOT EXISTS libraries (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    data        JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_libraries_owner ON libraries (owner_id);

-- ==============================
-- share_links
-- ==============================
CREATE TABLE IF NOT EXISTS share_links (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    canvas_id     UUID         NOT NULL REFERENCES canvases(id) ON DELETE CASCADE,
    created_by    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission    VARCHAR(20)  NOT NULL DEFAULT 'readonly'
                               CHECK (permission IN ('readonly','collaborate')),
    password_hash VARCHAR(255) NOT NULL DEFAULT '',
    code          VARCHAR(20)  NOT NULL UNIQUE,
    expires_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_share_links_canvas ON share_links (canvas_id);
CREATE INDEX IF NOT EXISTS idx_share_links_code   ON share_links (code);

-- ==============================
-- canvas_collaborators
-- ==============================
CREATE TABLE IF NOT EXISTS canvas_collaborators (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    canvas_id  UUID        NOT NULL REFERENCES canvases(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR(20) NOT NULL DEFAULT 'readonly'
                           CHECK (permission IN ('readonly','collaborate')),
    added_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (canvas_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_collab_canvas ON canvas_collaborators (canvas_id);
CREATE INDEX IF NOT EXISTS idx_collab_user   ON canvas_collaborators (user_id);

-- ==============================
-- refresh_tokens
-- ==============================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens (user_id);
