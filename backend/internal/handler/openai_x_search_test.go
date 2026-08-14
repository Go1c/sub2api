//go:build unit

package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBuildGrokXSearchResponsesBody(t *testing.T) {
	t.Parallel()
	understandImages := true
	understandVideos := false
	body, err := buildGrokXSearchResponsesBody(grokStandaloneSearchRequest{
		Query:                    "latest posts from xAI",
		AllowedXHandles:          []string{"xai"},
		ExcludedXHandles:         []string{"spam"},
		FromDate:                 "2026-08-01",
		ToDate:                   "2026-08-10",
		EnableImageUnderstanding: &understandImages,
		EnableVideoUnderstanding: &understandVideos,
	}, grokStandaloneSearchDefaultModel)
	require.NoError(t, err)
	require.Equal(t, grokStandaloneSearchDefaultModel, gjson.GetBytes(body, "model").String())
	require.Contains(t, gjson.GetBytes(body, "input").String(), "latest posts from xAI")
	require.Contains(t, gjson.GetBytes(body, "input").String(), "Return ONLY valid JSON")
	require.Equal(t, "x_search_call.action.sources", gjson.GetBytes(body, "include.0").String())
	require.Equal(t, "required", gjson.GetBytes(body, "tool_choice").String())
	require.Equal(t, "x_search", gjson.GetBytes(body, "tools.0.type").String())
	require.Equal(t, "xai", gjson.GetBytes(body, "tools.0.allowed_x_handles.0").String())
	require.Equal(t, "spam", gjson.GetBytes(body, "tools.0.excluded_x_handles.0").String())
	require.Equal(t, "2026-08-01", gjson.GetBytes(body, "tools.0.from_date").String())
	require.Equal(t, "2026-08-10", gjson.GetBytes(body, "tools.0.to_date").String())
	require.True(t, gjson.GetBytes(body, "tools.0.enable_image_understanding").Bool())
	require.False(t, gjson.GetBytes(body, "tools.0.enable_video_understanding").Bool())
	require.False(t, gjson.GetBytes(body, "store").Bool())
	require.False(t, gjson.GetBytes(body, "stream").Bool())
}

func TestBuildGrokXSearchResponsesBodyAcceptsInputAlias(t *testing.T) {
	t.Parallel()
	body, err := buildGrokXSearchResponsesBody(grokStandaloneSearchRequest{Input: "latest posts from xAI"}, grokStandaloneSearchDefaultModel)
	require.NoError(t, err)
	require.Contains(t, gjson.GetBytes(body, "input").String(), "latest posts from xAI")
}

func TestResolveGrokStandaloneSearchModelUsesForkDefault(t *testing.T) {
	t.Parallel()
	require.Equal(t, "grok-4.5", resolveGrokStandaloneSearchModel())
	body, err := buildGrokXSearchResponsesBody(grokStandaloneSearchRequest{Query: "latest posts from xAI"}, resolveGrokStandaloneSearchModel())
	require.NoError(t, err)
	require.Equal(t, "grok-4.5", gjson.GetBytes(body, "model").String())
}

func TestExtractGrokXSearchSourcesPrefersStructuredResultsOnAllowlist(t *testing.T) {
	t.Parallel()
	body, err := json.Marshal(map[string]any{
		"output": []map[string]any{
			{
				"type": "x_search_call",
				"action": map[string]any{
					"sources": []map[string]any{
						{"url": "https://x.com/xai/status/1", "title": "1", "snippet": "raw"},
						{"url": "https://x.com/xai/status/2", "title": "status 2", "snippet": "raw 2"},
					},
				},
			},
			{
				"type": "message",
				"content": []map[string]any{
					{
						"type": "output_text",
						"text": `{"results":[{"url":"https://x.com/xai/status/1","title":"Official xAI post","snippet":"latest launch"},{"url":"https://example.com/not-allowed","title":"dropped","snippet":"no"}]}`,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	results := extractGrokWebSearchSources(body, 5)
	require.Len(t, results, 2)
	require.Equal(t, "https://x.com/xai/status/1", results[0].URL)
	require.Equal(t, "Official xAI post", results[0].Title)
	require.Equal(t, "latest launch", results[0].Snippet)
	require.Equal(t, "https://x.com/xai/status/2", results[1].URL)
	require.Equal(t, "status 2", results[1].Title)
}

func TestNormalizeGrokWebSearchMaxResults(t *testing.T) {
	t.Parallel()
	require.Equal(t, 5, normalizeGrokWebSearchMaxResults(0))
	require.Equal(t, 20, normalizeGrokWebSearchMaxResults(99))
	require.Equal(t, 7, normalizeGrokWebSearchMaxResults(7))
}
