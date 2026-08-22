//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/stretchr/testify/require"
)

func grokMappingTestSettingService() *SettingService {
	return &SettingService{cfg: &config.Config{}}
}

func TestParseSettingsPublishesGrokRuntimeModelMapping(t *testing.T) {
	original := xai.RuntimeModelMappingOptions()
	t.Cleanup(func() { xai.SetRuntimeModelMappingOptions(original) })

	svc := grokMappingTestSettingService()
	settings := svc.parseSettings(map[string]string{})

	require.Equal(t, "grok-4.6", settings.GrokDefaultTextModel)
	require.True(t, settings.GrokCrossClientModelMapEnabled)

	opts := xai.RuntimeModelMappingOptions()
	require.Equal(t, "grok-4.6", opts.DefaultText)
	require.True(t, opts.EnableCrossClientMap)

	mapping := xai.DefaultModelMapping()
	require.Equal(t, "grok-4.6", mapping["grok"])
	require.Equal(t, "grok-4.6", mapping["gpt-*"])
	require.Equal(t, "grok-4.6", mapping["claude-*"])
}

func TestParseSettingsDisablesGrokCrossClientMap(t *testing.T) {
	original := xai.RuntimeModelMappingOptions()
	t.Cleanup(func() { xai.SetRuntimeModelMappingOptions(original) })

	svc := grokMappingTestSettingService()
	settings := svc.parseSettings(map[string]string{
		SettingKeyGrokDefaultTextModel:           "grok-4.3",
		SettingKeyGrokCrossClientModelMapEnabled: "false",
	})

	require.Equal(t, "grok-4.3", settings.GrokDefaultTextModel)
	require.False(t, settings.GrokCrossClientModelMapEnabled)

	mapping := xai.DefaultModelMapping()
	require.Equal(t, "grok-4.3", mapping["grok"])
	_, hasGPT := mapping["gpt-*"]
	require.False(t, hasGPT)
}
