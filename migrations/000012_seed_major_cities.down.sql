-- Rollback Migration 012
DELETE FROM cities WHERE id > 1;
DELETE FROM provinces WHERE id > 1;
