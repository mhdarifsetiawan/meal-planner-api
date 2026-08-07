-- Migration 006: Subscription & monetization (rollback)

DROP TABLE IF EXISTS payment_transactions;
DROP TABLE IF EXISTS user_subscriptions;
DROP TABLE IF EXISTS coupons;
DROP TABLE IF EXISTS subscription_plans;
