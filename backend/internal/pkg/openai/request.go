package openai

import "strings"

// CodexCLIUserAgentPrefixes matches Codex CLI User-Agent patterns
// Examples: "codex_vscode/1.0.0", "codex_cli_rs/0.1.2"
var CodexCLIUserAgentPrefixes = []string{
	"codex_vscode/",
	"codex_cli_rs/",
}

// CodexOfficialClientUserAgentPrefixes matches Codex 官方客户端家族 User-Agent 前缀。
// 该列表仅用于 OpenAI OAuth `codex_cli_only` 访问限制判定。
var CodexOfficialClientUserAgentPrefixes = []string{
	"codex_cli_rs/",
	"codex_vscode/",
	"codex_app/",
	"codex_chatgpt_desktop/",
	"codex_atlas/",
	"codex_exec/",
	"codex_sdk_ts/",
	"codex ",
}

// CodexOfficialClientOriginatorPrefixes matches Codex 官方客户端家族 originator 前缀。
// 说明：OpenAI 官方 Codex 客户端并不只使用固定的 codex_app 标识。
// 例如 codex_cli_rs、codex_vscode、codex_chatgpt_desktop、codex_atlas、codex_exec、codex_sdk_ts 等。
var CodexOfficialClientOriginatorPrefixes = []string{
	"codex_",
	"codex ",
}

// IsCodexCLIRequest checks if the User-Agent indicates a Codex CLI request
func IsCodexCLIRequest(userAgent string) bool {
	ua := normalizeCodexClientHeader(userAgent)
	if ua == "" {
		return false
	}
	return matchCodexClientHeaderPrefixes(ua, CodexCLIUserAgentPrefixes)
}

// IsCodexOfficialClientRequest checks if the User-Agent indicates a Codex 官方客户端请求。
// 与 IsCodexCLIRequest 解耦，避免影响历史兼容逻辑。
func IsCodexOfficialClientRequest(userAgent string) bool {
	ua := normalizeCodexClientHeader(userAgent)
	if ua == "" {
		return false
	}
	return matchCodexClientHeaderPrefixes(ua, CodexOfficialClientUserAgentPrefixes)
}

// IsCodexOfficialClientOriginator checks if originator indicates a Codex 官方客户端请求。
func IsCodexOfficialClientOriginator(originator string) bool {
	v := normalizeCodexClientHeader(originator)
	if v == "" {
		return false
	}
	return matchCodexClientHeaderPrefixes(v, CodexOfficialClientOriginatorPrefixes)
}

// IsCodexOfficialClientByHeaders checks whether the request headers indicate an
// official Codex client family request.
func IsCodexOfficialClientByHeaders(userAgent, originator string) bool {
	return IsCodexOfficialClientRequest(userAgent) || IsCodexOfficialClientOriginator(originator)
}

func normalizeCodexClientHeader(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func matchCodexClientHeaderPrefixes(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		normalizedPrefix := normalizeCodexClientHeader(prefix)
		if normalizedPrefix == "" {
			continue
		}
		// 优先前缀匹配；若 UA/Originator 被网关拼接为复合字符串时，退化为包含匹配。
		if strings.HasPrefix(value, normalizedPrefix) || strings.Contains(value, normalizedPrefix) {
			return true
		}
	}
	return false
}

var codexPairedOriginators = map[string]struct{}{
	"codex_cli_rs":          {},
	"codex_vscode":          {},
	"codex_app":             {},
	"codex_chatgpt_desktop": {},
	"codex_atlas":           {},
	"codex_exec":            {},
	"codex_sdk_ts":          {},
	"codex-tui":             {},
}

// PairCodexClientIdentity derives an originator paired with the final outbound
// User-Agent. The Codex upstream rejects mismatched identity pairs with 404.
func PairCodexClientIdentity(userAgent string) (originator string, pairedUA string, ok bool) {
	ua := strings.TrimSpace(userAgent)
	slash := strings.IndexByte(ua, '/')
	if slash <= 0 {
		return "", "", false
	}
	if leading := strings.TrimSpace(ua[:slash]); saneCodexOriginator(leading) && pairedCodexOriginator(leading) {
		leading = canonicalPairedCodexOriginator(leading)
		return leading, leading + ua[slash:], true
	}
	if trailer := codexUATrailerName(ua); trailer != "" && !strings.ContainsRune(trailer, '/') && saneCodexOriginator(trailer) && pairedCodexOriginator(trailer) {
		trailer = canonicalPairedCodexOriginator(trailer)
		return trailer, trailer + ua[slash:], true
	}
	return "", "", false
}

func codexUATrailerName(ua string) string {
	lastOpen := strings.LastIndex(ua, "(")
	lastClose := strings.LastIndex(ua, ")")
	if lastOpen < 0 || lastClose <= lastOpen {
		return ""
	}
	inside := ua[lastOpen+1 : lastClose]
	name, _, ok := strings.Cut(inside, ";")
	if !ok {
		return ""
	}
	return strings.TrimSpace(name)
}

func saneCodexOriginator(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for i := 0; i < len(name); i++ {
		if c := name[i]; c < 0x20 || c > 0x7e {
			return false
		}
	}
	return true
}

func pairedCodexOriginator(name string) bool {
	if _, ok := codexPairedOriginators[strings.ToLower(strings.TrimSpace(name))]; ok {
		return true
	}
	return strings.HasPrefix(name, "Codex ")
}

func canonicalPairedCodexOriginator(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	if _, ok := codexPairedOriginators[lower]; ok {
		return lower
	}
	return strings.TrimSpace(name)
}
