package service

import (
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/tidwall/gjson"
)

const (
	openAIResponsesEndpoint          = "/v1/responses"
	openAIResponsesCompactEndpoint   = "/v1/responses/compact"
	imageGenerationPermissionMessage = "Image generation is not enabled for this group"
)

// ImageGenerationPermissionMessage returns the stable end-user error text for disabled groups.
func ImageGenerationPermissionMessage() string {
	return imageGenerationPermissionMessage
}

// GroupAllowsImageGeneration preserves ungrouped-key behavior and enforces the flag when a group is present.
func GroupAllowsImageGeneration(group *Group) bool {
	return group == nil || group.AllowImageGeneration
}

// IsImageGenerationIntent classifies requests that can produce generated images.
func IsImageGenerationIntent(endpoint string, requestedModel string, body []byte) bool {
	if IsImageGenerationEndpoint(endpoint) {
		return true
	}
	if isOpenAIImageGenerationModel(requestedModel) {
		return true
	}
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return false
	}
	if model := strings.TrimSpace(gjson.GetBytes(body, "model").String()); isOpenAIImageGenerationModel(model) {
		return true
	}
	if openAIJSONToolsContainImageGeneration(gjson.GetBytes(body, "tools")) {
		return true
	}
	if openAIJSONToolsContainImageGeneration(gjson.GetBytes(body, "functions")) {
		return true
	}
	if openAIJSONInputContainsImageGenTool(gjson.GetBytes(body, "input")) {
		return true
	}
	if openAIJSONToolChoiceSelectsImageGeneration(gjson.GetBytes(body, "tool_choice")) {
		return true
	}
	return openAIJSONToolChoiceSelectsImageGeneration(gjson.GetBytes(body, "function_call"))
}

// IsImageGenerationIntentMap is the map-backed variant used after service-side request mutation.
func IsImageGenerationIntentMap(endpoint string, requestedModel string, reqBody map[string]any) bool {
	if IsImageGenerationEndpoint(endpoint) {
		return true
	}
	if isOpenAIImageGenerationModel(requestedModel) {
		return true
	}
	if reqBody == nil {
		return false
	}
	if isOpenAIImageGenerationModel(firstNonEmptyString(reqBody["model"])) {
		return true
	}
	if toolsContainImageGeneration(reqBody["tools"]) {
		return true
	}
	if openAIAnyCollectionContainsImageGenerationCapability(reqBody["functions"]) {
		return true
	}
	if inputContainsImageGenerationTool(reqBody["input"]) {
		return true
	}
	if openAIAnyToolChoiceSelectsImageGeneration(reqBody["tool_choice"]) {
		return true
	}
	return openAIAnyToolChoiceSelectsImageGeneration(reqBody["function_call"])
}

// IsCodexTextImageGenerationIntent detects Codex /responses requests where the
// user asks for an image but the request has no explicit image model or tool.
func IsCodexTextImageGenerationIntent(endpoint string, requestedModel string, body []byte, userAgent string, originator string, forceCodexCLI bool) bool {
	if !forceCodexCLI && !openai.IsCodexOfficialClientByHeaders(userAgent, originator) {
		return false
	}
	if !isOpenAIResponsesEndpoint(endpoint) {
		return false
	}
	if IsImageGenerationIntent(endpoint, requestedModel, body) {
		return false
	}
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return false
	}
	for _, text := range openAIResponsesUserInputTexts(body) {
		if openAITextLooksLikeImageGenerationPrompt(text) {
			return true
		}
	}
	return false
}

// IsImageGenerationEndpoint identifies dedicated generated-image endpoints.
func IsImageGenerationEndpoint(endpoint string) bool {
	switch normalizeImageGenerationEndpoint(endpoint) {
	case "/v1/images/generations", "/v1/images/edits", "/images/generations", "/images/edits":
		return true
	default:
		return false
	}
}

func isOpenAIResponsesEndpoint(endpoint string) bool {
	switch normalizeImageGenerationEndpoint(endpoint) {
	case "/v1/responses", "/responses":
		return true
	default:
		return false
	}
}

func normalizeImageGenerationEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(strings.ToLower(endpoint))
	if endpoint == "" {
		return ""
	}
	endpoint = strings.TrimPrefix(endpoint, "https://api.openai.com")
	if idx := strings.IndexByte(endpoint, '?'); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return strings.TrimRight(endpoint, "/")
}

func openAIResponsesUserInputTexts(body []byte) []string {
	texts := make([]string, 0, 2)
	appendMessages := func(messages gjson.Result) {
		if !messages.Exists() {
			return
		}
		if messages.Type == gjson.String {
			if text := strings.TrimSpace(messages.String()); text != "" {
				texts = append(texts, text)
			}
			return
		}
		if messages.IsObject() {
			if strings.TrimSpace(messages.Get("role").String()) == "user" {
				appendOpenAIMessageContentTexts(&texts, messages.Get("content"))
			}
			return
		}
		if !messages.IsArray() {
			return
		}
		messages.ForEach(func(_, item gjson.Result) bool {
			if item.Type == gjson.String {
				if text := strings.TrimSpace(item.String()); text != "" {
					texts = append(texts, text)
				}
				return true
			}
			if !item.IsObject() || strings.TrimSpace(item.Get("role").String()) != "user" {
				return true
			}
			appendOpenAIMessageContentTexts(&texts, item.Get("content"))
			return true
		})
	}
	appendMessages(gjson.GetBytes(body, "input"))
	appendMessages(gjson.GetBytes(body, "messages"))
	return texts
}

func appendOpenAIMessageContentTexts(texts *[]string, content gjson.Result) {
	if texts == nil || !content.Exists() {
		return
	}
	if content.Type == gjson.String {
		if text := strings.TrimSpace(content.String()); text != "" {
			*texts = append(*texts, text)
		}
		return
	}
	appendPart := func(part gjson.Result) {
		if part.Type == gjson.String {
			if text := strings.TrimSpace(part.String()); text != "" {
				*texts = append(*texts, text)
			}
			return
		}
		if !part.IsObject() {
			return
		}
		partType := strings.TrimSpace(part.Get("type").String())
		if partType != "" && partType != "input_text" && partType != "text" {
			return
		}
		if text := strings.TrimSpace(part.Get("text").String()); text != "" {
			*texts = append(*texts, text)
		}
	}
	if content.IsObject() {
		appendPart(content)
		return
	}
	if !content.IsArray() {
		return
	}
	content.ForEach(func(_, part gjson.Result) bool {
		appendPart(part)
		return true
	})
}

func openAITextLooksLikeImageGenerationPrompt(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" || openAITextContainsImageGenerationNegation(normalized) {
		return false
	}
	if openAITextContainsCodingMention(normalized) {
		return false
	}
	if openAITextContainsAny(normalized, []string{
		"生图", "出图", "生成一张", "生成一幅", "生成一个图", "画一张", "绘制一张", "做一张", "来一张",
		"generate an image", "generate a picture", "generate a photo", "create an image", "create a picture", "draw an image", "render an image",
	}) {
		return true
	}
	hasImageSubject := openAITextContainsAny(normalized, []string{
		"图片", "图像", "图标", "头像", "海报", "插画", "壁纸", "封面", "表情包", "照片",
		"image", "picture", "photo", "illustration", "artwork", "logo", "poster", "wallpaper", "avatar", "sticker", "icon",
	})
	if !hasImageSubject {
		return false
	}
	return openAITextContainsAny(normalized, []string{
		"生成", "画", "绘制", "创建", "制作", "设计", "做一个", "做张",
		"generate", "create", "draw", "render", "design", "paint", "make",
	})
}

func openAITextContainsImageGenerationNegation(text string) bool {
	return openAITextContainsAny(text, []string{
		"不要生成", "不用生成", "无需生成", "别生成", "不要画", "不用画",
		"do not generate", "don't generate", "not generate", "do not create", "don't create",
	})
}

func openAITextContainsCodingMention(text string) bool {
	return openAITextContainsAny(text, []string{
		"写代码", "代码", "调用示例", "函数", "实现", "开发", "build an app", "write code", "code example",
	})
}

func openAITextContainsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func openAIJSONToolsContainImageGeneration(tools gjson.Result) bool {
	if !tools.IsArray() {
		return false
	}
	found := false
	tools.ForEach(func(_, item gjson.Result) bool {
		if strings.TrimSpace(item.Get("type").String()) == "image_generation" {
			found = true
			return false
		}
		if openAIJSONToolContainsImageGenerationMCP(item) {
			found = true
			return false
		}
		if isImageGenNamespaceTool(item) {
			found = true
			return false
		}
		return true
	})
	return found
}

func isOpenAIImageGenerationType(value string) bool {
	return strings.TrimSpace(value) == "image_generation"
}

func isOpenAIImageGenNamespaceName(value string) bool {
	return strings.TrimSpace(value) == "image_gen"
}

// isImageGenNamespaceTool detects the Codex namespace-style image generation
// tool declaration: { "type": "namespace", "name": "image_gen", ... }.
// Codex /image uses this instead of the flat { "type": "image_generation" }.
func isImageGenNamespaceTool(tool gjson.Result) bool {
	return openAIJSONString(tool.Get("type")) == "namespace" &&
		isOpenAIImageGenNamespaceName(openAIJSONString(tool.Get("name")))
}

// openAIJSONInputContainsImageGenTool scans Responses input items for
// additional_tools entries that declare the image_gen namespace. This covers
// the "Responses Lite" format where tools are embedded inside input items
// rather than top-level tools.
func openAIJSONInputContainsImageGenTool(input gjson.Result) bool {
	if !input.IsArray() {
		return false
	}
	found := false
	input.ForEach(func(_, item gjson.Result) bool {
		if openAIJSONString(item.Get("type")) != "additional_tools" {
			return true
		}
		found = openAIJSONToolsContainImageGeneration(item.Get("tools"))
		return !found
	})
	return found
}

func openAIJSONString(value gjson.Result) string {
	if value.Type != gjson.String {
		return ""
	}
	return strings.TrimSpace(value.String())
}

func openAIJSONToolContainsImageGenerationMCP(tool gjson.Result) bool {
	if !tool.Exists() || !tool.IsObject() {
		return false
	}
	hasImageNamespace := false
	hasGenerateImage := false
	walkOpenAIJSONStrings(tool, func(value string) {
		markOpenAIImageGenerationMCPString(value, &hasImageNamespace, &hasGenerateImage)
	})
	return hasImageNamespace && hasGenerateImage
}

func openAIStringContainsImageGenerationCapability(value string) bool {
	hasImageNamespace := false
	hasGenerateImage := false
	markOpenAIImageGenerationMCPString(value, &hasImageNamespace, &hasGenerateImage)
	return hasImageNamespace && hasGenerateImage
}

func walkOpenAIJSONStrings(value gjson.Result, visit func(string)) {
	if !value.Exists() {
		return
	}
	switch {
	case value.Type == gjson.String:
		visit(value.String())
	case value.IsArray() || value.IsObject():
		value.ForEach(func(_, item gjson.Result) bool {
			walkOpenAIJSONStrings(item, visit)
			return true
		})
	}
}

func markOpenAIImageGenerationMCPString(value string, hasImageNamespace *bool, hasGenerateImage *bool) {
	normalized := normalizeOpenAIImageGenerationIntentString(value)
	if normalized == "" {
		return
	}
	if openAIImageIntentStringHasImageNamespace(normalized) {
		*hasImageNamespace = true
	}
	if openAIImageIntentStringHasImageAction(normalized) {
		*hasGenerateImage = true
	}
}

func normalizeOpenAIImageGenerationIntentString(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}
	return strings.NewReplacer("-", "_", ".", "_", "/", "_", " ", "_", ":", "_").Replace(normalized)
}

func openAIImageIntentStringHasImageNamespace(normalized string) bool {
	switch {
	case strings.Contains(normalized, "image"),
		strings.Contains(normalized, "picture"),
		strings.Contains(normalized, "photo"),
		strings.Contains(normalized, "illustration"),
		strings.Contains(normalized, "artwork"),
		strings.Contains(normalized, "gpt_image"),
		strings.Contains(normalized, "image_2"),
		strings.Contains(normalized, "image2"),
		strings.Contains(normalized, "imagegen"),
		strings.Contains(normalized, "image_gen"),
		strings.Contains(normalized, "txt2img"),
		strings.Contains(normalized, "img2img"),
		strings.Contains(normalized, "dalle"):
		return true
	default:
		return false
	}
}

func openAIImageIntentStringHasImageAction(normalized string) bool {
	switch {
	case strings.Contains(normalized, "image_generation"),
		strings.Contains(normalized, "generate_image"),
		strings.Contains(normalized, "create_image"),
		strings.Contains(normalized, "render_image"),
		strings.Contains(normalized, "draw_image"),
		strings.Contains(normalized, "edit_image"),
		strings.Contains(normalized, "image_edit"),
		strings.Contains(normalized, "variation"),
		strings.Contains(normalized, "upscale"),
		strings.Contains(normalized, "outpaint"),
		strings.Contains(normalized, "inpaint"),
		strings.Contains(normalized, "generate"),
		strings.Contains(normalized, "generation"),
		strings.Contains(normalized, "create"),
		strings.Contains(normalized, "render"),
		strings.Contains(normalized, "draw"),
		strings.Contains(normalized, "paint"),
		strings.Contains(normalized, "edit"),
		strings.Contains(normalized, "transform"):
		return true
	default:
		return false
	}
}

func openAIJSONToolChoiceSelectsImageGeneration(choice gjson.Result) bool {
	if !choice.Exists() {
		return false
	}
	if choice.Type == gjson.String {
		value := strings.TrimSpace(choice.String())
		return value == "image_generation" || openAIStringContainsImageGenerationCapability(value)
	}
	if !choice.IsObject() {
		return false
	}
	choiceType := openAIJSONString(choice.Get("type"))
	if isOpenAIImageGenerationType(choiceType) {
		return true
	}
	if choiceType == "namespace" &&
		(isOpenAIImageGenNamespaceName(openAIJSONString(choice.Get("name"))) ||
			isOpenAIImageGenNamespaceName(openAIJSONString(choice.Get("namespace")))) {
		return true
	}
	if tool := choice.Get("tool"); tool.IsObject() && openAIJSONToolChoiceSelectsImageGeneration(tool) {
		return true
	}
	if isOpenAIImageGenerationType(openAIJSONString(choice.Get("function.name"))) {
		return true
	}
	return openAIJSONToolContainsImageGenerationMCP(choice)
}

func openAIAnyToolChoiceSelectsImageGeneration(choice any) bool {
	switch v := choice.(type) {
	case string:
		value := strings.TrimSpace(v)
		return value == "image_generation" || openAIStringContainsImageGenerationCapability(value)
	case map[string]any:
		choiceType := strings.TrimSpace(firstNonEmptyString(v["type"]))
		if isOpenAIImageGenerationType(choiceType) {
			return true
		}
		if choiceType == "namespace" &&
			(isOpenAIImageGenNamespaceName(firstNonEmptyString(v["name"])) ||
				isOpenAIImageGenNamespaceName(firstNonEmptyString(v["namespace"]))) {
			return true
		}
		if tool, ok := v["tool"].(map[string]any); ok && openAIAnyToolChoiceSelectsImageGeneration(tool) {
			return true
		}
		if fn, ok := v["function"].(map[string]any); ok && isOpenAIImageGenerationType(firstNonEmptyString(fn["name"])) {
			return true
		}
		return openAIAnyToolContainsImageGenerationCapability(v)
	}
	return false
}

func openAIAnyCollectionContainsImageGenerationCapability(value any) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		if openAIAnyToolContainsImageGenerationCapability(item) {
			return true
		}
	}
	return false
}

func openAIAnyToolContainsImageGenerationCapability(tool any) bool {
	if toolMap, ok := tool.(map[string]any); ok {
		if strings.TrimSpace(firstNonEmptyString(toolMap["type"])) == "image_generation" {
			return true
		}
	}
	hasImageNamespace := false
	hasGenerateImage := false
	walkOpenAIAnyStrings(tool, func(value string) {
		markOpenAIImageGenerationMCPString(value, &hasImageNamespace, &hasGenerateImage)
	})
	return hasImageNamespace && hasGenerateImage
}

func walkOpenAIAnyStrings(value any, visit func(string)) {
	switch v := value.(type) {
	case string:
		visit(v)
	case []any:
		for _, item := range v {
			walkOpenAIAnyStrings(item, visit)
		}
	case map[string]any:
		for _, item := range v {
			walkOpenAIAnyStrings(item, visit)
		}
	case map[string]string:
		for _, item := range v {
			visit(item)
		}
	}
}

func getAPIKeyFromContext(c interface{ Get(string) (any, bool) }) *APIKey {
	if c == nil {
		return nil
	}
	v, exists := c.Get("api_key")
	if !exists {
		return nil
	}
	apiKey, _ := v.(*APIKey)
	return apiKey
}

func apiKeyGroup(apiKey *APIKey) *Group {
	if apiKey == nil {
		return nil
	}
	return apiKey.Group
}

func cloneRequestMapForImageIntent(body []byte) map[string]any {
	if len(body) == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil
	}
	return out
}

func resolveOpenAIResponsesImageBillingConfig(reqBody map[string]any, fallbackModel string) (string, string, error) {
	imageModel := ""
	imageSize := ""
	hasImageTool := false
	if reqBody != nil {
		rawTools, _ := reqBody["tools"].([]any)
		for _, rawTool := range rawTools {
			toolMap, ok := rawTool.(map[string]any)
			if !ok || strings.TrimSpace(firstNonEmptyString(toolMap["type"])) != "image_generation" {
				continue
			}
			hasImageTool = true
			imageModel = strings.TrimSpace(firstNonEmptyString(toolMap["model"]))
			imageSize = strings.TrimSpace(firstNonEmptyString(toolMap["size"]))
			break
		}
		if imageSize == "" {
			imageSize = strings.TrimSpace(firstNonEmptyString(reqBody["size"]))
		}
	}
	if imageModel == "" && reqBody != nil {
		bodyModel := strings.TrimSpace(firstNonEmptyString(reqBody["model"]))
		if isOpenAIImageBillingModelAlias(bodyModel) || !hasImageTool {
			imageModel = bodyModel
		}
	}
	if imageModel == "" && hasImageTool {
		imageModel = "gpt-image-2"
	}
	if imageModel == "" {
		imageModel = strings.TrimSpace(fallbackModel)
	}
	sizeTier := normalizeOpenAIImageSizeTier(imageSize)
	return imageModel, sizeTier, nil
}

func resolveOpenAIResponsesImageBillingConfigFromBody(body []byte, fallbackModel string) (string, string, error) {
	reqBody := cloneRequestMapForImageIntent(body)
	return resolveOpenAIResponsesImageBillingConfig(reqBody, fallbackModel)
}

func isOpenAIImageBillingModelAlias(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return false
	}
	return isOpenAIImageGenerationModel(normalized) || strings.Contains(normalized, "image")
}
