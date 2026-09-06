package service

import (
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

// codexUpstreamMinVersion 上游 /backend-api/codex 接受的最低 version 头：
// 若请求携带 version 且低于该值，上游直接 404（issue #3901，2026-07 实测）。
const codexUpstreamMinVersion = "0.144.0"

// codexOriginatorNormalization 控制 enforceCodexIdentityHeaders 是否把落在上游降载桶的
// Codex 身份改写为 CLI 身份，由 gateway.disable_codex_originator_normalization 在服务构造时取反发布。
// 默认开启：降载桶命中会让上游回 server_is_overloaded，网关据此判定瞬时故障并冷却账号。
var codexOriginatorNormalization = func() *atomic.Bool {
	v := &atomic.Bool{}
	v.Store(true)
	return v
}()

// SetCodexOriginatorNormalizationEnabled 发布 Codex 降载身份归一化开关。
// enforceCodexIdentityHeaders 是所有出站路径共用的纯函数收口点，无法在热路径注入配置，
// 故由持有配置的服务在构造时发布进程级快照。
func SetCodexOriginatorNormalizationEnabled(enabled bool) {
	codexOriginatorNormalization.Store(enabled)
}

// codexClientVersionMaxLen 官方版本号均为短 ASCII 串，远低于此上限。
const codexClientVersionMaxLen = 64

// codexClientVersionPattern 允许 0.153.4 与 0.147.0-alpha.4 两类官方形态。
var codexClientVersionPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+){1,3}(-[0-9A-Za-z.]+)?$`)

// NormalizeCodexClientVersion 校验并归一化 Codex 客户端版本号，非法值返回空串。
func NormalizeCodexClientVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || len(version) > codexClientVersionMaxLen || !codexClientVersionPattern.MatchString(version) {
		return ""
	}
	return version
}

// CodexCanonicalUserAgent 返回当前生效的规范 Codex User-Agent。
// fork 没有面板 UA 解析器：规范身份就是编译期 CLI 常量，与推理解析链同源。
func CodexCanonicalUserAgent() string {
	return resolveCodexOutboundIdentity("").userAgent
}

// CodexCanonicalAuthIdentity 返回凭据面（auth.openai.com：换 Token / 刷新 / whoami）
// 出站请求的身份对：规范 User-Agent 与配套 originator。
// 凭据面不发 version 头——真实 Codex 客户端在该面只携带 originator 与 User-Agent
// （codex-rs login/default_client.rs 的 default_headers()），version 门槛
// （issue #3901）只存在于 /backend-api/codex 推理面。
func CodexCanonicalAuthIdentity() (userAgent, originator string) {
	identity := resolveCodexOutboundIdentity("")
	return identity.userAgent, identity.originator
}

// ApplyCodexCanonicalAuthIdentity 为凭据面出站请求写入身份对（不含 version）。
func ApplyCodexCanonicalAuthIdentity(h http.Header) {
	if h == nil {
		return
	}
	userAgent, originator := CodexCanonicalAuthIdentity()
	h.Set("user-agent", userAgent)
	h.Set("originator", originator)
}

// CodexCanonicalClientVersion 返回当前生效的 Codex 客户端版本号。
func CodexCanonicalClientVersion() string {
	return resolveCodexOutboundIdentity("").version
}

// codexOutboundIdentity 出站身份三元组，三者必须同源自洽：
// originator 与 User-Agent 首段配套（否则上游 404，issue #3901），
// version 等于 User-Agent 的版本段且不低于上游门槛。
type codexOutboundIdentity struct {
	userAgent  string
	originator string
	version    string
}

// resolveCodexOutboundIdentity 由候选 User-Agent 推导自洽的出站身份。
// candidateUA 为空时使用编译期 CLI 规范身份。推导不出官方身份时整体回退。
//
// 与 origin 不同：fork 的收口仍是「配对 + 降载桶归一化」，不把所有出站强制改写成
// 规范 UA。候选 UA 的版本段原样保留（只要合法且不低于门槛）。
func resolveCodexOutboundIdentity(candidateUA string) codexOutboundIdentity {
	canonical := codexOutboundIdentity{
		userAgent:  codexCLIUserAgent,
		originator: openai.CodexCLIOriginator,
		version:    codexCLIVersion,
	}
	ua := strings.TrimSpace(candidateUA)
	if ua == "" {
		return canonical
	}
	originator, pairedUA, ok := openai.PairCodexClientIdentity(ua)
	if !ok {
		return canonical
	}
	if codexOriginatorNormalization.Load() {
		originator, pairedUA, _ = openai.NormalizeCodexClientIdentityToCLI(originator, pairedUA)
	}
	version := canonical.version
	if parsed, ok := openai.ParseCodexEngineVersion(pairedUA); ok {
		if normalized := NormalizeCodexClientVersion(parsed); normalized != "" && CompareVersions(normalized, codexUpstreamMinVersion) >= 0 {
			version = normalized
		}
	}
	return codexOutboundIdentity{userAgent: pairedUA, originator: originator, version: version}
}

// ensureCodexIdentityHeaders 补齐 OAuth（ChatGPT 内部接口）出站请求所需的 Codex 身份头。
// 已有 User-Agent 与 version 保持不变，交给紧随其后的 enforceCodexIdentityHeaders
// 做官方身份配对与最低版本校正。
func ensureCodexIdentityHeaders(h http.Header) {
	if h == nil {
		return
	}
	identity := resolveCodexOutboundIdentity("")
	if strings.TrimSpace(h.Get("user-agent")) == "" {
		h.Set("user-agent", identity.userAgent)
	}
	if strings.TrimSpace(h.Get("originator")) == "" {
		h.Set("originator", identity.originator)
	}
	if strings.TrimSpace(h.Get("version")) == "" {
		h.Set("version", identity.version)
	}
	h.Set("OpenAI-Beta", "responses=experimental")
}

// enforceCodexIdentityHeaders 收口 OAuth（ChatGPT 内部接口）出站请求的客户端身份头。
// 上游要求 originator 与 User-Agent 首段配套且为官方客户端标识，version 头（若携带）
// 不低于 0.144.0，任一不满足即 404（issue #3901）。以最终 User-Agent 为准推导配套
// originator；推导不出官方身份（第三方 UA / UA 缺失）时整体回退为默认 Codex CLI 身份。
//
// 配对之后再做降载身份归一化：上游按 originator 分桶调度容量，命中降载桶的请求会被回
// server_is_overloaded，网关据此判定瞬时上游故障并冷却账号（对外表现为账号过载不可用），
// 故这类身份统一改写为 CLI 身份——只替换身份段，保留版本 / OS / 架构 / 终端指纹。
//
// 仅对携带 originator 的请求生效；需要从缺失身份头恢复的调用方应先调用
// ensureCodexIdentityHeaders。
// 必须在所有 User-Agent 改写（自定义 UA / ForceCodexCLI / 浏览器 UA 兜底）之后调用。
func enforceCodexIdentityHeaders(h http.Header) {
	if h == nil || h.Get("originator") == "" {
		return
	}
	originator, pairedUA, ok := openai.PairCodexClientIdentity(h.Get("user-agent"))
	if !ok {
		canonical := resolveCodexOutboundIdentity("")
		originator, pairedUA = canonical.originator, canonical.userAgent
	}
	if codexOriginatorNormalization.Load() {
		originator, pairedUA, _ = openai.NormalizeCodexClientIdentityToCLI(originator, pairedUA)
	}
	h.Set("user-agent", pairedUA)
	h.Set("originator", originator)
	if v := strings.TrimSpace(h.Get("version")); v != "" && CompareVersions(v, codexUpstreamMinVersion) < 0 {
		h.Set("version", CodexCanonicalClientVersion())
	}
}
