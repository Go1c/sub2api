package service

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service/basispoints"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	basisPointsExtraKey        = "basispoints"
	basisPointsUpstreamURL     = "https://bps.openai.com/basispoints/api/responses"
	basisPointsUpstreamHost    = "bps.openai.com"
	basisPointsModelAstra      = "gpt-6-astra"
	basisPointsModelSol        = "gpt-6-sol"
	basisPointsAttachURL       = "https://bps.openai.com/basispoints/api/attachments"
	basisPointsTransport       = "run_officejs"
	basisPointsTransportAlt    = "functions.run_officejs"
	basisPointsAuthMode        = "chatgpt"
	basisPointsUpstreamPath    = "/basispoints/api/responses"
	basisPointsBypassHeader    = "X-Codex2API-Basispoints-Bypass"
	basisPointsEnvelopeMaxSize = 1 << 20
)

// BasisPointsEnabled reports whether this OpenAI OAuth account sends
// gpt-6-astra and gpt-6-sol through the Excel Basis Points upstream.
// Missing or non-object extra stays off.
func (a *Account) BasisPointsEnabled() bool {
	if a == nil || !a.IsOpenAIOAuthLike() || a.Extra == nil {
		return false
	}
	raw, ok := a.Extra[basisPointsExtraKey]
	if !ok || raw == nil {
		return false
	}
	record, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	return extraBool(record["enabled"])
}

func extraBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

// basisPointsRouteEligible names the hardcoded public models an enabled
// account redirects onto Basis Points. The upstream model stays the requested
// name. Other models stay on Codex.
func basisPointsRouteEligible(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case basisPointsModelAstra, basisPointsModelSol:
		return true
	default:
		return false
	}
}

func basisPointsUpstreamModel(model string) string {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case basisPointsModelSol:
		return basisPointsModelSol
	default:
		return basisPointsModelAstra
	}
}

// basisPointsNativeFallbackReason names a capability Basis Points cannot run.
// Those requests stay on Codex so the client does not fail at the BPS boundary.
// An empty result means the request can use the BPS bridge.
func basisPointsNativeFallbackReason(body []byte) string {
	if !gjson.ValidBytes(body) {
		return ""
	}
	if tools := gjson.GetBytes(body, "tools"); tools.IsArray() {
		fallback := ""
		tools.ForEach(func(_, tool gjson.Result) bool {
			kind := strings.ToLower(strings.TrimSpace(tool.Get("type").String()))
			switch kind {
			case "image_generation":
				fallback = "image_generation"
				return false
			case "web_search", "web_search_preview", "web_search_preview_2025_03_11", "web_search_2025_08_26":
				if tool.Get("external_web_access").Bool() || tool.Get("search_context_size").String() == "high" {
					fallback = "web_search"
					return false
				}
			}
			return true
		})
		if fallback != "" {
			return fallback
		}
	}
	choice := gjson.GetBytes(body, "tool_choice")
	if choice.Exists() && choice.Type == gjson.JSON {
		name := strings.ToLower(choice.Get("name").String())
		if strings.Contains(name, "web_search") || strings.Contains(name, "image_generation") {
			return "tool_choice"
		}
	}
	return ""
}

func (s *OpenAIGatewayService) rewriteBasisPointsRequest(ctx context.Context, c *gin.Context, account *Account, body []byte, token string) ([]byte, error) {
	if account == nil || !account.BasisPointsEnabled() {
		return body, nil
	}
	model := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if !basisPointsRouteEligible(model) {
		return body, nil
	}
	if reason := basisPointsNativeFallbackReason(body); reason != "" {
		if c != nil {
			c.Header(basisPointsBypassHeader, reason)
			logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Basis Points native fallback account_id=%d model=%s reason=%s", account.ID, model, reason)
		}
		return body, nil
	}
	relayed, err := s.relayBasisPointsImages(ctx, c, account, body)
	if err != nil {
		return nil, err
	}
	if relayed != nil {
		body = relayed
	}
	uploaded, err := s.uploadBasisPointsImages(ctx, c, account, body, token)
	if err != nil {
		return nil, err
	}
	structured, err := prepareBasisPointsStructuredOutput(uploaded)
	if err != nil {
		return nil, err
	}
	rewritten, err := prepareBasisPointsBody(uploaded, structured)
	if err != nil {
		return nil, err
	}
	if c != nil {
		c.Set(basisPointsRoutedContextKey, true)
		c.Set(basisPointsSourceBodyKey, append([]byte(nil), body...))
		if structured != nil {
			c.Set(basisPointsStructuredFormatKey, structured)
		}
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Basis Points route account_id=%d model=%s", account.ID, model)
	}
	return rewritten, nil
}

const (
	basisPointsRoutedContextKey    = "openai_basispoints_routed"
	basisPointsSourceBodyKey       = "openai_basispoints_source_body"
	basisPointsStructuredFormatKey = "openai_basispoints_structured_format"
)

func basisPointsRouted(c *gin.Context) bool {
	if c == nil {
		return false
	}
	value, ok := c.Get(basisPointsRoutedContextKey)
	enabled, _ := value.(bool)
	return ok && enabled
}

func basisPointsSourceBody(c *gin.Context) []byte {
	if c == nil {
		return nil
	}
	value, ok := c.Get(basisPointsSourceBodyKey)
	body, _ := value.([]byte)
	if !ok {
		return nil
	}
	return body
}

func basisPointsStructuredFromContext(c *gin.Context) *basisPointsStructuredOutput {
	if c == nil {
		return nil
	}
	value, ok := c.Get(basisPointsStructuredFormatKey)
	format, _ := value.(*basisPointsStructuredOutput)
	if !ok {
		return nil
	}
	return format
}

func (s *OpenAIGatewayService) handleBasisPointsStreamingResponse(_ context.Context, resp *http.Response, c *gin.Context, _ *Account, startTime time.Time, _, _ string) (*openaiStreamingResult, error) {
	raw, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}
	replay, err := replayBasisPointsSSE(raw, basisPointsSourceBody(c), basisPointsStructuredFromContext(c))
	if err != nil {
		return nil, fmt.Errorf("restore basis points client tools: %w", err)
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	if requestID := strings.TrimSpace(resp.Header.Get("x-request-id")); requestID != "" {
		c.Header("x-request-id", requestID)
	}
	flusher, _ := c.Writer.(http.Flusher)
	var firstToken *int
	for _, event := range bytes.Split(replay, []byte("\n\n")) {
		event = bytes.TrimSpace(event)
		if len(event) == 0 {
			continue
		}
		if _, err := c.Writer.Write(append(event, '\n', '\n')); err != nil {
			return nil, err
		}
		if flusher != nil {
			flusher.Flush()
		}
		if firstToken == nil && basisPointsEventHasText(event) {
			firstToken = basisPointsFirstTokenMs(startTime)
		}
	}
	usage := &OpenAIUsage{}
	if parsed, ok := extractOpenAIUsageFromSSE(replay); ok {
		usage = &parsed
	}
	responseID := ""
	for _, line := range bytes.Split(replay, []byte("\n")) {
		data := basisPointsSSEData(line)
		if id := gjson.GetBytes(data, "response.id").String(); id != "" {
			responseID = id
		}
	}
	return &openaiStreamingResult{
		usage:        usage,
		responseID:   responseID,
		firstTokenMs: firstToken,
	}, nil
}

func basisPointsEventHasText(event []byte) bool {
	for _, line := range bytes.Split(event, []byte("\n")) {
		data := basisPointsSSEData(line)
		if len(data) == 0 {
			continue
		}
		eventType := gjson.GetBytes(data, "type").String()
		if strings.Contains(eventType, "output_text.delta") || strings.Contains(eventType, "arguments.delta") {
			return true
		}
	}
	return false
}

func basisPointsSSEData(line []byte) []byte {
	trimmed := bytes.TrimSpace(line)
	if !bytes.HasPrefix(trimmed, []byte("data:")) {
		return nil
	}
	return bytes.TrimSpace(bytes.TrimPrefix(trimmed, []byte("data:")))
}

func basisPointsRewriteError(err error) (int, string, string) {
	switch {
	case err == nil:
		return http.StatusBadRequest, "basispoints_request_invalid", "Invalid Basis Points request"
	case errors.Is(err, basispoints.ErrImageRelayFull):
		return http.StatusServiceUnavailable, "basispoints_image_relay_full", err.Error()
	case errors.Is(err, basispoints.ErrImageRelayStorage):
		return http.StatusServiceUnavailable, "basispoints_image_relay_unavailable", err.Error()
	default:
		return http.StatusBadRequest, "basispoints_request_invalid", err.Error()
	}
}

func basisPointsFirstTokenMs(start time.Time) *int {
	if start.IsZero() {
		return nil
	}
	ms := int(time.Since(start).Milliseconds())
	return &ms
}

func extractOpenAIUsageFromSSE(body []byte) (OpenAIUsage, bool) {
	var usage OpenAIUsage
	found := false
	for _, line := range bytes.Split(body, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if !bytes.HasPrefix(trimmed, []byte("data:")) {
			continue
		}
		data := bytes.TrimSpace(bytes.TrimPrefix(trimmed, []byte("data:")))
		if !gjson.ValidBytes(data) {
			continue
		}
		if parsed, ok := extractOpenAIUsageFromJSONBytes(data); ok {
			usage = parsed
			found = true
		}
		if response := gjson.GetBytes(data, "response"); response.Exists() {
			if parsed, ok := extractOpenAIUsageFromJSONBytes([]byte(response.Raw)); ok {
				usage = parsed
				found = true
			}
		}
	}
	return usage, found
}

func (s *OpenAIGatewayService) relayBasisPointsImages(ctx context.Context, c *gin.Context, account *Account, body []byte) ([]byte, error) {
	if s == nil || !bytes.Contains(body, []byte("data:")) {
		return nil, nil
	}
	relay, err := s.basisPointsImageRelay(ctx)
	if err != nil || relay == nil {
		return nil, err
	}
	scope := basisPointsImageScope(c, account, body)
	rewritten, err := relay.Rewrite(body, scope)
	if err != nil {
		return nil, err
	}
	return rewritten, nil
}

func basisPointsImageScope(c *gin.Context, account *Account, body []byte) string {
	accountID := int64(0)
	if account != nil {
		accountID = account.ID
	}
	threadID := ""
	if c != nil {
		_, threadID = resolveOpenAIWSExecutionScope(c, body, getAPIKeyIDFromContext(c))
	}
	return fmt.Sprintf("account:%d/key:%d/thread:%s", accountID, getAPIKeyIDFromContext(c), threadID)
}

func (s *OpenAIGatewayService) basisPointsImageRelay(ctx context.Context) (*basispoints.ImageRelay, error) {
	return s.excelBPSImageRelay(ctx)
}

func (s *OpenAIGatewayService) CloseBasisPointsImages() error {
	return s.CloseExcelBPSImages()
}

func (s *OpenAIGatewayService) ServeBasisPointsImage(c *gin.Context) {
	s.ServeExcelBPSImage(c)
}

func (s *OpenAIGatewayService) uploadBasisPointsImages(ctx context.Context, c *gin.Context, account *Account, body []byte, token string) ([]byte, error) {
	if !bytes.Contains(body, []byte("data:")) || !gjson.GetBytes(body, "input").IsArray() {
		return body, nil
	}
	proxyURL := ""
	if c != nil {
		proxyURL, _, _ = s.lookupOpenAIProxyURL(ctx, c, account, body)
	}
	changed := false
	items := gjson.GetBytes(body, "input").Array()
	rebuilt := make([]any, 0, len(items))
	for _, item := range items {
		content := item.Get("content")
		if !content.IsArray() || !strings.EqualFold(item.Get("role").String(), "user") {
			var decoded any
			if err := json.Unmarshal([]byte(item.Raw), &decoded); err != nil {
				return nil, err
			}
			rebuilt = append(rebuilt, decoded)
			continue
		}
		parts := make([]any, 0)
		partChanged := false
		for _, part := range content.Array() {
			imageURL := part.Get("image_url").String()
			if part.Get("type").String() != "input_image" || !strings.HasPrefix(strings.ToLower(imageURL), "data:") {
				var decoded any
				if err := json.Unmarshal([]byte(part.Raw), &decoded); err != nil {
					return nil, err
				}
				parts = append(parts, decoded)
				continue
			}
			if part.Get("file_id").String() != "" {
				return nil, fmt.Errorf("basis points input_image cannot contain both image_url and file_id")
			}
			fileID, err := s.uploadBasisPointsImage(ctx, account, token, proxyURL, imageURL)
			if err != nil {
				return nil, err
			}
			copy := map[string]any{}
			if err := json.Unmarshal([]byte(part.Raw), &copy); err != nil {
				return nil, err
			}
			delete(copy, "image_url")
			copy["file_id"] = fileID
			if _, exists := copy["detail"]; !exists {
				copy["detail"] = "auto"
			}
			parts = append(parts, copy)
			partChanged = true
		}
		if !partChanged {
			var decoded any
			if err := json.Unmarshal([]byte(item.Raw), &decoded); err != nil {
				return nil, err
			}
			rebuilt = append(rebuilt, decoded)
			continue
		}
		itemCopy := map[string]any{}
		if err := json.Unmarshal([]byte(item.Raw), &itemCopy); err != nil {
			return nil, err
		}
		itemCopy["content"] = parts
		rebuilt = append(rebuilt, itemCopy)
		changed = true
	}
	if !changed {
		return body, nil
	}
	return sjson.SetBytes(body, "input", rebuilt)
}

func (s *OpenAIGatewayService) uploadBasisPointsImage(ctx context.Context, account *Account, token, proxyURL, dataURL string) (string, error) {
	mediaType, data, err := decodeBasisPointsDataURL(dataURL)
	if err != nil {
		return "", err
	}
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	filename := "image"
	if extensions, _ := mime.ExtensionsByType(mediaType); len(extensions) > 0 {
		filename += extensions[0]
	}
	partHeaders := make(textproto.MIMEHeader)
	partHeaders.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": filename}))
	partHeaders.Set("Content-Type", mediaType)
	part, err := writer.CreatePart(partHeaders)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, basisPointsAttachURL, &payload)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	applyBasisPointsHeaders(req, account)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	resp, err := s.doOpenAIUpstream(req, proxyURL, account)
	if err != nil {
		return "", fmt.Errorf("basis points attachment upload: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("basis points attachment upload HTTP %d", resp.StatusCode)
	}
	fileID := strings.TrimSpace(gjson.GetBytes(responseBody, "openai_file_id").String())
	if fileID == "" {
		return "", fmt.Errorf("basis points attachment upload returned no openai_file_id")
	}
	return fileID, nil
}

func decodeBasisPointsDataURL(dataURL string) (string, []byte, error) {
	metadata, encoded, found := strings.Cut(dataURL[len("data:"):], ",")
	if !found {
		return "", nil, fmt.Errorf("basis points image data url is missing its separator")
	}
	isBase64 := strings.HasSuffix(strings.ToLower(metadata), ";base64")
	if isBase64 {
		metadata = metadata[:len(metadata)-len(";base64")]
	}
	mediaType, _, err := mime.ParseMediaType(metadata)
	if err != nil || !strings.HasPrefix(mediaType, "image/") {
		return "", nil, fmt.Errorf("basis points image data url must declare an image media type")
	}
	var data []byte
	if isBase64 {
		data, err = base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(data) == 0 {
			return "", nil, fmt.Errorf("basis points image data url is empty or invalid")
		}
	} else {
		data = []byte(encoded)
	}
	return mediaType, data, nil
}

func restoreBasisPointsClientTools(c *gin.Context, body []byte) ([]byte, error) {
	if !basisPointsRouted(c) {
		return body, nil
	}
	restored, err := unwrapBasisPointsResponseBody(body, basisPointsSourceBody(c))
	if err != nil {
		return nil, err
	}
	return applyBasisPointsStructuredResponse(restored, basisPointsStructuredFromContext(c))
}

func applyBasisPointsHeaders(req *http.Request, account *Account) {
	if req == nil || account == nil {
		return
	}
	// 出站前 resolve 已经把影子账号换成母账号 ID。这里只在还没有 ID 时补上，
	// 避免影子自己的空 chatgpt_account_id 把母账号 ID 清掉。
	accountID := strings.TrimSpace(req.Header.Get("ChatGPT-Account-ID"))
	if accountID == "" {
		accountID = strings.TrimSpace(account.GetChatGPTAccountID())
	}
	req.Host = basisPointsUpstreamHost
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-ID", accountID)
		req.Header.Set("X-OpenAI-Account-ID", accountID)
	}
	req.Header.Set("X-Basispoints-Auth-Mode", basisPointsAuthMode)
	req.Header.Set("Origin", "https://bps.openai.com")
	req.Header.Set("X-OpenAI-Internal-Basispoints-Client-Agent-Profile", "excel")
	req.Header.Set("X-OpenAI-Internal-Basispoints-Client-Editor", "excel")
	req.Header.Set("X-OpenAI-Internal-Basispoints-Client-Host", "office")
	req.Header.Set("X-OpenAI-Internal-Basispoints-Client-Platform", "excel")
	req.Header.Set("X-OpenAI-Internal-Basispoints-Client-Platform-Class", "PC")
	req.Header.Set("X-OpenAI-Internal-Basispoints-Client-Product", "basispoints-excel-plugin")
	req.Header.Set("X-OpenAI-Internal-Basispoints-Client-Runtime", "desktop")
	req.Header.Set("X-OpenAI-Internal-Basispoints-Office-Host", "Excel")
	req.Header.Set("X-OpenAI-Internal-Basispoints-Office-Platform", "PC")
	req.Header.Set("X-Stainless-Arch", "unknown")
	req.Header.Set("X-Stainless-Lang", "js")
	req.Header.Set("X-Stainless-OS", "Unknown")
	req.Header.Set("X-Stainless-Package-Version", "6.31.0")
	req.Header.Set("X-Stainless-Retry-Count", "0")
	req.Header.Set("X-Stainless-Runtime", "browser:chrome")
	req.Header.Set("User-Agent", "oai-basispoints/0.1.9")
	req.Header.Del("OpenAI-Beta")
	req.Header.Del("originator")
	req.Header.Del("version")
	req.Header.Del("session_id")
	req.Header.Del("conversation_id")
	req.Header.Del("x-codex-turn-state")
	req.Header.Del("x-codex-beta-features")
	req.Header.Del("x-codex-turn-metadata")
	stream := gjson.GetBytes(readRequestBody(req), "stream").Bool()
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Content-Type", "application/json")
}

func readRequestBody(req *http.Request) []byte {
	if req == nil || req.Body == nil {
		return nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

func prepareBasisPointsBody(body []byte, structured ...*basisPointsStructuredOutput) ([]byte, error) {
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("basis points request body is not json")
	}
	// 按白名单重组成 Basis Points 认识的字段。Codex 的 include、text、
	// service_tier、max_output_tokens 留在原请求里会被上游 400。
	items, err := translateBasisPointsInput(body)
	if err != nil {
		return nil, err
	}
	var structuredOutput *basisPointsStructuredOutput
	if len(structured) > 0 {
		structuredOutput = structured[0]
	}
	items = prependBasisPointsPrologue(body, items, structuredOutput)
	out := []byte(`{}`)
	out, err = sjson.SetBytes(out, "model", basisPointsUpstreamModel(gjson.GetBytes(body, "model").String()))
	if err != nil {
		return nil, err
	}
	out, err = sjson.SetBytes(out, "model_selection", "explicit")
	if err != nil {
		return nil, err
	}
	out, err = sjson.SetBytes(out, "stream", gjson.GetBytes(body, "stream").Bool())
	if err != nil {
		return nil, err
	}
	out, err = sjson.SetBytes(out, "store", false)
	if err != nil {
		return nil, err
	}
	out, err = sjson.SetBytes(out, "input", items)
	if err != nil {
		return nil, err
	}
	effort := "medium"
	if raw := strings.TrimSpace(gjson.GetBytes(body, "reasoning.effort").String()); raw != "" {
		effort = normalizeBasisPointsEffort(raw)
	} else if raw := strings.TrimSpace(gjson.GetBytes(body, "reasoning_effort").String()); raw != "" {
		effort = normalizeBasisPointsEffort(raw)
	}
	out, err = sjson.SetBytes(out, "reasoning_effort", effort)
	if err != nil {
		return nil, err
	}
	// 上游只认识 Excel 的 run_officejs。客户端工具目录写进 developer 说明。
	// 空目录时不发 tools。
	if basisPointsHasClientTools(body) {
		out, err = sjson.SetRawBytes(out, "tools", basisPointsTransportTool)
		if err != nil {
			return nil, err
		}
	}
	if cacheKey := strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String()); cacheKey != "" {
		out, err = sjson.SetBytes(out, "prompt_cache_key", cacheKey)
		if err != nil {
			return nil, err
		}
	}
	if policy := gjson.GetBytes(body, "context_management"); policy.Exists() && policy.Type != gjson.Null && !(policy.IsArray() && len(policy.Array()) == 0) {
		out, err = sjson.SetRawBytes(out, "context_management", []byte(policy.Raw))
		if err != nil {
			return nil, err
		}
	}
	conversation := basisPointsConversationKey(body)
	turnFingerprint, iteration := basisPointsTurnState(body)
	metadata := map[string]string{
		"task_id":         basisPointsUUIDv5("sub2api-basispoints/" + conversation),
		"turn_id":         basisPointsUUIDv5("sub2api-basispoints/" + conversation + "/turn/" + turnFingerprint),
		"agent_iteration": iteration,
	}
	if rawMeta := gjson.GetBytes(body, "metadata"); rawMeta.IsObject() {
		rawMeta.ForEach(func(key, value gjson.Result) bool {
			name := key.String()
			if name == "task_id" || name == "turn_id" || name == "agent_iteration" {
				return true
			}
			if len(name) > 64 {
				name = name[:64]
			}
			text := value.String()
			if value.Type != gjson.String {
				text = value.Raw
			}
			if len(text) > 512 {
				text = text[:512]
			}
			metadata[name] = text
			return true
		})
	}
	out, err = sjson.SetBytes(out, "metadata", metadata)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func normalizeBasisPointsEffort(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low":
		return "low"
	case "high":
		return "high"
	case "xhigh", "x-high", "extra-high", "extra_high", "max":
		return "xhigh"
	case "ultra":
		return "ultra"
	default:
		return "medium"
	}
}

func basisPointsConversationKey(body []byte) string {
	for _, path := range []string{"prompt_cache_key", "client_metadata.session_id"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			return value
		}
	}
	input := gjson.GetBytes(body, "input")
	if !input.Exists() {
		return "anonymous"
	}
	sum := sha256.Sum256([]byte(input.Raw))
	return hex.EncodeToString(sum[:8])
}

func basisPointsTurnState(body []byte) (string, string) {
	input := gjson.GetBytes(body, "input")
	if !input.IsArray() {
		sum := sha256.Sum256([]byte(input.Raw))
		return hex.EncodeToString(sum[:]), "1"
	}
	items := input.Array()
	lastUser := 0
	foundUser := false
	for index, item := range items {
		if strings.EqualFold(item.Get("role").String(), "user") {
			lastUser = index
			foundUser = true
		}
	}
	if !foundUser {
		lastUser = 0
	}
	var prefix strings.Builder
	for index := 0; index <= lastUser && index < len(items); index++ {
		prefix.WriteString(items[index].Raw)
	}
	sum := sha256.Sum256([]byte(prefix.String()))
	iteration := 1
	for _, item := range items[lastUser+1:] {
		itemType := item.Get("type").String()
		if itemType == "function_call_output" || itemType == "custom_tool_call_output" {
			iteration++
		}
	}
	return hex.EncodeToString(sum[:]), fmt.Sprintf("%d", iteration)
}

var basisPointsUUIDNamespace = [16]byte{0x6b, 0xa7, 0xb8, 0x11, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}

func basisPointsUUIDv5(name string) string {
	hash := sha1.New()
	_, _ = hash.Write(basisPointsUUIDNamespace[:])
	_, _ = hash.Write([]byte(name))
	digest := hash.Sum(nil)
	digest[6] = (digest[6] & 0x0f) | 0x50
	digest[8] = (digest[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", digest[0:4], digest[4:6], digest[6:8], digest[8:10], digest[10:16])
}

var basisPointsTransportTool = []byte(`[{"type":"function","name":"run_officejs","description":"Transport for one client tool. code is JSON text {\"tool\":\"name\",\"args\":{...}}, not JavaScript.","parameters":{"type":"object","properties":{"summary":{"type":"string"},"extended_summary":{"type":"string"},"code":{"type":"string"},"destructive":{"type":"boolean"},"references":{"type":"array"}},"required":["code"],"additionalProperties":false}}]`)

func basisPointsOmittedHostedTools(body []byte) string {
	tools := gjson.GetBytes(body, "tools")
	if !tools.IsArray() {
		return ""
	}
	seen := map[string]struct{}{}
	var walk func(items []gjson.Result)
	walk = func(items []gjson.Result) {
		for _, tool := range items {
			toolType := strings.ToLower(strings.TrimSpace(tool.Get("type").String()))
			if toolType == "namespace" && tool.Get("tools").IsArray() {
				walk(tool.Get("tools").Array())
				continue
			}
			if basisPointsUnsupportedHostedTool(toolType) {
				seen[toolType] = struct{}{}
			}
		}
	}
	walk(tools.Array())
	if len(seen) == 0 {
		return ""
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func basisPointsUnsupportedHostedTool(kind string) bool {
	switch kind {
	case "web_search", "web_search_preview", "web_search_preview_2025_03_11", "web_search_2025_08_26",
		"tool_search", "image_generation", "file_search", "code_interpreter", "computer", "computer_use_preview", "mcp":
		return true
	default:
		return false
	}
}

func basisPointsHasClientTools(body []byte) bool {
	tools := gjson.GetBytes(body, "tools")
	if !tools.IsArray() {
		return false
	}
	found := false
	var walk func(items []gjson.Result)
	walk = func(items []gjson.Result) {
		for _, tool := range items {
			toolType := strings.ToLower(strings.TrimSpace(tool.Get("type").String()))
			name := strings.TrimSpace(tool.Get("name").String())
			if toolType == "namespace" && name != "" && tool.Get("tools").IsArray() {
				walk(tool.Get("tools").Array())
				continue
			}
			if (toolType == "function" || toolType == "custom") && name != "" {
				found = true
			}
		}
	}
	walk(tools.Array())
	return found
}

func prependBasisPointsPrologue(source []byte, items []any, structured *basisPointsStructuredOutput) []any {
	instructions := strings.TrimSpace(gjson.GetBytes(source, "instructions").String())
	catalog := basisPointsToolCatalog(source)
	prologue := "This request is relayed by an external Responses API client, not by the live Excel workbook. The native run_officejs function is a transport endpoint owned by this proxy. The proxy intercepts it before execution, so it never runs Office code. Other native Excel, Office, connector, workbook, list_skills, and web-search tools are unavailable."
	if catalog != "" {
		prologue += " Call run_officejs once per client tool. Its arguments include summary, extended_summary, destructive=false, references=[], and code. code is JSON text containing exactly one object {\"tool\":\"CLIENT_TOOL\",\"args\":{...}} for a function tool, or {\"tool\":\"CLIENT_TOOL\",\"args\":\"RAW_INPUT\"} for a custom tool. Do not put JavaScript inside code. Available client tools:\n" + catalog
		if omitted := basisPointsOmittedHostedTools(source); omitted != "" {
			prologue += "\nHosted tools unavailable through Basis Points: " + omitted + ". These declarations were omitted. Do not claim to have used them."
		}
		if choice := gjson.GetBytes(source, "tool_choice"); choice.Exists() && choice.Type != gjson.Null {
			prologue += "\nClient tool_choice: " + choice.Raw
		}
		if gjson.GetBytes(source, "parallel_tool_calls").Exists() && !gjson.GetBytes(source, "parallel_tool_calls").Bool() {
			prologue += "\nInvoke at most one client tool in this response."
		}
	} else {
		prologue += " Do not call tools. Return the answer as assistant text."
	}
	prefix := make([]any, 0, 3)
	if instructions != "" {
		prefix = append(prefix, basisPointsMessage("developer", instructions))
	}
	prefix = append(prefix, basisPointsMessage("developer", prologue))
	if prompt := structured.instructions(); prompt != "" {
		prefix = append(prefix, basisPointsMessage("developer", prompt))
	}
	return basisPointsPrependBeforeCompaction(items, prefix)
}

func basisPointsPrependBeforeCompaction(items, prefix []any) []any {
	if len(items) > 0 {
		if last, ok := items[len(items)-1].(map[string]any); ok && fmt.Sprint(last["type"]) == "compaction_trigger" {
			result := append([]any{}, prefix...)
			result = append(result, items[:len(items)-1]...)
			return append(result, items[len(items)-1])
		}
	}
	return append(prefix, items...)
}

// translateBasisPointsInput keeps the history Basis Points already saw.
// A remembered run_officejs call is replayed as-is; a new client tool call is
// wrapped. Reasoning keeps only encrypted_content, and item_reference is dropped.
func translateBasisPointsInput(body []byte) ([]any, error) {
	input := gjson.GetBytes(body, "input")
	if input.Type == gjson.String {
		return []any{basisPointsMessage("user", input.String())}, nil
	}
	if !input.IsArray() {
		return []any{}, nil
	}
	items := make([]any, 0, len(input.Array()))
	origins := map[string]string{}
	for _, item := range input.Array() {
		itemType := strings.ToLower(strings.TrimSpace(item.Get("type").String()))
		switch itemType {
		case "function_call", "custom_tool_call":
			callID := strings.TrimSpace(item.Get("call_id").String())
			if native := rememberedBasisPointsCall(callID); native != nil {
				if callID != "" {
					origins[callID] = basisPointsTransport
				}
				items = append(items, native)
				continue
			}
			name := basisPointsClientToolName(item)
			if isBasisPointsTransportName(name) {
				rememberBasisPointsCall(item)
				if callID != "" {
					origins[callID] = basisPointsTransport
				}
				decoded, err := basisPointsDecodeItem(item)
				if err != nil {
					return nil, err
				}
				items = append(items, decoded)
				continue
			}
			if name != "" && basisPointsCatalogContains(body, name) {
				wrapped, err := basisPointsTransportCall(item)
				if err != nil {
					return nil, err
				}
				if callID != "" {
					origins[callID] = basisPointsTransport
				}
				items = append(items, wrapped)
				continue
			}
			decoded, err := basisPointsDecodeItem(item)
			if err != nil {
				return nil, err
			}
			items = append(items, decoded)
		case "function_call_output", "custom_tool_call_output":
			callID := strings.TrimSpace(item.Get("call_id").String())
			decoded, err := basisPointsDecodeItem(item)
			if err != nil {
				return nil, err
			}
			if origins[callID] == basisPointsTransport || rememberedBasisPointsCall(callID) != nil {
				decoded["type"] = "function_call_output"
				decoded["id"] = basisPointsFunctionItemID(callID)
				delete(decoded, "name")
				delete(decoded, "namespace")
			}
			items = append(items, decoded)
		case "reasoning":
			if encrypted := strings.TrimSpace(item.Get("encrypted_content").String()); encrypted != "" {
				items = append(items, map[string]any{
					"type":              "reasoning",
					"summary":           []any{},
					"encrypted_content": encrypted,
				})
			}
		case "item_reference":
		default:
			decoded, err := basisPointsDecodeItem(item)
			if err != nil {
				return nil, err
			}
			items = append(items, decoded)
		}
	}
	return items, nil
}

func basisPointsDecodeItem(item gjson.Result) (map[string]any, error) {
	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(item.Raw), &decoded); err != nil {
		return nil, err
	}
	delete(decoded, "internal_chat_message_metadata_passthrough")
	return decoded, nil
}

func basisPointsClientToolName(item gjson.Result) string {
	name := strings.TrimSpace(item.Get("name").String())
	if namespace := strings.TrimSpace(item.Get("namespace").String()); namespace != "" {
		return namespace + "." + name
	}
	return name
}

func basisPointsFunctionItemID(callID string) string {
	if callID == "" {
		return ""
	}
	if strings.HasPrefix(callID, "fc_") {
		return callID
	}
	return "fc_" + callID
}

var basisPointsNativeCalls = struct {
	sync.Mutex
	items map[string]map[string]any
	order []string
}{items: map[string]map[string]any{}}

func rememberBasisPointsCall(item gjson.Result) {
	callID := strings.TrimSpace(item.Get("call_id").String())
	if callID == "" {
		return
	}
	decoded, err := basisPointsDecodeItem(item)
	if err != nil {
		return
	}
	basisPointsNativeCalls.Lock()
	defer basisPointsNativeCalls.Unlock()
	if _, exists := basisPointsNativeCalls.items[callID]; !exists {
		basisPointsNativeCalls.order = append(basisPointsNativeCalls.order, callID)
	}
	basisPointsNativeCalls.items[callID] = decoded
	for len(basisPointsNativeCalls.order) > 512 {
		oldest := basisPointsNativeCalls.order[0]
		basisPointsNativeCalls.order = basisPointsNativeCalls.order[1:]
		delete(basisPointsNativeCalls.items, oldest)
	}
}

func rememberedBasisPointsCall(callID string) map[string]any {
	if callID == "" {
		return nil
	}
	basisPointsNativeCalls.Lock()
	defer basisPointsNativeCalls.Unlock()
	item := basisPointsNativeCalls.items[callID]
	if item == nil {
		return nil
	}
	copy := make(map[string]any, len(item))
	for key, value := range item {
		copy[key] = value
	}
	return copy
}

func basisPointsMessage(role, text string) map[string]any {
	contentType := "input_text"
	if role == "assistant" {
		contentType = "output_text"
	}
	return map[string]any{
		"type":    "message",
		"role":    role,
		"content": []any{map[string]any{"type": contentType, "text": text}},
	}
}

func basisPointsToolCatalog(body []byte) string {
	tools := gjson.GetBytes(body, "tools")
	if !tools.IsArray() {
		return ""
	}
	lines := make([]string, 0)
	var walk func(items []gjson.Result, namespace string)
	walk = func(items []gjson.Result, namespace string) {
		for _, tool := range items {
			toolType := strings.ToLower(strings.TrimSpace(tool.Get("type").String()))
			name := strings.TrimSpace(tool.Get("name").String())
			if toolType == "namespace" && name != "" && tool.Get("tools").IsArray() {
				walk(tool.Get("tools").Array(), name)
				continue
			}
			if (toolType != "function" && toolType != "custom") || name == "" {
				continue
			}
			key := name
			if namespace != "" {
				key = namespace + "." + name
			}
			line := "- " + key + " (" + toolType + ")"
			if description := strings.TrimSpace(tool.Get("description").String()); description != "" {
				line += ": " + description
			}
			if toolType == "function" {
				parameters := tool.Get("parameters")
				if !parameters.IsObject() {
					parameters = tool.Get("input_schema")
				}
				if !parameters.IsObject() {
					parameters = tool.Get("inputSchema")
				}
				if parameters.IsObject() {
					line += ". Its arguments are an object with " + basisPointsParameterNames(parameters) + ". JSON Schema: " + parameters.Raw
				}
			} else {
				line += ". It receives raw text in input."
				if format := tool.Get("format"); format.IsObject() {
					line += " Input format: " + format.Raw
				}
			}
			lines = append(lines, line)
		}
	}
	walk(tools.Array(), "")
	return strings.Join(lines, "\n")
}

func basisPointsParameterNames(parameters gjson.Result) string {
	properties := parameters.Get("properties")
	if !properties.IsObject() {
		return "the arguments required by the client"
	}
	required := map[string]bool{}
	for _, name := range parameters.Get("required").Array() {
		required[name.String()] = true
	}
	names := make([]string, 0)
	properties.ForEach(func(key, _ gjson.Result) bool {
		suffix := "optional"
		if required[key.String()] {
			suffix = "required"
		}
		names = append(names, key.String()+" ("+suffix+")")
		return true
	})
	if len(names) == 0 {
		return "the arguments required by the client"
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func basisPointsTransportCall(item gjson.Result) (map[string]any, error) {
	name := basisPointsClientToolName(item)
	callID := strings.TrimSpace(item.Get("call_id").String())
	inner := map[string]any{"tool": name}
	if item.Get("type").String() == "custom_tool_call" {
		inner["args"] = item.Get("input").String()
	} else if args := strings.TrimSpace(item.Get("arguments").String()); args != "" && gjson.Valid(args) {
		var parsed any
		if err := json.Unmarshal([]byte(args), &parsed); err != nil {
			return nil, err
		}
		inner["args"] = parsed
	} else {
		inner["args"] = map[string]any{}
	}
	encoded, err := json.Marshal(inner)
	if err != nil {
		return nil, err
	}
	outer := map[string]any{
		"summary":          "Run client tool " + name,
		"extended_summary": "Relay " + name + " through the external client",
		"code":             string(encoded),
		"destructive":      false,
		"references":       []any{},
	}
	outerJSON, err := json.Marshal(outer)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(item.Get("id").String())
	if id == "" {
		id = basisPointsFunctionItemID(callID)
	}
	return map[string]any{
		"type":      "function_call",
		"id":        id,
		"call_id":   callID,
		"name":      basisPointsTransport,
		"arguments": string(outerJSON),
		"status":    "completed",
	}, nil
}

// unwrapBasisPointsResponseBody turns run_officejs calls back into the client
// tool names from the original request. Responses without that transport are
// returned unchanged.
func unwrapBasisPointsResponseBody(body, source []byte) ([]byte, error) {
	if !bytes.Contains(body, []byte(basisPointsTransport)) {
		return body, nil
	}
	if bodyHasSSEFraming(body) || isBasisPointsSSE(body) {
		return unwrapBasisPointsSSE(body, source)
	}
	return unwrapBasisPointsJSON(body, source)
}

func isBasisPointsSSE(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	return bytes.HasPrefix(trimmed, []byte("event:")) || bytes.HasPrefix(trimmed, []byte("data:"))
}

func unwrapBasisPointsJSON(body, source []byte) ([]byte, error) {
	output := gjson.GetBytes(body, "output")
	if !output.IsArray() {
		return body, nil
	}
	changed := false
	items := make([]any, 0, len(output.Array()))
	for _, item := range output.Array() {
		if item.Get("type").String() == "function_call" && isBasisPointsTransportName(item.Get("name").String()) {
			unwrapped, ok := unwrapBasisPointsCall(item, source)
			if !ok {
				return nil, fmt.Errorf("basis points tool call does not match the client catalog")
			}
			items = append(items, unwrapped)
			changed = true
			continue
		}
		var decoded any
		if err := json.Unmarshal([]byte(item.Raw), &decoded); err != nil {
			return nil, err
		}
		items = append(items, decoded)
	}
	if !changed {
		return body, nil
	}
	return sjson.SetBytes(body, "output", items)
}

func unwrapBasisPointsSSE(body, source []byte) ([]byte, error) {
	var completed []byte
	lines := bytes.Split(body, []byte("\n"))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if !bytes.HasPrefix(trimmed, []byte("data:")) {
			continue
		}
		data := bytes.TrimSpace(bytes.TrimPrefix(trimmed, []byte("data:")))
		if bytes.Equal(data, []byte("[DONE]")) || !gjson.ValidBytes(data) {
			continue
		}
		if gjson.GetBytes(data, "type").String() == "response.completed" {
			if response := gjson.GetBytes(data, "response"); response.Exists() {
				completed = []byte(response.Raw)
			}
		}
	}
	if len(completed) == 0 {
		return body, nil
	}
	unwrapped, err := unwrapBasisPointsJSON(completed, source)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(unwrapped, completed) {
		return body, nil
	}
	var response map[string]any
	if err := json.Unmarshal(unwrapped, &response); err != nil {
		return nil, err
	}
	return syntheticBasisPointsSSE(response), nil
}

// replayBasisPointsSSE forwards text deltas as they arrive and rewrites a
// completed run_officejs call into the client tool once its code is complete.
// Structured answers stay withheld until the terminal response validates.
func replayBasisPointsSSE(raw, source []byte, structured ...*basisPointsStructuredOutput) ([]byte, error) {
	var format *basisPointsStructuredOutput
	if len(structured) > 0 {
		format = structured[0]
	}
	if !isBasisPointsSSE(raw) && !bodyHasSSEFraming(raw) {
		var response map[string]any
		if err := json.Unmarshal(raw, &response); err != nil {
			return nil, fmt.Errorf("basis points stream is not a response object: %w", err)
		}
		return syntheticBasisPointsSSE(response), nil
	}
	var builder strings.Builder
	lines := bytes.Split(raw, []byte("\n"))
	pendingEvent := ""
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			pendingEvent = ""
			continue
		}
		if bytes.HasPrefix(trimmed, []byte("event:")) {
			pendingEvent = strings.TrimSpace(string(bytes.TrimPrefix(trimmed, []byte("event:"))))
			continue
		}
		if !bytes.HasPrefix(trimmed, []byte("data:")) {
			continue
		}
		data := bytes.TrimSpace(bytes.TrimPrefix(trimmed, []byte("data:")))
		if bytes.Equal(data, []byte("[DONE]")) {
			builder.WriteString("data: [DONE]\n\n")
			continue
		}
		if !gjson.ValidBytes(data) {
			writeBasisPointsSSE(&builder, pendingEvent, data)
			continue
		}
		eventType := gjson.GetBytes(data, "type").String()
		if eventType == "" {
			eventType = pendingEvent
		}
		if format != nil && basisPointsStructuredMessageEvent(eventType, gjson.GetBytes(data, "item")) {
			continue
		}
		if strings.Contains(eventType, "function_call_arguments") && isBasisPointsTransportItem(data) {
			if eventType == "response.function_call_arguments.done" || strings.HasSuffix(eventType, ".done") {
				rewritten, ok := rewriteBasisPointsSSEToolDone(data, source)
				if !ok {
					return nil, fmt.Errorf("basis points tool call does not match the client catalog")
				}
				writeBasisPointsSSE(&builder, "response.function_call_arguments.done", rewritten)
			}
			// 增量参数还是 run_officejs 外壳。客户端只看 .done 里的完整参数。
			continue
		}
		if (eventType == "response.output_item.added" || eventType == "response.output_item.done") &&
			isBasisPointsTransportName(gjson.GetBytes(data, "item.name").String()) {
			item := gjson.GetBytes(data, "item")
			rememberBasisPointsCall(item)
			call, ok := unwrapBasisPointsCall(item, source)
			if !ok {
				return nil, fmt.Errorf("basis points tool call does not match the client catalog")
			}
			if eventType == "response.output_item.added" {
				if call["type"] == "custom_tool_call" {
					call["input"] = ""
				} else {
					call["arguments"] = ""
					call["status"] = "in_progress"
				}
			}
			encoded, err := json.Marshal(call)
			if err != nil {
				return nil, err
			}
			rewritten, err := sjson.SetRawBytes(data, "item", encoded)
			if err != nil {
				return nil, err
			}
			writeBasisPointsSSE(&builder, eventType, rewritten)
			continue
		}
		if eventType == "response.completed" || eventType == "response.done" {
			response := gjson.GetBytes(data, "response")
			if response.Exists() {
				unwrapped, err := unwrapBasisPointsJSON([]byte(response.Raw), source)
				if err != nil {
					return nil, err
				}
				unwrapped, err = applyBasisPointsStructuredResponse(unwrapped, format)
				if err != nil {
					return nil, err
				}
				if format != nil {
					if err := writeBasisPointsStructuredMessages(&builder, unwrapped); err != nil {
						return nil, err
					}
				}
				rewritten, err := sjson.SetRawBytes(data, "response", unwrapped)
				if err != nil {
					return nil, err
				}
				writeBasisPointsSSE(&builder, eventType, rewritten)
				continue
			}
		}
		writeBasisPointsSSE(&builder, eventType, data)
	}
	return []byte(builder.String()), nil
}

func isBasisPointsTransportItem(data []byte) bool {
	name := gjson.GetBytes(data, "item.name").String()
	if name == "" {
		name = gjson.GetBytes(data, "name").String()
	}
	return name == "" || isBasisPointsTransportName(name)
}

func rewriteBasisPointsSSEToolDone(data, source []byte) ([]byte, bool) {
	arguments := gjson.GetBytes(data, "arguments").String()
	if arguments == "" {
		arguments = gjson.GetBytes(data, "item.arguments").String()
	}
	callID := gjson.GetBytes(data, "call_id").String()
	if callID == "" {
		callID = "call"
	}
	item := []byte(`{"type":"function_call","name":"run_officejs","arguments":""}`)
	item, _ = sjson.SetBytes(item, "call_id", callID)
	item, _ = sjson.SetBytes(item, "arguments", arguments)
	if id := gjson.GetBytes(data, "item_id").String(); id != "" {
		item, _ = sjson.SetBytes(item, "id", id)
	}
	call, ok := unwrapBasisPointsCall(gjson.ParseBytes(item), source)
	if !ok {
		return nil, false
	}
	rewritten := data
	var err error
	if name, _ := call["name"].(string); name != "" {
		rewritten, err = sjson.SetBytes(rewritten, "name", name)
		if err != nil {
			return nil, false
		}
	}
	if namespace, _ := call["namespace"].(string); namespace != "" {
		rewritten, err = sjson.SetBytes(rewritten, "namespace", namespace)
		if err != nil {
			return nil, false
		}
	}
	if call["arguments"] != nil {
		rewritten, err = sjson.SetBytes(rewritten, "arguments", call["arguments"])
	} else {
		text, _ := call["input"].(string)
		rewritten, err = sjson.SetBytes(rewritten, "arguments", text)
	}
	if err != nil {
		return nil, false
	}
	return rewritten, true
}

func writeBasisPointsSSE(builder *strings.Builder, event string, data []byte) {
	if event != "" {
		builder.WriteString("event: ")
		builder.WriteString(event)
		builder.WriteByte('\n')
	}
	builder.WriteString("data: ")
	builder.Write(data)
	builder.WriteString("\n\n")
}

func syntheticBasisPointsSSE(response map[string]any) []byte {
	var builder strings.Builder
	sequence := 0
	emit := func(event string, value map[string]any) {
		value["type"] = event
		value["sequence_number"] = sequence
		sequence++
		encoded, _ := json.Marshal(value)
		builder.WriteString("event: ")
		builder.WriteString(event)
		builder.WriteString("\ndata: ")
		builder.Write(encoded)
		builder.WriteString("\n\n")
	}
	created := map[string]any{}
	for key, value := range response {
		created[key] = value
	}
	created["status"] = "in_progress"
	created["output"] = []any{}
	emit("response.created", map[string]any{"response": created})
	emit("response.completed", map[string]any{"response": response})
	builder.WriteString("data: [DONE]\n\n")
	return []byte(builder.String())
}

func isBasisPointsTransportName(name string) bool {
	return name == basisPointsTransport || name == basisPointsTransportAlt
}

func unwrapBasisPointsCall(item gjson.Result, source []byte) (map[string]any, bool) {
	arguments := item.Get("arguments").String()
	code := gjson.Get(arguments, "code").String()
	tool, argsRaw, ok := basisPointsEnvelope(code, source)
	if !ok {
		tool, argsRaw, ok = basisPointsRecoveredEnvelope(code, source)
	}
	if !ok || tool == "" || isBasisPointsTransportName(tool) || !basisPointsCatalogContains(source, tool) {
		return nil, false
	}
	name, namespace := tool, ""
	if left, right, found := strings.Cut(tool, "."); found {
		namespace, name = left, right
	}
	callID := item.Get("call_id").String()
	result := map[string]any{
		"type":    "function_call",
		"id":      item.Get("id").String(),
		"call_id": callID,
		"name":    name,
		"status":  "completed",
	}
	if namespace != "" {
		result["namespace"] = namespace
	}
	args := gjson.Parse(argsRaw)
	if args.Type == gjson.String {
		result["type"] = "custom_tool_call"
		result["input"] = args.String()
		delete(result, "status")
		return result, true
	}
	if argsRaw == "" {
		result["arguments"] = "{}"
	} else {
		result["arguments"] = argsRaw
	}
	rememberBasisPointsCall(item)
	return result, true
}

func basisPointsEnvelope(code string, source []byte) (string, string, bool) {
	code = strings.TrimSpace(code)
	if code == "" || !gjson.Valid(code) {
		return "", "", false
	}
	tool := strings.TrimSpace(gjson.Get(code, "tool").String())
	if tool == "" {
		tool = strings.TrimSpace(gjson.Get(code, "name").String())
	}
	if tool == "" || !basisPointsCatalogContains(source, tool) {
		return "", "", false
	}
	args := gjson.Get(code, "args")
	if !args.Exists() {
		args = gjson.Get(code, "arguments")
	}
	return tool, args.Raw, true
}

var basisPointsEmbeddedCall = regexp.MustCompile(`(?:^|\s)(?:await\s+|return\s+)?([A-Za-z_][A-Za-z0-9_.-]*)\s*\(`)

// basisPointsRecoveredEnvelope accepts one unambiguous catalog envelope inside
// a short wrapper such as functions.exec({...}). Multiple candidates, unknown
// tools, and unparseable code stay rejected. The surrounding text is never run.
func basisPointsRecoveredEnvelope(code string, source []byte) (string, string, bool) {
	raw := strings.TrimSpace(code)
	if raw == "" || len(raw) > basisPointsEnvelopeMaxSize {
		return "", "", false
	}
	matches := basisPointsEmbeddedCall.FindAllStringSubmatchIndex(raw, -1)
	if len(matches) == 1 {
		name := raw[matches[0][2]:matches[0][3]]
		if !basisPointsCatalogContains(source, name) {
			name = strings.TrimPrefix(name, "functions.")
		}
		if basisPointsCatalogContains(source, name) {
			open := matches[0][1] - 1
			if open >= 0 && open < len(raw) && raw[open] == '(' {
				if tool, args, ok := basisPointsLeadingEnvelope(raw[open+1:], source); ok {
					return tool, args, true
				}
				// functions.exec({...}) carries the client arguments directly.
				if args, end, ok := basisPointsDecodeJSONValue(raw[open+1:]); ok && end > 0 {
					encoded, err := json.Marshal(args)
					if err == nil && string(encoded) != "null" {
						return name, string(encoded), true
					}
				}
			}
		}
	}
	foundTool, foundArgs := "", ""
	found := false
	for i := 0; i < len(raw); i++ {
		if raw[i] != '{' {
			continue
		}
		tool, args, end, ok := basisPointsLeadingEnvelopeSpan(raw[i:], source)
		if !ok {
			continue
		}
		if found {
			return "", "", false
		}
		foundTool, foundArgs, found = tool, args, true
		if end > 0 {
			i += end - 1
		}
	}
	if !found {
		return "", "", false
	}
	return foundTool, foundArgs, true
}

func basisPointsLeadingEnvelope(raw string, source []byte) (string, string, bool) {
	tool, args, _, ok := basisPointsLeadingEnvelopeSpan(raw, source)
	return tool, args, ok
}

func basisPointsDecodeJSONValue(raw string) (any, int, bool) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, 0, false
	}
	return value, int(decoder.InputOffset()), true
}

func basisPointsLeadingEnvelopeSpan(raw string, source []byte) (string, string, int, bool) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", "", 0, false
	}
	item, ok := value.(map[string]any)
	if !ok || item == nil {
		return "", "", 0, false
	}
	tool := basisPointsText(item["name"])
	alias := basisPointsText(item["tool"])
	if tool != "" && alias != "" && tool != alias {
		return "", "", 0, false
	}
	if tool == "" {
		tool = alias
	}
	if !basisPointsCatalogContains(source, tool) && basisPointsCatalogContains(source, strings.TrimPrefix(tool, "functions.")) {
		tool = strings.TrimPrefix(tool, "functions.")
	}
	if tool == "" || !basisPointsCatalogContains(source, tool) {
		return "", "", int(decoder.InputOffset()), false
	}
	if _, hasArgs := item["arguments"]; hasArgs {
		if _, hasAlias := item["args"]; hasAlias {
			return "", "", 0, false
		}
	}
	args, exists := item["arguments"]
	if !exists {
		args = item["args"]
	}
	encoded, err := json.Marshal(args)
	if err != nil {
		return "", "", 0, false
	}
	if string(encoded) == "null" {
		encoded = nil
	}
	return tool, string(encoded), int(decoder.InputOffset()), true
}

func basisPointsText(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func basisPointsCatalogContains(source []byte, tool string) bool {
	tools := gjson.GetBytes(source, "tools")
	if !tools.IsArray() {
		return false
	}
	var walk func(items []gjson.Result, namespace string) bool
	walk = func(items []gjson.Result, namespace string) bool {
		for _, item := range items {
			toolType := strings.ToLower(item.Get("type").String())
			name := strings.TrimSpace(item.Get("name").String())
			if toolType == "namespace" && name != "" && item.Get("tools").IsArray() {
				if walk(item.Get("tools").Array(), name) {
					return true
				}
				continue
			}
			key := name
			if namespace != "" {
				key = namespace + "." + name
			}
			if key == tool {
				return true
			}
		}
		return false
	}
	return walk(tools.Array(), "")
}
