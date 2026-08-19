package server

import (
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/checkin"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func ProvideCheckInModule(
	db *sql.DB,
	balanceCache *service.BillingCacheService,
	authCache *service.APIKeyService,
) *checkin.Module {
	return checkin.NewModule(db, balanceCache, authCache)
}
