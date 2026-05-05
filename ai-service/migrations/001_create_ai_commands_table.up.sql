-- ai_commands stores natural-language commands sent through ai-service,
-- their LLM-derived operation plans, lifecycle status and per-op execution
-- results. user_id is owned by auth-service — there is no FK from this table.

CREATE TYPE command_status AS ENUM (
    'awaiting_confirmation',
    'executed',
    'failed',
    'cancelled',
    'expired'
);

CREATE TABLE ai_commands (
    id             UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID           NOT NULL,
    input          TEXT           NOT NULL,
    plan_ops       JSONB          NOT NULL DEFAULT '[]'::jsonb,
    explanation    TEXT           NOT NULL DEFAULT '',
    status         command_status NOT NULL DEFAULT 'awaiting_confirmation',
    llm_model      VARCHAR(256)   NOT NULL DEFAULT '',
    llm_tokens_in  INT            NOT NULL DEFAULT 0,
    llm_tokens_out INT            NOT NULL DEFAULT 0,
    results        JSONB          NULL,
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    expires_at     TIMESTAMPTZ    NOT NULL,
    executed_at    TIMESTAMPTZ    NULL,
    cancelled_at   TIMESTAMPTZ    NULL
);

-- Hot lookup: user dashboard ("show my recent commands by status").
CREATE INDEX idx_ai_commands_user_status
    ON ai_commands (user_id, status, created_at DESC);

-- Janitor lookup: only awaiting_confirmation rows that may need expiring.
CREATE INDEX idx_ai_commands_expires_pending
    ON ai_commands (expires_at)
    WHERE status = 'awaiting_confirmation';
