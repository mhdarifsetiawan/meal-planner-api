-- Migration 000010: Admin seed template
-- ============================================================
-- HOW TO INSERT AN ADMIN USER MANUALLY:
--
-- 1. The admin must first login via Google OAuth in the admin app
--    so that Supabase creates their auth.users entry.
--
-- 2. Run the following SQL in Supabase SQL Editor or psql,
--    replacing the values with the actual admin's data:
--
--    INSERT INTO users (id, email, name, role, created_at)
--    VALUES (
--      '<supabase-auth-uuid>',     -- copy from auth.users.id in Supabase Dashboard
--      'admin@yourcompany.com',
--      'Admin Name',
--      'admin',
--      NOW()
--    )
--    ON CONFLICT (id) DO UPDATE
--      SET role = 'admin';         -- promote existing user to admin
--
-- 3. To demote an admin back to user:
--    UPDATE users SET role = 'user' WHERE email = 'admin@yourcompany.com';
--
-- ============================================================
-- This migration itself adds a check constraint to enforce role values.

ALTER TABLE users
  ADD CONSTRAINT users_role_check
  CHECK (role IN ('user', 'admin'));
