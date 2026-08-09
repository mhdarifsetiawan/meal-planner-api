-- Rollback migration 000010
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
