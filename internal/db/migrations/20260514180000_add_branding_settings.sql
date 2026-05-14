-- +goose Up
-- +goose StatementBegin
CREATE TABLE thunderdome.branding_settings (
    id smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    brand_name text NOT NULL DEFAULT '',
    primary_color text NOT NULL DEFAULT '',
    accent_color text NOT NULL DEFAULT '',
    dark_color text NOT NULL DEFAULT '',
    logo_main_data bytea,
    logo_main_content_type text,
    logo_dark_data bytea,
    logo_dark_content_type text,
    favicon_data bytea,
    favicon_content_type text,
    email_logo_data bytea,
    email_logo_content_type text,
    updated_by uuid REFERENCES thunderdome.users(id) ON DELETE SET NULL,
    updated_date timestamp with time zone NOT NULL DEFAULT now()
);
INSERT INTO thunderdome.branding_settings (id) VALUES (1);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE thunderdome.branding_settings;
-- +goose StatementEnd
