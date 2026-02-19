-- Create enum types
CREATE TYPE channel_type AS ENUM ('TEXT', 'AUDIO', 'VIDEO');
CREATE TYPE member_role AS ENUM ('ADMIN', 'MODERATOR', 'GUEST');

-- Users table
CREATE TABLE users (
    id BIGINT PRIMARY KEY,  -- Snowflake ID from app
    username VARCHAR(32) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    image_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- create index on users table
-- Already index on id, username, email

-- user sessions
CREATE TABLE user_sessions (
    id BIGINT PRIMARY KEY,  -- Snowflake ID from app
    user_id       BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token TEXT         NOT NULL,
    device_info   JSONB,                          -- {os, browser, model …}
    ip_address    INET,
    created_at    TIMESTAMPTZ  DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  DEFAULT NOW(),
    expires_at    TIMESTAMPTZ  NOT NULL
);

-- Create index on the users sessions table
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);

-- Servers table
CREATE TABLE servers (
    id BIGINT PRIMARY KEY,  -- Snowflake ID from app
    name VARCHAR(255) NOT NULL,
    image_url TEXT,
    owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_servers_owner_id ON servers(owner_id);


-- Create invites table
CREATE TABLE server_invites (
    code VARCHAR(10) PRIMARY KEY,           -- unique invite code (62^10)
    server_id BIGINT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    max_uses INT,                           -- NULL = unlimited
    used_count INT NOT NULL DEFAULT 0,      
    expires_at TIMESTAMPTZ,                 -- NULL = never expires
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_server_invites_server_id_created ON server_invites(server_id, created_by);  -- server created invites of a user
CREATE INDEX idx_server_invites_expires_at ON server_invites(expires_at) WHERE expires_at IS NOT NULL;  -- for cleanup expired
-- Note: If you need limit on invite creation by the user in the server then use the composite indexing

-- Channels table
CREATE TABLE channels (
    id BIGINT PRIMARY KEY,  -- Snowflake ID from app
    name VARCHAR(255) NOT NULL,
    type channel_type NOT NULL DEFAULT 'TEXT',
    creator_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id BIGINT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_channels_creator_id ON channels(creator_id);
CREATE INDEX idx_channels_server_id_created ON channels(server_id, created_at); -- getting server channels

-- Members table
CREATE TABLE members (
    id BIGINT PRIMARY KEY,  -- Snowflake ID from app
    role member_role NOT NULL DEFAULT 'GUEST',
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id BIGINT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_members_unique_user_server ON members(user_id, server_id); -- uniqueness in table
CREATE INDEX idx_members_server_id ON members(server_id, id);  -- find by server_id order by id(a sorted snowflake id); covering two queries in one index
CREATE INDEX idx_members_user_created_server ON members (user_id, created_at); -- for getting user joined servers

-- CREATE INDEX idx_members_user_id ON members(user_id);  // from unique index on (user_id, server_id)

CREATE TABLE conversations (
    id BIGINT PRIMARY KEY,  -- Snowflake ID from app
    user1_id       BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id       BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ  DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  DEFAULT NOW(),

    -- Inline constraint (enforced in DDL)
    CONSTRAINT check_user_order CHECK (user1_id < user2_id)
);

CREATE UNIQUE INDEX idx_conversations_pair  ON conversations(user1_id, user2_id);
CREATE INDEX idx_conversations_user2_updated ON conversations(user2_id, updated_at DESC);  -- fast user1_id OR user2_id ORDER BY updated_at DESC;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Cleanup expired invites
CREATE OR REPLACE FUNCTION cleanup_expired_server_invites()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM server_invites WHERE expires_at < NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_servers_updated_at BEFORE UPDATE ON servers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_channels_updated_at BEFORE UPDATE ON channels FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_members_updated_at BEFORE UPDATE ON members FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
