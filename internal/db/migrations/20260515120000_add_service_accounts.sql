-- +goose Up
-- +goose StatementBegin
ALTER TABLE thunderdome.users
    ADD COLUMN is_service_account boolean NOT NULL DEFAULT false;

-- posted_by_user_id records who actually submitted a checkin when it differs
-- from the subject (user_id). NULL means self-submitted. Used for agent /
-- service-account attribution.
ALTER TABLE thunderdome.team_checkin
    ADD COLUMN posted_by_user_id uuid REFERENCES thunderdome.users(id) ON DELETE SET NULL;

CREATE INDEX users_is_service_account_idx
    ON thunderdome.users (is_service_account) WHERE is_service_account = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX thunderdome.users_is_service_account_idx;
ALTER TABLE thunderdome.team_checkin DROP COLUMN posted_by_user_id;
ALTER TABLE thunderdome.users DROP COLUMN is_service_account;
-- +goose StatementEnd
