//go:build unit

package middleware

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsGatewayModelsListPath(t *testing.T) {
	t.Parallel()

	require.True(t, isGatewayModelsListPath(http.MethodGet, "/v1/models"))
	require.True(t, isGatewayModelsListPath(http.MethodGet, "/models"))
	require.True(t, isGatewayModelsListPath(http.MethodGet, "/backend-api/codex/models"))
	require.True(t, isGatewayModelsListPath(http.MethodGet, "/v1beta/models"))
	require.True(t, isGatewayModelsListPath(http.MethodGet, "/v1beta/models/gemini-pro"))
	require.True(t, isGatewayModelsListPath(http.MethodGet, "/antigravity/models"))
	require.True(t, isGatewayModelsListPath(http.MethodGet, "/antigravity/v1/models"))
	require.True(t, isGatewayModelsListPath(http.MethodGet, "/antigravity/v1beta/models"))
	require.True(t, isGatewayModelsListPath(http.MethodGet, "/antigravity/v1beta/models/gemini-pro"))

	require.False(t, isGatewayModelsListPath(http.MethodPost, "/v1/models"))
	require.False(t, isGatewayModelsListPath(http.MethodPost, "/v1beta/models/gemini-pro:generateContent"))
	require.False(t, isGatewayModelsListPath(http.MethodGet, "/v1/images/batches/models"))
	require.False(t, isGatewayModelsListPath(http.MethodGet, "/v1/messages"))
	require.False(t, isGatewayModelsListPath(http.MethodGet, "/t"))
	require.False(t, isGatewayModelsListPath(http.MethodGet, "/v1/models/gpt-5"))
	require.False(t, isGatewayModelsListPath(http.MethodGet, "/admin/models"))
	require.False(t, isGatewayModelsListPath(http.MethodGet, "/dashboard/models"))
}
