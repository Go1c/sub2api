//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodexVersionConstants_Consistency(t *testing.T) {
	require.True(t, strings.Contains(codexCLIUserAgent, "codex_cli_rs/"+codexCLIVersion),
		"codexCLIUserAgent must embed codexCLIVersion")
	require.Equal(t, codexCLIVersion, CodexCanonicalClientVersion())
	require.Equal(t, codexCLIUserAgent, CodexCanonicalUserAgent())
}
