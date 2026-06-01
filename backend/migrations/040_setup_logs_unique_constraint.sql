-- Add unique constraint on (server_id, module) so that
-- ON CONFLICT DO NOTHING in setup_handler.go works correctly.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'uq_setup_logs_server_module'
    ) THEN
        ALTER TABLE setup_logs
            ADD CONSTRAINT uq_setup_logs_server_module UNIQUE (server_id, module);
    END IF;
END $$;
