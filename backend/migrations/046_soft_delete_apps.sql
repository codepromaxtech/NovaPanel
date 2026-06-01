-- Add soft-delete support to applications
ALTER TABLE applications
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_applications_deleted ON applications(deleted_at)
  WHERE deleted_at IS NULL;
