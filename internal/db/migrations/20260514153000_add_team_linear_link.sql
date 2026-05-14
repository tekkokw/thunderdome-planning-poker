-- +goose Up
-- +goose StatementBegin
CREATE TABLE thunderdome.team_linear_link (
    team_id uuid NOT NULL PRIMARY KEY REFERENCES thunderdome.team(id) ON DELETE CASCADE,
    linear_instance_id uuid NOT NULL REFERENCES thunderdome.linear_instance(id) ON DELETE CASCADE,
    linear_team_id text NOT NULL,
    linear_team_key text NOT NULL,
    linear_team_name text NOT NULL DEFAULT '',
    created_date timestamp with time zone NOT NULL DEFAULT now(),
    updated_date timestamp with time zone NOT NULL DEFAULT now()
);
CREATE INDEX team_linear_link_instance_id_idx ON thunderdome.team_linear_link (linear_instance_id);

ALTER TABLE thunderdome.team_checkin ADD COLUMN linear_cycle_id text NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE thunderdome.team_checkin DROP COLUMN linear_cycle_id;
DROP TABLE thunderdome.team_linear_link;
-- +goose StatementEnd
