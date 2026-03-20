CREATE TABLE IF NOT EXISTS docker_stacks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    compose_yaml TEXT NOT NULL,
    env_vars JSONB DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'deployed',
    server_id UUID REFERENCES servers(id),
    user_id UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
