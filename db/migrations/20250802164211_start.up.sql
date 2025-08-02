-- users table
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(100) UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login TIMESTAMP
);


-- personal_chats table
CREATE TABLE personal_chats (
    id BIGSERIAL PRIMARY KEY,
    user1_id BIGINT NOT NULL REFERENCES users(id),
    user2_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_personal_chat UNIQUE (user1_id, user2_id)
);

-- group_chats table
CREATE TABLE group_chats (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    owner_id BIGINT REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    is_global BOOLEAN NOT NULL DEFAULT FALSE,
    network_identifier VARCHAR(255)
);

-- group_members table
CREATE TABLE group_members (
    group_id BIGINT REFERENCES group_chats(id),
    user_id BIGINT REFERENCES users(id),
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    role VARCHAR(20) NOT NULL DEFAULT 'member',
    PRIMARY KEY (group_id, user_id)
);

-- messages table
CREATE TABLE messages (
    id BIGSERIAL PRIMARY KEY,
    sender_id BIGINT NOT NULL REFERENCES users(id),
    chat_type VARCHAR(10) NOT NULL CHECK (chat_type IN ('personal', 'group', 'global')),
    chat_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
    read_by BIGINT[]
);
