package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	adminDB "github.com/StevenWeathers/thunderdome-planning-poker/internal/db/admin"
	"github.com/StevenWeathers/thunderdome-planning-poker/thunderdome"
)

// maxLogoSize caps each uploaded image at 512KB. Logos for an internal app are
// typically <50KB; the limit blocks accidental full-resolution dumps without
// being so tight it rejects a small PNG.
const maxLogoSize = 512 * 1024

// handleGetBranding returns the current branding metadata. Public — every
// request hits this on app boot to apply colors and decide whether to render
// the uploaded logo.
//
//	@Summary		Get Branding
//	@Tags			branding
//	@Produce		json
//	@Success		200		object	standardJsonResponse{data=thunderdome.Branding}
//	@Router			/branding [get]
func (s *Service) handleGetBranding() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		b, err := s.AdminDataSvc.GetBranding(ctx)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleGetBranding error", zap.Error(err))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}
		s.Success(w, r, http.StatusOK, b, nil)
	}
}

// handleGetBrandingLogo streams a logo binary by variant.
//
//	@Summary		Get Branding Logo
//	@Tags			branding
//	@Produce		image/png
//	@Param			variant		query	string	true	"logo variant: main, dark, favicon, email"
//	@Success		200			string	binary
//	@Failure		404			object	standardJsonResponse{}
//	@Router			/branding/logo [get]
func (s *Service) handleGetBrandingLogo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		variant := thunderdome.LogoVariant(strings.TrimSpace(r.URL.Query().Get("variant")))
		if variant == "" {
			variant = thunderdome.LogoVariantMain
		}

		logo, err := s.AdminDataSvc.GetLogo(ctx, variant)
		if errors.Is(err, adminDB.ErrLogoNotFound) {
			s.Failure(w, r, http.StatusNotFound, Errorf(ENOTFOUND, "LOGO_NOT_SET"))
			return
		}
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleGetBrandingLogo error", zap.Error(err), zap.String("variant", string(variant)))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		ct := logo.ContentType
		if ct == "" {
			ct = http.DetectContentType(logo.Data)
		}
		w.Header().Set("Content-Type", ct)
		// Short-lived cache: branding can change at any moment. We trade hot
		// re-fetches for instant visibility of admin changes.
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(logo.Data)
	}
}

type brandingUpdateRequestBody struct {
	BrandName    *string `json:"brand_name"`
	PrimaryColor *string `json:"primary_color"`
	AccentColor  *string `json:"accent_color"`
	DarkColor    *string `json:"dark_color"`
}

// handleUpdateBranding updates the text/color portion of branding settings.
//
//	@Summary		Update Branding
//	@Tags			branding
//	@Produce		json
//	@Param			branding	body	brandingUpdateRequestBody	true	"branding metadata"
//	@Success		200			object	standardJsonResponse{data=thunderdome.Branding}
//	@Security		ApiKeyAuth
//	@Router			/admin/branding [put]
func (s *Service) handleUpdateBranding() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)

		var req brandingUpdateRequestBody
		body, bodyErr := io.ReadAll(r.Body)
		if bodyErr != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, bodyErr.Error()))
			return
		}
		if err := json.Unmarshal(body, &req); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		for _, color := range []*string{req.PrimaryColor, req.AccentColor, req.DarkColor} {
			if color != nil && !isValidHexColor(*color) && *color != "" {
				s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, "INVALID_HEX_COLOR"))
				return
			}
		}

		updated, err := s.AdminDataSvc.UpdateBrandingMeta(ctx, sessionUserID, req.BrandName, req.PrimaryColor, req.AccentColor, req.DarkColor)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleUpdateBranding error", zap.Error(err), zap.String("session_user_id", sessionUserID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}
		s.Success(w, r, http.StatusOK, updated, nil)
	}
}

// handleUploadBrandingLogo accepts a single logo file (multipart form, field
// "file") for the given variant and stores it.
//
//	@Summary		Upload Branding Logo
//	@Tags			branding
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			variant		query	string	true	"logo variant: main, dark, favicon, email"
//	@Param			file		formData file	true	"image file (SVG or PNG, <=512KB)"
//	@Success		200			object	standardJsonResponse{data=thunderdome.Branding}
//	@Security		ApiKeyAuth
//	@Router			/admin/branding/logo [post]
func (s *Service) handleUploadBrandingLogo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)

		variant := thunderdome.LogoVariant(strings.TrimSpace(r.URL.Query().Get("variant")))
		switch variant {
		case thunderdome.LogoVariantMain, thunderdome.LogoVariantDark,
			thunderdome.LogoVariantFavicon, thunderdome.LogoVariantEmailLogo:
		default:
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, "INVALID_LOGO_VARIANT"))
			return
		}

		if err := r.ParseMultipartForm(maxLogoSize + 1024); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, "FILE_REQUIRED"))
			return
		}
		defer file.Close()

		if header.Size > maxLogoSize {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, fmt.Sprintf("FILE_TOO_LARGE_MAX_%dKB", maxLogoSize/1024)))
			return
		}

		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, io.LimitReader(file, maxLogoSize+1)); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}
		if buf.Len() > maxLogoSize {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, fmt.Sprintf("FILE_TOO_LARGE_MAX_%dKB", maxLogoSize/1024)))
			return
		}

		data := buf.Bytes()
		contentType := detectImageContentType(data, header.Filename, header.Header.Get("Content-Type"))
		if contentType == "" {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, "UNSUPPORTED_IMAGE_TYPE"))
			return
		}

		if err := s.AdminDataSvc.SetLogo(ctx, sessionUserID, variant, data, contentType); err != nil {
			s.Logger.Ctx(ctx).Error("handleUploadBrandingLogo error", zap.Error(err),
				zap.String("session_user_id", sessionUserID), zap.String("variant", string(variant)))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		updated, _ := s.AdminDataSvc.GetBranding(ctx)
		s.Success(w, r, http.StatusOK, updated, nil)
	}
}

// handleDeleteBrandingLogo clears a single logo variant.
//
//	@Summary		Delete Branding Logo
//	@Tags			branding
//	@Produce		json
//	@Param			variant		query	string	true	"logo variant"
//	@Success		200			object	standardJsonResponse{data=thunderdome.Branding}
//	@Security		ApiKeyAuth
//	@Router			/admin/branding/logo [delete]
func (s *Service) handleDeleteBrandingLogo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)

		variant := thunderdome.LogoVariant(strings.TrimSpace(r.URL.Query().Get("variant")))
		switch variant {
		case thunderdome.LogoVariantMain, thunderdome.LogoVariantDark,
			thunderdome.LogoVariantFavicon, thunderdome.LogoVariantEmailLogo:
		default:
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, "INVALID_LOGO_VARIANT"))
			return
		}

		if err := s.AdminDataSvc.SetLogo(ctx, sessionUserID, variant, nil, ""); err != nil {
			s.Logger.Ctx(ctx).Error("handleDeleteBrandingLogo error", zap.Error(err))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}
		updated, _ := s.AdminDataSvc.GetBranding(ctx)
		s.Success(w, r, http.StatusOK, updated, nil)
	}
}

// handleResetBranding wipes all branding fields back to defaults.
//
//	@Summary		Reset Branding
//	@Tags			branding
//	@Produce		json
//	@Success		200		object	standardJsonResponse{data=thunderdome.Branding}
//	@Security		ApiKeyAuth
//	@Router			/admin/branding [delete]
func (s *Service) handleResetBranding() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)

		updated, err := s.AdminDataSvc.ResetBranding(ctx, sessionUserID)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleResetBranding error", zap.Error(err))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}
		s.Success(w, r, http.StatusOK, updated, nil)
	}
}

func isValidHexColor(s string) bool {
	if !strings.HasPrefix(s, "#") {
		return false
	}
	hex := s[1:]
	if len(hex) != 3 && len(hex) != 6 && len(hex) != 8 {
		return false
	}
	for _, c := range hex {
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !isHex {
			return false
		}
	}
	return true
}

// detectImageContentType returns a canonical content-type for the upload, or
// "" if it doesn't look like an SVG/PNG/JPEG/GIF/WebP/ICO. Filename extension
// is the priority for SVG (http.DetectContentType returns "text/xml" for SVG).
func detectImageContentType(data []byte, filename, hint string) string {
	lowerName := strings.ToLower(filename)
	if strings.HasSuffix(lowerName, ".svg") || strings.Contains(strings.ToLower(hint), "svg") {
		return "image/svg+xml"
	}
	if strings.HasSuffix(lowerName, ".ico") || strings.Contains(strings.ToLower(hint), "icon") {
		return "image/x-icon"
	}
	sniffed := http.DetectContentType(data)
	switch sniffed {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return sniffed
	}
	return ""
}
