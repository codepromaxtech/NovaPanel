-- Add unique constraint on (server_id, module) so that
-- ON CONFLICT DO NOTHING in setup_handler.go works correctly.
ALTER TABLE setup_logs
    ADD CONSTRAINT IF NOT EXISTS uq_setup_logs_server_module UNIQUE (server_id, module);
