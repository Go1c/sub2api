package admin

import (
	"strings"
	"time"

	responsepkg "github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type BalanceClientHandler struct {
	svc *service.BalanceWalletService
}

func NewBalanceClientHandler(svc *service.BalanceWalletService) *BalanceClientHandler {
	return &BalanceClientHandler{svc: svc}
}

type balanceClientResponse struct {
	ClientID        string   `json:"client_id"`
	Name            string   `json:"name"`
	Secret          string   `json:"secret,omitempty"`
	SecretPrefix    string   `json:"secret_prefix"`
	AllowedPurposes []string `json:"allowed_purposes"`
	Status          string   `json:"status"`
	LastUsedAt      *string  `json:"last_used_at"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

func (h *BalanceClientHandler) Create(c *gin.Context) {
	var req struct {
		Name            string   `json:"name" binding:"required"`
		AllowedPurposes []string `json:"allowed_purposes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		responsepkg.WalletErrorFrom(c, service.ErrBalanceClientInvalid)
		return
	}
	client, err := h.svc.CreateClient(c.Request.Context(), service.CreateBalanceClientInput{Name: req.Name, AllowedPurposes: req.AllowedPurposes})
	if err != nil {
		responsepkg.WalletErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"client_id": client.ClientID, "client_name": client.Name})
	responsepkg.WalletCreated(c, toBalanceClientResponse(client, true))
}

func (h *BalanceClientHandler) List(c *gin.Context) {
	clients, err := h.svc.ListClients(c.Request.Context())
	if err != nil {
		responsepkg.WalletErrorFrom(c, err)
		return
	}
	items := make([]balanceClientResponse, 0, len(clients))
	for i := range clients {
		items = append(items, toBalanceClientResponse(&clients[i], false))
	}
	responsepkg.WalletSuccess(c, items)
}

func (h *BalanceClientHandler) Get(c *gin.Context) {
	client, err := h.svc.GetClient(c.Request.Context(), c.Param("id"))
	if err != nil {
		responsepkg.WalletErrorFrom(c, err)
		return
	}
	responsepkg.WalletSuccess(c, toBalanceClientResponse(client, false))
}

func (h *BalanceClientHandler) Update(c *gin.Context) {
	var req struct {
		Name            *string   `json:"name"`
		AllowedPurposes *[]string `json:"allowed_purposes"`
		Status          *string   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		responsepkg.WalletErrorFrom(c, service.ErrBalanceClientInvalid)
		return
	}
	client, err := h.svc.UpdateClient(c.Request.Context(), c.Param("id"), service.UpdateBalanceClientInput{
		Name: req.Name, AllowedPurposes: req.AllowedPurposes, Status: req.Status,
	})
	if err != nil {
		responsepkg.WalletErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"client_id": client.ClientID, "client_name": client.Name})
	responsepkg.WalletSuccess(c, toBalanceClientResponse(client, false))
}

func (h *BalanceClientHandler) RotateSecret(c *gin.Context) {
	client, err := h.svc.RotateClientSecret(c.Request.Context(), c.Param("id"))
	if err != nil {
		responsepkg.WalletErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"client_id": client.ClientID, "client_name": client.Name})
	responsepkg.WalletSuccess(c, toBalanceClientResponse(client, true))
}

func (h *BalanceClientHandler) Delete(c *gin.Context) {
	clientID := strings.TrimSpace(c.Param("id"))
	if err := h.svc.DeactivateClient(c.Request.Context(), clientID); err != nil {
		responsepkg.WalletErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"client_id": clientID})
	responsepkg.WalletSuccess(c, gin.H{"client_id": clientID, "status": service.BalanceClientStatusInactive})
}

func toBalanceClientResponse(client *service.BalanceDebitClient, includeSecret bool) balanceClientResponse {
	if client == nil {
		return balanceClientResponse{}
	}
	result := balanceClientResponse{
		ClientID: client.ClientID, Name: client.Name, SecretPrefix: client.SecretPrefix,
		AllowedPurposes: client.AllowedPurposes, Status: client.Status,
		CreatedAt: client.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: client.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if includeSecret {
		result.Secret = client.Secret
	}
	if client.LastUsedAt != nil {
		value := client.LastUsedAt.UTC().Format(time.RFC3339Nano)
		result.LastUsedAt = &value
	}
	return result
}
