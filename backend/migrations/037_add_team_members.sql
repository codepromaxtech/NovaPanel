-- Team management: invite secondary users with scoped access
CREATE TABLE IF NOT EXISTS team_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    member_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'viewer',
    scope_type VARCHAR(20) NOT NULL DEFAULT 'all',
    scope_id UUID,
    invited_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted_at TIMESTAMPTZ,
    UNIQUE(owner_id, member_id, scope_type, scope_id)
);

CREATE INDEX IF NOT EXISTS idx_team_members_owner ON team_members(owner_id);
CREATE INDEX IF NOT EXISTS idx_team_members_member ON team_members(member_id);
