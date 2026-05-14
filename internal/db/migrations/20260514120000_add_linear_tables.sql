-- +goose Up
-- +goose StatementBegin
CREATE TABLE thunderdome.linear_instance (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES thunderdome.users(id) ON DELETE CASCADE,
    label text NOT NULL,
    workspace_url_key text NOT NULL DEFAULT '',
    access_token text NOT NULL,
    created_date timestamp with time zone NOT NULL DEFAULT now(),
    updated_date timestamp with time zone NOT NULL DEFAULT now()
);
CREATE INDEX linear_instance_user_id_idx ON thunderdome.linear_instance (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE thunderdome.linear_instance;
-- +goose StatementEnd
