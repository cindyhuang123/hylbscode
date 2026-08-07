-- +goose Up
-- +goose StatementBegin
-- Historical session costs were accumulated in each model's native currency:
-- USD for foreign providers, CNY for DeepSeek/GLM. The GUI used to display
-- cost * CnyRate. Normalize existing costs to CNY (default rate 6.7777,
-- matching config's cnyRate default) so the GUI can show ¥ directly.
-- Sessions whose messages only use CNY-priced models keep their cost as-is.
-- Mixed-model sessions are converted as USD (the common case).
UPDATE sessions
SET cost = cost * 6.7777
WHERE id IN (
    SELECT DISTINCT session_id FROM messages
    WHERE model IS NOT NULL AND model != ''
      AND model NOT LIKE 'deepseek-%'
      AND model NOT LIKE 'glm-%'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE sessions
SET cost = cost / 6.7777
WHERE id IN (
    SELECT DISTINCT session_id FROM messages
    WHERE model IS NOT NULL AND model != ''
      AND model NOT LIKE 'deepseek-%'
      AND model NOT LIKE 'glm-%'
);
-- +goose StatementEnd
