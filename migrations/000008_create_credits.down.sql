-- Migration 008: Credits system (rollback)

DROP TABLE IF EXISTS credit_transactions;
DROP TABLE IF EXISTS user_credits;
