-- Add jti column to refresh_tokens to store JWT Token ID (jti claim).
-- Previously SaveRefreshToken returned the DB-generated UUID as id,
-- but IsRefreshTokenValid and RevokeRefreshToken queried by JWT jti
-- (a nanosecond timestamp string), which never matched the UUID column.
-- This migration adds a dedicated jti column and index to fix the mismatch.

ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS jti VARCHAR(64) NOT NULL DEFAULT '';

-- Fill existing rows with a placeholder — existing tokens were already broken,
-- so a placeholder is acceptable. New rows will always have the real jti set.

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_jti ON refresh_tokens (jti);
