//go:build unit

package handler

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
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
	require.False(t, gjson.GetBytes(body, "include").Exists(), "xAI rejects include=x_search_call.action.sources with HTTP 400")
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

func TestResolveGrokStandaloneSearchModelUsesRuntimeDefault(t *testing.T) {
	original := xai.RuntimeModelMappingOptions()
	t.Cleanup(func() { xai.SetRuntimeModelMappingOptions(original) })
	xai.SetRuntimeModelMappingOptions(xai.ModelMappingOptions{DefaultText: "grok-4.6"})

	model := resolveGrokStandaloneSearchModel()
	body, err := buildGrokXSearchResponsesBody(grokStandaloneSearchRequest{Query: "latest posts from xAI"}, model)
	require.NoError(t, err)
	require.Equal(t, "grok-4.6", model)
	require.Equal(t, model, gjson.GetBytes(body, "model").String())
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

func TestExtractGrokXSearchSourcesFromLiveCustomToolCallAndCitations(t *testing.T) {
	t.Parallel()
	body, err := json.Marshal(map[string]any{
		"output": []map[string]any{
			{
				"type": "custom_tool_call",
				"name": "x_keyword_search",
				"input": `{"query":"from:xai","limit":"5","mode":"Latest"}`,
			},
			{
				"type": "message",
				"content": []map[string]any{
					{
						"type": "output_text",
						"text": `The latest official xAI post is at https://x.com/SpaceXAI/status/2074214064746832060.`,
						"annotations": []map[string]any{{
							"type":  "url_citation",
							"url":   "https://x.com/SpaceXAI/status/2074214064746832060",
							"title": "1",
						}},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	results := extractGrokWebSearchSources(body, 5)
	require.Len(t, results, 1)
	require.Equal(t, "https://x.com/SpaceXAI/status/2074214064746832060", results[0].URL)
}

func TestExtractGrokXSearchSourcesFromStructuredJSONWithoutNativeCall(t *testing.T) {
	t.Parallel()
	body, err := json.Marshal(map[string]any{
		"output": []map[string]any{{
			"type": "message",
			"content": []map[string]any{{
				"type": "output_text",
				"text": `{"results":[{"url":"https://x.com/xai/status/9","title":"Official post","snippet":"rebrand"}]}`,
			}},
		}},
	})
	require.NoError(t, err)

	results := extractGrokWebSearchSources(body, 5)
	require.Len(t, results, 1)
	require.Equal(t, "https://x.com/xai/status/9", results[0].URL)
	require.Equal(t, "Official post", results[0].Title)
	require.Equal(t, "rebrand", results[0].Snippet)
}

func TestNormalizeGrokWebSearchMaxResults(t *testing.T) {
	t.Parallel()
	require.Equal(t, 5, normalizeGrokWebSearchMaxResults(0))
	require.Equal(t, 20, normalizeGrokWebSearchMaxResults(99))
	require.Equal(t, 7, normalizeGrokWebSearchMaxResults(7))
}
