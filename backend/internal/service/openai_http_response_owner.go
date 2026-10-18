package service

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

const openAIHTTPResponseOwnerContextKey = "openai_http_response_owner"

type openAIHTTPResponseOwner struct {
	userID   int64
	apiKeyID int64
}

// SetOpenAIHTTPResponseOwner marks the authenticated downstream owner whose
// successful Responses IDs may be used for later HTTP continuations.
func SetOpenAIHTTPResponseOwner(c *gin.Context, userID, apiKeyID int64) {
	if c == nil || userID <= 0 || apiKeyID <= 0 {
		return
	}
	c.Set(openAIHTTPResponseOwnerContextKey, openAIHTTPResponseOwner{userID: userID, apiKeyID: apiKeyID})
}

// ValidateOpenAIHTTPResponseOwner authorizes a continuation by downstream
// tenant. API key identity is retained in the binding, while keys owned by the
// same user remain interoperable.
func (s *OpenAIGatewayService) ValidateOpenAIHTTPResponseOwner(
	ctx context.Context,
	groupID int64,
	responseID string,
	userID, apiKeyID int64,
) (bool, error) {
	if s == nil || strings.TrimSpace(responseID) == "" || userID <= 0 || apiKeyID <= 0 {
		return false, nil
	}
	store := s.getOpenAIWSStateStore()
	if store == nil {
		return false, nil
	}
	ownerUserID, ownerAPIKeyID, found, err := store.GetHTTPResponseOwner(ctx, groupID, responseID)
	if err != nil || !found {
		return false, err
	}
	return ownerUserID == userID || (ownerUserID <= 0 && ownerAPIKeyID == apiKeyID), nil
}
