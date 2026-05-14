package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/StevenWeathers/thunderdome-planning-poker/thunderdome"
)

// ErrLogoNotFound is returned by GetLogo when the requested slot is empty.
var ErrLogoNotFound = errors.New("branding logo not found")

const brandingColumns = `brand_name, primary_color, accent_color, dark_color,
	(logo_main_data IS NOT NULL) AS has_logo_main,
	(logo_dark_data IS NOT NULL) AS has_logo_dark,
	(favicon_data IS NOT NULL) AS has_favicon,
	(email_logo_data IS NOT NULL) AS has_email_logo,
	updated_by, updated_date`

// GetBranding returns the singleton branding row (without logo bytes).
func (d *Service) GetBranding(ctx context.Context) (thunderdome.Branding, error) {
	b := thunderdome.Branding{}
	var updatedBy sql.NullString
	err := d.DB.QueryRowContext(ctx,
		`SELECT `+brandingColumns+` FROM thunderdome.branding_settings WHERE id = 1;`,
	).Scan(
		&b.BrandName, &b.PrimaryColor, &b.AccentColor, &b.DarkColor,
		&b.HasLogoMain, &b.HasLogoDark, &b.HasFavicon, &b.HasEmailLogo,
		&updatedBy, &b.UpdatedDate,
	)
	if err != nil {
		return b, fmt.Errorf("get branding: %v", err)
	}
	if updatedBy.Valid {
		v := updatedBy.String
		b.UpdatedBy = &v
	}
	return b, nil
}

// UpdateBrandingMeta updates just the text/color fields. Pass nil for any
// field to leave it unchanged; pass a non-nil pointer to set it.
func (d *Service) UpdateBrandingMeta(ctx context.Context, actorUserID string, brandName, primaryColor, accentColor, darkColor *string) (thunderdome.Branding, error) {
	_, err := d.DB.ExecContext(ctx,
		`UPDATE thunderdome.branding_settings SET
			brand_name      = COALESCE($1, brand_name),
			primary_color   = COALESCE($2, primary_color),
			accent_color    = COALESCE($3, accent_color),
			dark_color      = COALESCE($4, dark_color),
			updated_by      = $5,
			updated_date    = now()
		WHERE id = 1;`,
		brandName, primaryColor, accentColor, darkColor, actorUserID,
	)
	if err != nil {
		return thunderdome.Branding{}, fmt.Errorf("update branding meta: %v", err)
	}
	return d.GetBranding(ctx)
}

// SetLogo replaces the bytes + content type for a specific variant. Passing
// nil data clears the slot.
func (d *Service) SetLogo(ctx context.Context, actorUserID string, variant thunderdome.LogoVariant, data []byte, contentType string) error {
	dataCol, typeCol, err := logoColumns(variant)
	if err != nil {
		return err
	}

	var dataArg any
	var typeArg any
	if data == nil {
		dataArg = nil
		typeArg = nil
	} else {
		dataArg = data
		typeArg = contentType
	}

	query := fmt.Sprintf(
		`UPDATE thunderdome.branding_settings
			SET %s = $1, %s = $2, updated_by = $3, updated_date = now()
			WHERE id = 1;`, dataCol, typeCol)
	if _, err := d.DB.ExecContext(ctx, query, dataArg, typeArg, actorUserID); err != nil {
		return fmt.Errorf("set logo %s: %v", variant, err)
	}
	return nil
}

// GetLogo returns the bytes + content-type for a variant; returns
// ErrLogoNotFound when the slot is empty.
func (d *Service) GetLogo(ctx context.Context, variant thunderdome.LogoVariant) (thunderdome.BrandLogo, error) {
	dataCol, typeCol, err := logoColumns(variant)
	if err != nil {
		return thunderdome.BrandLogo{}, err
	}
	var data []byte
	var contentType sql.NullString
	query := fmt.Sprintf(`SELECT %s, %s FROM thunderdome.branding_settings WHERE id = 1;`, dataCol, typeCol)
	err = d.DB.QueryRowContext(ctx, query).Scan(&data, &contentType)
	if err != nil {
		return thunderdome.BrandLogo{}, fmt.Errorf("get logo %s: %v", variant, err)
	}
	if len(data) == 0 {
		return thunderdome.BrandLogo{}, ErrLogoNotFound
	}
	return thunderdome.BrandLogo{Data: data, ContentType: contentType.String}, nil
}

// ResetBranding clears all branding fields back to defaults (empty strings,
// no logos).
func (d *Service) ResetBranding(ctx context.Context, actorUserID string) (thunderdome.Branding, error) {
	_, err := d.DB.ExecContext(ctx,
		`UPDATE thunderdome.branding_settings SET
			brand_name = '', primary_color = '', accent_color = '', dark_color = '',
			logo_main_data = NULL, logo_main_content_type = NULL,
			logo_dark_data = NULL, logo_dark_content_type = NULL,
			favicon_data = NULL, favicon_content_type = NULL,
			email_logo_data = NULL, email_logo_content_type = NULL,
			updated_by = $1, updated_date = now()
		WHERE id = 1;`, actorUserID,
	)
	if err != nil {
		return thunderdome.Branding{}, fmt.Errorf("reset branding: %v", err)
	}
	return d.GetBranding(ctx)
}

func logoColumns(variant thunderdome.LogoVariant) (data, contentType string, err error) {
	switch variant {
	case thunderdome.LogoVariantMain:
		return "logo_main_data", "logo_main_content_type", nil
	case thunderdome.LogoVariantDark:
		return "logo_dark_data", "logo_dark_content_type", nil
	case thunderdome.LogoVariantFavicon:
		return "favicon_data", "favicon_content_type", nil
	case thunderdome.LogoVariantEmailLogo:
		return "email_logo_data", "email_logo_content_type", nil
	default:
		return "", "", fmt.Errorf("unknown logo variant: %s", variant)
	}
}
