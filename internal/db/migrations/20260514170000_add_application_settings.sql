-- +goose Up
-- +goose StatementBegin
CREATE TABLE thunderdome.application_settings (
    id smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    registration_open boolean NOT NULL DEFAULT true,
    updated_by uuid REFERENCES thunderdome.users(id) ON DELETE SET NULL,
    updated_date timestamp with time zone NOT NULL DEFAULT now()
);
INSERT INTO thunderdome.application_settings (id) VALUES (1);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE thunderdome.application_settings;
-- +goose StatementEnd
