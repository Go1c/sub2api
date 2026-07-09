---
status: completed
---

# Codex Imagegen Tool Conflict

## Task

修复 Codex `/v1/responses` 图片生成桥接在请求已携带本地函数工具 `image_gen.imagegen` 时仍注入 hosted `image_generation` 工具的问题，并避免后续转换重新注入或追加误导性的 hosted-tool 指令。

## Acceptance

- `tools` 中存在 Responses flat 或 Chat Completions nested 结构的 `image_gen.imagegen` 时，不注入 hosted `image_generation`。
- 仅精确匹配 trim 后的 `image_gen.imagegen`；近似但不同的函数名仍按原行为注入 hosted tool。
- 仅当请求中实际存在 `type: image_generation` 时追加 Codex image-generation bridge marker。
- 图片专用模型规范化不会在同一请求已有本地 `image_gen.imagegen` 时重新注入 hosted tool。
- 真实 `Forward` 链路覆盖 flat / nested 两种本地函数结构，最终上游请求保留本地函数且没有 hosted tool 或 bridge marker。

## Verification

- 先运行新增负控测试，确认在实现收紧前因近似名称被误判而失败（红）。
- 实现最小修复后运行相关 transform 与 gateway 聚焦测试，确认全部通过（绿）。
- 运行 `gofmt` 和 `git diff --check`。

## Knowledge

纯 bugfix，沿用现有 Codex image-generation bridge 设计，不新增知识库文档。

## Verification Results

- 红：`TestEnsureOpenAIResponsesImageGenerationTool_DoesNotMatchSimilarFunctionName` 在修复前按预期失败，证明模糊归一化误匹配近似名称。
- 红：`TestApplyCodexImageGenerationBridgeInstructions_SkipsCodexImageGenFunction` 在修复前按预期失败，证明真实本地工具描述会误触发 hosted bridge marker。
- 红：`TestNormalizeOpenAIResponsesImageOnlyModel_SkipsCodexImageGenFunction` 在修复前按预期失败，证明图片专用模型路径会重新注入；补充最小工具结构后也确认原 validator 未返回显式错误。
- 绿：`go test ./internal/service -run 'TestEnsureOpenAIResponsesImageGenerationTool|TestApplyCodexImageGenerationBridgeInstructions|TestNormalizeOpenAIResponsesImageOnlyModel|TestValidateOpenAIResponsesImageModel|TestOpenAIGatewayServiceForward_CodexImageInjection' -count=1`
- 格式与 diff：相关 Go 文件已运行 `gofmt`，`git diff --check` 通过。
