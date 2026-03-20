DROP TABLE IF EXISTS domain_backend_servers;
ALTER TABLE domains DROP COLUMN IF EXISTS is_load_balancer;
