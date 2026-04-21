-- Add webhook_secret to applications for GitHub/GitLab webhook authentication
ALTER TABLE applications ADD COLUMN IF NOT EXISTS webhook_secret VARCHAR(255) DEFAULT '';
