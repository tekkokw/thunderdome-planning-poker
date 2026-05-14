package thunderdome

import "time"

// Branding holds workspace-wide branding fields editable by an admin at
// runtime. Logo bytes live in their own getter to avoid sending raw bytea on
// every settings page load.
type Branding struct {
	BrandName     string    `json:"brand_name"`
	PrimaryColor  string    `json:"primary_color"`
	AccentColor   string    `json:"accent_color"`
	DarkColor     string    `json:"dark_color"`
	HasLogoMain   bool      `json:"has_logo_main"`
	HasLogoDark   bool      `json:"has_logo_dark"`
	HasFavicon    bool      `json:"has_favicon"`
	HasEmailLogo  bool      `json:"has_email_logo"`
	UpdatedBy     *string   `json:"updated_by,omitempty"`
	UpdatedDate   time.Time `json:"updated_date"`
}

// BrandLogo is a downloadable logo asset.
type BrandLogo struct {
	Data        []byte
	ContentType string
}

// LogoVariant identifies which logo slot to read/write.
type LogoVariant string

const (
	LogoVariantMain      LogoVariant = "main"
	LogoVariantDark      LogoVariant = "dark"
	LogoVariantFavicon   LogoVariant = "favicon"
	LogoVariantEmailLogo LogoVariant = "email"
)
