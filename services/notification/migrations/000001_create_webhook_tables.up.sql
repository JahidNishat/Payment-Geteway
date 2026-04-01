CREATE TABLE IF NOT EXISTS webhook_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    payment_id VARCHAR(255) NOT NULL,
    merchant_id VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    request_body JSONB,
    response_body JSONB,
    status_code INT,
    error TEXT,
    attempt INT NOT NULL DEFAULT 1,
    duration_ms BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_webhook_attempts_payment_id ON webhook_attempts (payment_id);
CREATE INDEX IF NOT EXISTS idx_webhook_attempts_merchant_id ON webhook_attempts (merchant_id);
CREATE INDEX IF NOT EXISTS idx_webhook_attempts_success ON webhook_attempts (success);
