-- Migration 007: Community Price Watch (rollback)

DROP INDEX IF EXISTS idx_price_submissions_consensus;
DROP TABLE IF EXISTS price_submissions;
DROP TABLE IF EXISTS price_watch_items;
DROP TABLE IF EXISTS price_watch_campaigns;
