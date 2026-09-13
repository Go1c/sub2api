//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodexVersionConstants_Consistency(t *testing.T) {
	const wantVersion = "0.154.0"
	const wantUserAgent = "codex_cli_rs/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color"
	const cliTrailer = "(codex_cli_rs; 0.154.0)"

	require.Equal(t, wantVersion, codexCLIVersion)
	require.Equal(t, wantUserAgent, codexCLIUserAgent)
	require.False(t, strings.HasSuffix(codexCLIUserAgent, cliTrailer),
		"CLI User-Agent must not include the TUI/exec trailer")
	require.Contains(t, codexCLIUserAgent, "xterm-256color")
	require.NotContains(t, codexCLIUserAgent, "WindowsTerminal",
		"Linux host terminal segment is TERM=xterm-256color, not the settings placeholder")
	require.True(t, strings.Contains(codexCLIUserAgent, "codex_cli_rs/"+codexCLIVersion),
		"codexCLIUserAgent must embed codexCLIVersion")
	require.Equal(t, codexCLIVersion, CodexCanonicalClientVersion())
	require.Equal(t, codexCLIUserAgent, CodexCanonicalUserAgent())
}
