package server

import (
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/telemetry"
)

func ProvideTelemetryModule(db *sql.DB, authService *service.AuthService) *telemetry.Module {
	var identity telemetry.AccessTokenIdentifier
	if authService != nil {
		identity = authServiceAccessToken{auth: authService}
	}
	mod := telemetry.NewModule(db, identity)
	if authService != nil {
		authService.SetFirstPartyTelemetry(mod)
	}
	return mod
}

type authServiceAccessToken struct {
	auth *service.AuthService
}

func (a authServiceAccessToken) IdentifyAccessToken(token string) (int64, error) {
	if a.auth == nil {
		return 0, errors.New("auth service unavailable")
	}
	claims, err := a.auth.ValidateToken(token)
	if err != nil {
		return 0, err
	}
	if claims == nil || claims.UserID <= 0 {
		return 0, errors.New("invalid token")
	}
	return claims.UserID, nil
}
