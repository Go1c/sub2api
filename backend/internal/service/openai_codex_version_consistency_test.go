//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

func TestCodexVersionConstants_Consistency(t *testing.T) {
	const wantVersion = "0.154.0"
	const wantUserAgent = "codex-tui/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color (codex-tui; 0.154.0)"

	require.Equal(t, wantVersion, codexCLIVersion)
	require.Equal(t, wantUserAgent, codexCLIUserAgent)
	require.True(t, strings.HasSuffix(codexCLIUserAgent, "(codex-tui; "+wantVersion+")"),
		"Kin 规范身份是 TUI，必须带 trailer")
	require.Contains(t, codexCLIUserAgent, "xterm-256color")
	require.NotContains(t, codexCLIUserAgent, "WindowsTerminal",
		"Linux 实测 TERM 是 xterm-256color，不是设置页占位符")
	require.True(t, strings.Contains(codexCLIUserAgent, openai.CodexDefaultOriginator+"/"+codexCLIVersion),
		"codexCLIUserAgent must embed codexCLIVersion")
	require.True(t, strings.Contains(DefaultOpenAICodexUserAgent, codexCLIVersion),
		"DefaultOpenAICodexUserAgent must embed codexCLIVersion")
}
