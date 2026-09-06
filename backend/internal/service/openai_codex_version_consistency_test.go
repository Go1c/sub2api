//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodexVersionConstants_Consistency(t *testing.T) {
	const wantVersion = "0.153.4"
	const wantUserAgent = "codex_cli_rs/0.153.4 (Mac OS 26.5.2; arm64) iTerm.app/3.6.8"
	const cliTrailer = "(codex_cli_rs; 0.153.4)"

	require.Equal(t, wantVersion, codexCLIVersion)
	require.Equal(t, wantUserAgent, codexCLIUserAgent)
	require.False(t, strings.HasSuffix(codexCLIUserAgent, cliTrailer),
		"CLI User-Agent must not include the TUI/exec trailer")
	require.Contains(t, codexCLIUserAgent, "iTerm.app/3.6.8")
	require.NotContains(t, codexCLIUserAgent, "iTerm2",
		"terminal segment must be TERM_PROGRAM iTerm.app, not table name iTerm2")
	require.True(t, strings.Contains(codexCLIUserAgent, "codex_cli_rs/"+codexCLIVersion),
		"codexCLIUserAgent must embed codexCLIVersion")
	require.Equal(t, codexCLIVersion, CodexCanonicalClientVersion())
	require.Equal(t, codexCLIUserAgent, CodexCanonicalUserAgent())
}
