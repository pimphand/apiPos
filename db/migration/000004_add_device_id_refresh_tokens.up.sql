-- Add device_id to refresh_tokens for device-bound refresh tokens
ALTER TABLE refresh_tokens
    ADD COLUMN device_id VARCHAR(255) NOT NULL DEFAULT '';

CREATE INDEX idx_refresh_tokens_device_id ON refresh_tokens(device_id);
