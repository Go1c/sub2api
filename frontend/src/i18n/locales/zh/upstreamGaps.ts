// Fills locale leaves missing after the 0.2.17 merge.
// Upstream strings come from Wei-Shaw/sub2api b0a02a330. Keys with no upstream copy use a readable English label.
// Loaded last so existing fork translations stay in place.
export default {
  "admin": {
    "accounts": {
      "accountSchedulingThresholdOverride": "账号自动停调阈值覆盖",
      "accountSchedulingThresholdOverrideDisabledHint": "1-100，达到该用量百分比后临时不可调度；100 表示禁用当前账号自动停调。",
      "accountSchedulingThresholdOverrideHint": "仅对当前账号覆盖平台级自动停调阈值；关闭后使用平台设置。",
      "accountSchedulingThresholdOverrideValue": "账号阈值百分比",
      "audioPreview": "生成音频：",
      "autoResetCredit": {
        "thresholdInvalid": "自动使用重置卡阈值必须在 0.1% 到 100% 之间。"
      },
      "bulkActions": {
        "confirmDelete": "确认删除选中的 {count} 个账号吗？此操作不可恢复。",
        "deleteSuccess": "已成功删除 {count} 个账号",
        "selectAllFailed": "获取全部账号失败，原有选择未改变",
        "selectAllResults": "全选所有结果（{count}）",
        "selectedAll": "已选择全部 {count} 个账号",
        "selectingAll": "正在选择全部结果..."
      },
      "claudeResetCredits": {
        "clears": "可清除窗口：{windows}",
        "confirmMessage": "将消耗 1 次重置次数，立即恢复 {windows} 窗口，剩余 {count} 次。此操作不可撤销，确定继续吗？",
        "confirmTitle": "确认使用 Claude 重置",
        "cooldown": "冷却至 {time}",
        "count": "次数",
        "countTooltipLoad": "点击查询 Claude 剩余重置次数（只读，不会消耗）",
        "countTooltipRefresh": "点击刷新 Claude 剩余重置次数（只读，不会消耗）",
        "error": "无法查询重置次数",
        "expiresAt": "到期 {time}",
        "expiresAtFull": "重置次数到期时间：{time}",
        "fetched": "查询时间：{time}",
        "ineligible": "此账号当前不可使用重置",
        "notUsableNow": "暂不可用",
        "outcome": {
          "busy": "另一个重置正在进行中，请稍后再试",
          "failed": "重置请求失败",
          "inProgress": "该重置请求仍在处理中，请稍后查询结果",
          "notAvailable": "当前没有可立即使用的重置，未消耗次数",
          "retryBackoff": "该重置请求刚刚失败，请稍后再试",
          "unavailable": "重置服务暂时不可用，未确认消耗，请稍后再试",
          "unknown": "结果未确认，已阻止再次兑换，请稍后查询"
        },
        "requiresLimit": "需达到限额后才能使用",
        "reset": "重置",
        "resetTooltipNeedQuery": "请先点「次数」查询；查询到可用的重置后才能使用",
        "resetTooltipNone": "当前没有可立即使用的重置",
        "resetTooltipReady": "消耗 1 次重置，清除限额窗口（需确认）"
      },
      "errorPrefix": "错误：{message}",
      "expiresAtTimezoneHint": "输入按浏览器本地时区（{timezone}）解释。",
      "grok": {
        "audioUploadHint": "上传真实音频做转写；不上传则用静音 WAV 仅测连通。",
        "audioUploadLabel": "音频文件（STT 可选）",
        "chooseAudioFile": "选择音频",
        "chooseImageFile": "选择图片",
        "fileReadFailed": "读取所选文件失败",
        "imageUploadHint": "建议 PNG/JPEG，宽高均 ≥ 8 像素，编辑建议小于约 4MB。上传源图会走 /images/edits（图生图）；不上传则走 /images/generations 文生图。",
        "imageUploadLabel": "源图片（可选，图生图/编辑）",
        "mediaTooLarge": "文件过大（管理端测试上传上限约 6MB）。",
        "noResponseBody": "服务器未返回响应体",
        "realtimeTestHint": "独立 WebSocket 拨号 /v1/realtime（model=grok-voice-latest）。握手成功即连通；若有首包服务端事件会一并显示。",
        "realtimeTestMode": "模式：Realtime（WS /realtime）",
        "searchQueryDefault": "xAI Grok",
        "searchQueryLabel": "搜索关键词",
        "searchQueryPlaceholder": "例如：xAI Grok",
        "searchTestHint": "独立网页搜索探测（与网关 /v1/web_search 语义一致），不是带 tools 的自由对话。",
        "searchTestMode": "模式：网页搜索（/web_search）",
        "selectedTestMode": "测试模式：{mode}",
        "sendingRealtimeRequest": "正在拨号独立 /realtime WebSocket...",
        "sendingSTTRequest": "正在发送独立 /stt 请求...",
        "sendingSearchRequest": "正在发送独立 web_search 请求...",
        "sendingTTSRequest": "正在发送独立 /tts 请求...",
        "sttTestHint": "独立调用 /v1/stt，使用合成静音 WAV；成功表示接口可达。",
        "sttTestMode": "模式：STT（/stt）",
        "testMode": "测试模式",
        "testModeHint": "文本 / 图片 / 视频使用所选模型。网页搜索、TTS、STT、Realtime 走独立接口探测（不是对话里的 tools）。",
        "testModeImage": "图片（/images/generations）",
        "testModeRealtime": "实时语音 Realtime（WS /realtime）",
        "testModeSTT": "语音识别 STT（/stt）",
        "testModeSearch": "网页搜索（/web_search）",
        "testModeTTS": "语音合成 TTS（/tts）",
        "testModeText": "文本（Responses）",
        "testModeVideo": "视频（/videos/generations）",
        "textTestMode": "模式：文本（Responses）",
        "ttsTestHint": "独立调用 /v1/tts（language=en）；成功时显示音频字节数。",
        "ttsTestMode": "模式：TTS（/tts）",
        "ttsTextDefault": "Hello from Sub2API account connectivity test.",
        "ttsTextLabel": "TTS 文本",
        "ttsTextPlaceholder": "例如：Hello from Sub2API connectivity test.",
        "uploadPreviewAlt": "上传预览",
        "videoFirstFrameHint": "可选首帧/参考图用于图生视频。建议 PNG/JPEG，宽高均 ≥ 8 像素。",
        "videoFirstFrameLabel": "首帧 / 参考图（可选）"
      },
      "grokMediaEligibility": {
        "loadFailed": "无法读取媒体资格",
        "partialSave": "账号其他配置可能已保存，但媒体资格未更新，请重试。"
      },
      "imageLightboxAlt": "图片预览",
      "imagePreviewAlt": "测试图片 {index}",
      "modelMappingConflict": "该模型已配置映射 {from} → {to}，请在模型映射中修改或删除后再添加白名单模型",
      "oauth": {
        "grok": {
          "failedToValidateSSO": "校验 Grok SSO 失败",
          "pleaseEnterSSOToken": "请输入 SSO Token"
        }
      },
      "ollamaCloud": {
        "autoRefresh": "自动刷新用量",
        "autoRefreshFailed": "更新自动刷新设置失败",
        "autoRefreshHint": "只有账号开关和全局开关同时启用时才会定时刷新。",
        "balance": "余额",
        "configured": "已配置",
        "deleteConfirm": "确定删除已保存的 Ollama 浏览器会话及其用量快照？",
        "deleteSession": "删除会话",
        "encryptionKeyRequired": "请先配置持久 TOTP_ENCRYPTION_KEY，再保存浏览器会话。",
        "failed": "刷新失败",
        "fiveHour": "5 小时",
        "fiveHourShort": "5h",
        "loadFailed": "加载 Ollama Cloud 用量设置失败",
        "models": "模型",
        "notConfigured": "未配置",
        "notRefreshed": "尚未刷新",
        "ok": "正常",
        "plan": "套餐",
        "refreshFailed": "刷新 Ollama Cloud 用量失败",
        "refreshNow": "刷新用量",
        "refreshSuccess": "Ollama Cloud 用量已刷新",
        "sessionDeleteFailed": "删除 Ollama 浏览器会话失败",
        "sessionDeleted": "Ollama 浏览器会话已删除",
        "sessionLabel": "Ollama 浏览器 Cookie",
        "sessionPlaceholder": "wos-session=...; __Secure-authjs.session-token.0=...",
        "sessionSaveFailed": "保存 Ollama 浏览器会话失败",
        "sessionSaved": "Ollama 浏览器会话已保存",
        "sessionSecurityHint": "浏览器会话会加密落库，且只发送到固定的 Ollama 官方设置页。",
        "sevenDay": "7 天",
        "sevenDayShort": "7d",
        "status": "状态",
        "title": "Ollama Cloud 用量",
        "unauthorized": "会话已过期",
        "updatedAt": "更新时间",
        "windowWithReset": "已用 {percent}，{reset} 重置",
        "writeOnlyHint": "仅写入。已保存内容不可查看，留空不会覆盖。"
      },
      "openai": {
        "flattenNamespaces": "摊平 Codex namespace 工具（兼容）",
        "flattenNamespacesDesc": "默认关闭：/responses 上的 namespace 工具声明原样转发，这正是 ChatGPT Codex 后端期望的形态。仅当该 OAuth 账号指向不认识 namespace 的兼容上游时才开启——摊平会把工具改名为 namespace__tool，使按 functions.<命名空间>.<工具> 寻址的模型（如 gpt-5.6 多智能体）无法调用。压缩（compact）请求不受该开关影响，始终摊平。"
      },
      "openaiReferral": {
        "available": "可邀请",
        "cacheFailed": "实时次数已获取，但缓存保存失败，请重新查询。",
        "checkedAt": "查询时间：{time}，点击刷新",
        "consent": "我已取得此人的同意，可以向其发送邀请。",
        "email": "受邀人邮箱",
        "fromAccount": "邀请账号：",
        "invite": "邀请用户",
        "queryHint": "点击查询剩余可邀请次数",
        "refreshFailed": "邀请已发送，但剩余次数刷新失败，请重新查询。",
        "send": "发送邀请",
        "sendUnknown": "暂时无法确认邀请是否已发送，请先在 Codex 中核对邀请状态，再决定是否重试。",
        "sending": "正在发送…",
        "sent": "邀请已发送至 {email}",
        "shadowHint": "请在母账号上发送邀请。",
        "unavailable": "当前无法邀请用户，可能未满足活动条件或次数已用完。"
      },
      "opencodeGo": {
        "failed": "刷新失败",
        "monthlyShort": "1m",
        "ok": "正常",
        "protocolRules": {
          "add": "添加规则",
          "alsoSupports": "也支持",
          "alsoSupportsHint": "以这些协议进来的请求同协议直通，免去协议转换",
          "hint": "自适应模式下按模型匹配上游协议。支持精确 ID 或末尾 * 通配（如 grok-*、qwen*）；自上而下第一条命中生效。入站协议是模型也支持的协议时同协议直通、不做转换，否则走所选协议。",
          "patternPlaceholder": "grok-* 或 deepseek-v4-flash",
          "remove": "删除规则",
          "restoreDefaults": "恢复默认",
          "title": "模型协议分流"
        },
        "rollingShort": "5h",
        "unauthorized": "会话已过期",
        "weeklyShort": "7d"
      },
      "priorityQuick": {
        "editHint": "点击直接输入；数值越小越优先",
        "failed": "更新优先级失败",
        "lower": "降低优先级（数值 +1）",
        "raise": "提高优先级（数值 -1）"
      },
      "sendingVideoRequest": "正在发送视频生成测试请求...",
      "status": {
        "tempUnschedulableUntil": "预计 {time} 恢复"
      },
      "tempUnschedulable": {
        "multipleErrorCount": "本次不可调度由累计 {count} 次匹配错误触发。",
        "multipleErrorCountInWindow": "{minutes} 分钟内累计发生 {count} 次匹配错误。",
        "multipleErrorTrigger": "{minutes} 分钟内累计 {count} 次匹配错误，达到触发阈值（{threshold}）。",
        "multipleErrorTriggerNoWindow": "累计 {count} 次匹配错误，达到触发阈值（{threshold}）。",
        "scope": "作用范围",
        "scopeAccount": "账户级",
        "scopeAccountDesc": "暂停整个账号的调度",
        "scopeHint": "无法识别请求模型时，模型级规则按账户级处理。",
        "scopeModel": "模型级",
        "scopeModelDesc": "只冷却本次请求的模型"
      },
      "upstreamRequestIdHeader": "上游ID",
      "upstreamRequestIdHeaderHelp": {
        "examplesTitle": "常见取值",
        "intro": "填写直接上游在响应头中声明请求标识的头名，记录到用量明细的“上游ID”列；留空则不记录。",
        "official": "{platform} 官方 API",
        "sub2apiNote": "对应对方用量明细的请求ID列"
      },
      "upstreamRequestIdHeaderPlaceholder": "留空不记录",
      "usageWindow": {
        "estimatedTotalCost": "预计总费用 ${cost}",
        "estimatedTotalCostTooltip": "根据当前窗口费用和使用率估算达到 100% 使用率时的总费用"
      },
      "vertexApiKeyHint": "在 Google Cloud 控制台 Agent Platform 的 API Keys 页面创建。Google 显式缓存接口不接受 API Key，此类账号不参与显式缓存与批量生图；隐式缓存正常生效。",
      "vertexApiKeyProjectIdHint": "留空走 Express mode 全局端点（不区分 Location）；绑定 Service Account 的标准 API Key 可填写 Project ID，按 Project ID + Location 访问。",
      "vertexApiKeyProjectIdPlaceholder": "可选",
      "vertexApiKeyRequired": "请输入 API Key",
      "vertexAuthMode": "认证方式",
      "vertexAuthModeApiKey": "API Key",
      "vertexAuthModeServiceAccount": "Service Account JSON",
      "vertexGeminiDesc": "Service Account / API Key",
      "videoPreview": "生成视频：",
      "videoPromptDefault": "A red ball bouncing once on a white floor, short simple motion.",
      "videoPromptLabel": "视频提示词",
      "videoPromptPlaceholder": "例如：一只红球在白地板上弹跳一次，动作简短。",
      "videoTestHint": "调用独立 /v1/videos/generations，轮询至完成后下载成品视频并在页面上预览。",
      "videoTestMode": "模式：视频生成测试"
    },
    "affiliates": {
      "signupBonusesDescription": "查看邀请注册赠送的发放结果、拒绝原因和风控信息",
      "withdraw": {
        "amount": "提现金额（USD）",
        "amountExceeds": "提现金额不能超过当前可提取额度",
        "amountHint": "填写已在站外实际打款给该用户的金额",
        "amountRequired": "请输入大于 0 的金额",
        "availableQuota": "当前可提取额度",
        "changeUser": "更换用户",
        "fillAll": "全部",
        "frozenHint": "冻结期内的返利不计入可提取额度",
        "noUserFound": "未找到匹配的用户",
        "submit": "确认登记",
        "submitting": "登记中...",
        "title": "登记线下提现",
        "uncertainHint": "上次提交没有收到结果，可能已经登记成功。用户与金额已锁定，重新提交会沿用同一笔登记，已登记时不会重复扣减。",
        "user": "用户",
        "userPlaceholder": "输入邮箱或用户名搜索",
        "warning": "登记后将从该用户的可提取返利额度中扣除，且无法撤销。请确认已完成站外打款。"
      }
    },
    "announcements": {
      "preview": "预览"
    },
    "backup": {
      "archive": {
        "badge": "月度归档",
        "copies": "份",
        "count": "归档保留数量",
        "dates": "归档日期（可多选）",
        "datesHint": "按备份计划的时区与备份开始日期匹配，每个日期归档首份成功的定时备份。",
        "day": "{day} 日",
        "disabledHint": "关闭后停止产生新归档，已有归档继续按原保留方式管理。",
        "done": "完成",
        "enabled": "启用",
        "fallbackHint": "当日无成功备份时，顺延至本月下次成功备份；当月没有所选日期时按月末处理，同一份备份只计一份。",
        "forever": "永久保留",
        "independentHint": "归档不额外执行备份，不占普通备份名额；本月无后续成功备份则不归档。",
        "invalidRetention": "保留天数和份数须为非负整数；非永久归档须填写至少 1 份。",
        "monthEnd": "月末",
        "preview": "每月 {dates} 各归档一份，{retention}。",
        "retainLatest": "合计保留最近 {count} 份",
        "retention": "归档保留方式",
        "selectDates": "请至少选择一个归档日期",
        "selectedDates": "已选择 {count} 个日期",
        "title": "月度归档"
      },
      "columns": {
        "parts": "分卷数"
      },
      "imageStorage": {
        "bucket": "存储桶",
        "bucketInherited": "留空则沿用备份存储桶",
        "description": "开启后，异步生图接口可用，生成结果转存到对象存储，只把短链接写入 Redis。与备份共用同一套 S3 客户端，保存后立即生效，无需重启。",
        "enabled": "启用异步生图",
        "prefix": "Key 前缀",
        "presignExpiryHours": "预签名链接有效期（小时）",
        "publicBaseUrl": "公开访问域名",
        "publicBaseUrlPlaceholder": "留空则返回预签名临时链接",
        "reuseBackupS3": "复用上方备份的 S3 配置（只用不同的存储桶/前缀）",
        "saved": "异步生图对象存储配置已保存",
        "title": "异步生图对象存储"
      },
      "schedule": {
        "ordinaryHint": "达到任一清理条件时，清理最旧的普通备份；月度归档独立保留。",
        "ordinaryRetention": "普通备份保留",
        "preview": "保留效果",
        "previewBoth": "最近 {days} 天内最多保留 {count} 份普通备份。",
        "previewCount": "保留最新 {count} 份普通备份，天数不限。",
        "previewDays": "保留最近 {days} 天的普通备份，份数不限。",
        "previewUnlimited": "普通备份不自动清理。"
      }
    },
    "channels": {
      "form": {
        "cacheWrite1hPrice": "缓存写入（1h）",
        "cacheWrite5mPrice": "缓存写入（5m）",
        "clearReasoningEffortMultipliers": "清空倍率",
        "inclusive": "（含）",
        "maxTokens": "最大",
        "minTokens": "最小",
        "reasoningEffortMultiplierDefault": "默认 1",
        "reasoningEffortMultiplierLabel": "{effort} 思考等级倍率",
        "reasoningEffortMultiplierPositive": "{effort} 思考等级倍率必须为大于 0 的有限数字",
        "reasoningEffortLevelInvalid": "不支持的思考等级：{effort}",
        "reasoningEffortMultipliers": "自定义思考等级倍率（可选）",
        "reasoningEffortMultipliersHint": "按请求最终转发的思考等级，对整次请求的费用与额度消耗应用对应倍率。未配置的等级按 1 倍计费，可与其他计费倍率叠加。"
      }
    },
    "groups": {
      "claudeMaxSimulation": {
        "disabled": "已禁用",
        "enabled": "已启用（模拟 1h 缓存）",
        "hint": "仅调整用量计费日志中的 token 类别。不会持久化每个请求的映射状态。",
        "title": "Claude Max 用量模拟",
        "tooltip": "启用后，对于没有上游缓存写入用量的 Claude 模型，系统会确定性地将 token 映射为少量输入加 1h 缓存创建，同时保持总 token 不变。"
      },
      "codexModelsManifest": {
        "accounts": "选定账号",
        "disabledHint": "未启用：普通列表使用本地映射或默认模型；Codex 优先使用本地目录，无本地目录时由调度器选账。",
        "enable": "使用特定账号获取模型列表",
        "enabledHint": "账号来源限定为当前分组内的 OpenAI 账号，最多选择 10 个。",
        "fallback": "选定账号全部不可用时回退调度器",
        "fallbackHint": "关闭时返回 503 / 上游错误；开启时回退到现有调度器选账路径。",
        "hint": "开启后，普通模型列表与 Codex Model Manifest 均优先从选定账号获取并合并，再应用账号映射和分组列表过滤；限流/过载中的选定账号仍会被使用。",
        "searchEmpty": "未找到匹配账号",
        "searchPlaceholder": "搜索账号（当前分组内 OpenAI 账号）",
        "selectAtLeastOne": "开启固定账号后至少选择一个账号",
        "title": "固定账号获取模型列表"
      },
      "explicitPricing": {
        "description": "分组级 web_search（每千次）与 Voice realtime / TTS / STT 单价（USD）。留空表示未配置。",
        "pricePlaceholder": "可选",
        "searchPricePer1k": "搜索每千次价格（USD）",
        "title": "Grok 搜索与 Voice 定价"
      },
      "form": {
        "addReasoningEffortMapping": "添加映射",
        "addReasoningEffortPair": "添加请求值",
        "maxReasoningEffort": "推理强度上限",
        "maxReasoningEffortHint": "仅限制客户端主动请求的 Anthropic/OpenAI 推理强度；Composite 分组按实际目标平台生效。不会为缺省请求主动开启推理。上限优先级高于推理强度映射。",
        "maxReasoningEffortOverLimit": "超限访问控制",
        "maxReasoningEffortOverLimitDeny": "拒绝访问",
        "maxReasoningEffortOverLimitDowngrade": "超过上限时自动降档",
        "maxReasoningEffortOverLimitHint": "设置上限后生效。自动降档会将超过上限的请求改写为上限值后转发；拒绝访问则直接返回错误。",
        "maxReasoningEffortUnlimited": "不限制（跟随请求）",
        "reasoningEffortFrom": "请求值",
        "reasoningEffortFromPlaceholder": "请选择 A",
        "reasoningEffortMappings": "推理强度映射",
        "reasoningEffortMappingsHint": "类型和模型均可留空，表示匹配全部模型。同一类型和模型下可添加多条请求值映射，例如前缀 gpt 同时将 high、xhigh 转到 medium。转发值可选拒绝，命中对应请求值时直接返回错误。精确优先于前后缀，更长前后缀优先。",
        "reasoningEffortMatchExact": "精确",
        "reasoningEffortMatchPrefix": "前缀",
        "reasoningEffortMatchSuffix": "后缀",
        "reasoningEffortMatchType": "类型",
        "reasoningEffortMatchTypePlaceholder": "全匹配",
        "reasoningEffortModel": "模型",
        "reasoningEffortModelPlaceholder": "留空则全部 / gpt / gpt-5.4",
        "reasoningEffortTo": "转发值",
        "reasoningEffortToDeny": "拒绝",
        "reasoningEffortToPlaceholder": "请选择 B",
        "removeReasoningEffortMapping": "删除映射",
        "removeReasoningEffortPair": "删除请求值"
      },
      "i18nField": {
        "default": "默认",
        "placeholder": "{language} 版，留空则显示默认内容"
      },
      "modelAllowlist": {
        "emptySelectionError": "模型白名单已开启，请至少选择或添加一个模型条目"
      },
      "openaiFast": {
        "force": "强制使用 Fast（priority）",
        "free": "免费 Fast",
        "freeHint": "该分组的 Fast 请求仍使用 priority 档位，但客户实际费用按同一请求的 Standard 价格计算。",
        "hint": "开启后，此分组的 OpenAI 请求会强制写入 service_tier=priority；全局 Fast/Flex 策略仍可过滤或拦截。保存后新请求立即生效，已建立的 WebSocket 会话需重连。",
        "title": "OpenAI Fast 模式"
      },
      "openaiLive": {
        "allow": "允许访问 Live",
        "enableAnyway": "仍然开启",
        "hint": "启用后，此 OpenAI 分组的 API Key 可以创建并控制 Live 语音会话。默认关闭。运行 Sub2API 的服务端必须是 Apple Silicon Mac，并安装官方 ChatGPT App；客户端平台不受限制。",
        "title": "OpenAI Live",
        "unsupportedMessage": "当前 Sub2API 服务端无法生成 Live 所需的设备证明，即使开启也不能使用。是否仍然开启？",
        "unsupportedTitle": "当前服务端不支持 Live"
      },
      "voicePricing": {
        "audioRealtimePerMin": "Realtime 每分钟价格（USD）",
        "audioSttPerHour": "STT 每小时价格（USD）",
        "audioTtsPerMillionChars": "TTS 每百万字符价格（USD）",
        "pricePlaceholder": "可选"
      }
    },
    "ops": {
      "errorDetail": {
        "backToList": "返回列表"
      },
      "outputTps": "单次输出 TPS",
      "outputTpsSamples": "有效样本：{count}",
      "requestDetails": {
        "table": {
          "account": "账号"
        }
      },
      "systemLogs": {
        "retentionDaysCustom": "自定义天数",
        "retentionDaysInvalid": "运维日志须保留 1–3650 天；请求日志须保留 1–3650 天或选择永久保留。",
        "retentionDaysOption": "{days} 天",
        "retentionForever": "永久保留",
        "runtimeConfigLoadFailed": "日志配置加载失败，请刷新后重试。"
      },
      "tooltips": {
        "outputTps": "每条有效用量记录的输出 Token ÷ 总耗时（含首字等待），再按当前时间、平台和分组计算分位数。输出可含推理 Token，不重复相加。P50 为中位数，P5/P10 反映较慢请求；数值越高越快。排除图片、Live、无有效输出或耗时的记录。样本来自仍保留的使用明细；无样本或统计暂不可用时显示 —。"
      }
    },
    "plugins": {
      "bridgeRejected": "插件 UI 消息校验失败",
      "compatibility": "版本兼容性",
      "configTitle": "{name} 配置",
      "configure": "配置",
      "confirmDisable": "确定停用此插件吗？新的 OAuth 请求会立即恢复 Sub2API 原有路径。",
      "confirmUninstall": "确定卸载此插件吗？插件必须先停用。此操作会移除安装文件和配置。",
      "confirmUntested": "该插件兼容当前版本范围，但未声明已测试当前 Sub2API 版本。确定承担风险并启用吗？",
      "currentVersion": "当前 Sub2API",
      "description": "安装和管理独立运行的 OAuth 出站传输插件。API Key 流程不受影响。",
      "disable": "停用",
      "disableSuccess": "插件已停用",
      "empty": "尚未安装插件",
      "emptyHint": "选择本机的 .s2plugin 文件进行安装。Sub2API 不会自动下载第三方插件。",
      "enable": "启用",
      "enableSuccess": "插件已启用",
      "fileRequired": "请选择 .s2plugin 文件",
      "healthy": "运行正常",
      "loadingUI": "正在加载插件配置界面...",
      "menuNotice": "系统设置中的“插件管理”开关仅控制侧边栏菜单显示，不会停止已经加载或正在运行的插件。",
      "noAccountCoupling": "作用域为平台与账号类型，不修改账号数据，也不需要在账号页逐个开启。",
      "onlyOpenAI": "初期能力：仅 OpenAI OAuth 出站传输",
      "recommendedVersion": "建议版本",
      "requiredVersion": "要求范围",
      "rollout": "OAuth 流量比例",
      "runtime": "运行状态",
      "runtimeNotice": "插件安装、启用、停用和配置由 Sub2API 宿主动态处理，通常不需要重启宿主实例。只有宿主版本或宿主配置本身变化时，才按部署方式执行重启。",
      "signature": "包签名",
      "test": "测试",
      "testSuccess": "插件测试通过",
      "title": "插件管理",
      "uiUnavailable": "无法加载插件配置界面",
      "unhealthy": "未运行",
      "uninstall": "卸载",
      "uninstallSuccess": "插件已卸载",
      "upload": "安装插件",
      "uploadHint": "仅接受 .s2plugin 包；默认要求可信发布者签名。",
      "uploadSuccess": "插件安装成功，当前保持停用"
    },
    "redeem": {
      "expiryDateRequired": "请输入有效的过期日期和时间",
      "localTimeZoneHint": "自定义时间按浏览器本地时区（{timezone}）解释。"
    },
    "settings": {
      "aliyunCaptcha": {
        "accessKeyId": "AccessKey ID",
        "accessKeyIdHint": "用于服务端验证的阿里云 AccessKey ID，建议使用仅含验证码权限的 RAM 子账号",
        "accessKeySecret": "AccessKey Secret",
        "accessKeySecretConfiguredHint": "密钥已配置，留空以保留当前值。",
        "accessKeySecretHint": "服务端验证密钥（请保密）",
        "prefix": "身份标（prefix）",
        "prefixHint": "在控制台概览页实例基本信息中获取",
        "region": "服务地域",
        "regionCn": "中国内地",
        "regionHint": "决定前端脚本接入区域与服务端接入点，需与阿里云验证码实例所属地域一致",
        "regionSgp": "新加坡",
        "sceneId": "场景 ID",
        "sceneIdHint": "在阿里云验证码控制台创建验证场景后获取；验证方式（无痕/滑块/拼图）在控制台按场景配置"
      },
      "apiKeyAcl": {
        "forwardedClientIpHeaders": "自定义客户端 IP 请求头",
        "forwardedClientIpHeadersHint": "添加 CDN 或反代请求头名称，解析时优先于内置请求头。",
        "forwardedClientIpHeadersPlaceholder": "X-Client-IP",
        "forwardedClientIpHeadersRiskHint": "源站可被直接访问时，这些原始请求头可被伪造；请先限制源站访问再信任它们。",
        "forwardedClientIpHeaderInvalid": "请输入有效的 HTTP 请求头名称。",
        "forwardedClientIpHeadersLimit": "自定义客户端 IP 请求头最多允许 {max} 个。",
        "removeForwardedClientIpHeader": "移除 {header}"
      },
      "captcha": {
        "description": "登录和注册的机器人防护",
        "enable": "启用人机验证",
        "enableHint": "开启后登录、注册等入口需要通过人机验证",
        "provider": "验证服务商",
        "providerAliyun": "阿里云验证码 2.0",
        "providerTencent": "腾讯天御验证码",
        "providerTurnstile": "Cloudflare Turnstile",
        "title": "人机验证"
      },
      "customMenu": {
        "hideOpenButton": "隐藏“新窗口打开”按钮"
      },
      "features": {
        "channelMonitor": {
          "hideThroughput": "对用户隐藏吞吐速率（RPM / TPM）",
          "hideThroughputHint": "开启后，用户端渠道监控页面与用户 API 不返回 RPM/TPM，避免用「速率 × 时间窗」反推集群规模。管理员仍可见完整指标；错误率、延迟、缓存率照常展示。",
          "hideUserRanking": "对用户隐藏用户排行",
          "hideUserRankingHint": "开启后，用户端渠道监控 V2 不再显示「用户排行」页，用户 API 也不返回排行数据。管理员仍可查看。",
          "mode": "监控模式",
          "modeHint": "默认 V1（主动探测）。仅在需要被动聚合时切换到 V2；同一时间只能启用一种实现。",
          "modeV1": "V1 主动探测",
          "modeV1Hint": "默认模式：按配置的渠道监控项定时发起上游健康检查（产生探测流量）。",
          "modeV2": "V2 被动监控",
          "modeV2Hint": "需主动选择：基于真实网关流量聚合健康指标，不向上游发送探活请求；启用期间 V1 探测停止。"
        }
      },
      "gatewayForwarding": {
        "claudeCodeClientVersion": "Claude Code 客户端版本号",
        "claudeCodeClientVersionHint": "网关伪装为官方 Claude Code CLI 时对上游声明的客户端版本号。留空表示使用自动同步到的官方最新版本；填写后固定为该版本，不再跟随同步。仅在手填值和同步值均不可用时，才回退到环境变量 SUB2API_CLAUDE_CLI_VERSION 或内置版本。",
        "claudeCodeVersionAutoSync": "自动同步 Claude Code 版本号",
        "claudeCodeVersionAutoSyncHint": "每小时从官方发布渠道获取最新版本的 Claude Code 客户端版本号，无需为了跟版本而升级本服务。关闭后停止获取新版本，已同步的版本仍可使用；上方手填版本始终优先。",
        "claudeCodeVersionSyncedValue": "当前同步到：{version}",
        "openaiCodexClientVersion": "Codex 客户端版本号",
        "openaiCodexClientVersionHint": "网关对上游声明的 Codex 客户端版本号，User-Agent 与 version 头同源使用。留空表示使用自动同步到的官方最新稳定版；填写后固定为该版本，不再跟随同步。",
        "openaiCodexClientVersionPlaceholder": "留空则跟随自动同步",
        "openaiCodexVersionAutoSync": "自动同步 Codex 版本号",
        "openaiCodexVersionAutoSyncHint": "每 6 小时从官方仓库获取最新稳定版客户端版本号，无需为了跟版本而升级本服务。关闭后仅使用上方手填版本或内置版本。",
        "openaiCodexVersionSyncedValue": "当前同步到：{version}"
      },
      "ollamaCloudUsage": {
        "debounceHint": "范围 1–60 分钟。最后一次模型请求安静满此时长后再抓取用量。",
        "debounceMinutes": "请求安静等待（分钟）",
        "description": "在模型请求驱动下刷新账号在 Ollama 官方设置页展示的用量；默认关闭。无新请求时不会自动抓取。",
        "enabled": "启用全局自动刷新",
        "enabledHint": "仅刷新已保存浏览器会话且账号自身也开启自动刷新的账号；需有后续模型请求才会触发。手动刷新不受影响。",
        "intervalHint": "范围 15–1440 分钟。请求持续不断导致 debounce 一直后移时，最晚在此时间强制刷新。",
        "intervalMinutes": "持续请求最长等待（分钟）",
        "saveFailed": "保存 Ollama Cloud 用量刷新设置失败",
        "saved": "Ollama Cloud 用量刷新设置已保存",
        "title": "Ollama Cloud 用量刷新"
      },
      "openaiExperimentalScheduler": {
        "oauthRateInvalid": "OAuth 调度参考倍率必须是非负数字，或留空以使用账号倍率。"
      },
      "openaiFastPolicy": {
        "summaryOtherModels": "其他模型",
        "tierMissing": "省略 tier",
        "tierUltrafast": "ultrafast"
      },
      "opencodeGoUsage": {
        "debounceHint": "范围 1–60 分钟，且必须小于刷新间隔。最后一次模型请求安静满此时长后再抓取用量。",
        "debounceMinutes": "请求安静等待（分钟）",
        "description": "刷新上游 OpenCode Go 账号上报的用量窗口；默认关闭，仅对单独开启的账号生效。",
        "enabled": "启用全局自动刷新",
        "enabledHint": "仅刷新账号自身也开启自动刷新的账号。手动刷新不受影响。",
        "intervalHint": "范围 5–1440 分钟。请求持续不断导致 debounce 一直后移时，最晚在此时间强制刷新。",
        "intervalMinutes": "请求持续时的最长等待（分钟）",
        "saveFailed": "保存 OpenCode Go 用量刷新设置失败",
        "saved": "OpenCode Go 用量刷新设置已保存",
        "title": "OpenCode Go 用量刷新"
      },
      "panelRateLimit": {
        "description": "限制面板 API 的请求频率，防止高频刷接口（如用量统计、仪表盘查询）打爆数据库",
        "enabled": "启用面板接口限流",
        "enabledHint": "对登录后的面板接口按账号限流；超出阈值返回 429，窗口重置后自动恢复。",
        "exemptAdmin": "管理员豁免",
        "exemptAdminHint": "开启后管理员账号不受面板限流约束，避免批量运维操作被误拦。",
        "heavyRpm": "重查询请求上限",
        "heavyRpmHint": "单个账号每分钟允许的用量/仪表盘等聚合统计查询次数（这类请求对数据库压力最大）；0 表示不限制。",
        "perMinute": "次/分钟",
        "proxySafeNote": "登录后的接口按「用户账号」维度计数，与来源 IP 无关——反向代理、NAT 共享出口等场景不会被误拦截；公开接口按真实客户端 IP 计数，回环与内网地址（反代内部转发地址）会自动跳过。",
        "publicIpRpm": "公开接口每 IP 上限",
        "publicIpRpmHint": "无需登录的公开接口（如站点公开设置）每个真实客户端 IP 每分钟的请求上限；0 表示不限制。",
        "title": "面板接口限流",
        "saved": "面板接口限流配置已保存",
        "saveFailed": "保存面板接口限流配置失败",
        "userRpm": "每账号请求上限",
        "userRpmHint": "单个账号每分钟允许的面板 API 请求总数，正常页面操作远达不到该阈值；0 表示不限制。"
      },
      "payment": {
        "alipayMobilePrecreateDeepLink": "支付宝移动端当面付唤起",
        "alipayMobilePrecreateDeepLinkHint": "启用后，移动端官方支付宝订单调用当面付并尝试打开支付宝；失败时显示动态二维码。该设置优先于强制二维码支付",
        "rechargeBonus": {
          "addTier": "添加档位",
          "empty": "尚未配置优惠档位，充值按原价到账。",
          "hint": "余额充值按用户输入的金额命中档位（取不超过该金额的最大档）；不配置则无优惠。订阅订单不参与。",
          "incompleteRow": "金额与百分比都填写后该档位才会生效",
          "label": "充值优惠阶梯",
          "minAmountLabel": "充值金额 ≥",
          "modeBonus": "赠金",
          "modeBonusHint": "赠金：实付不变，在到账基数（输入金额 × 充值倍率）之上额外赠送对应百分比的余额。",
          "modeDiscount": "折扣 OFF",
          "modeDiscountHint": "折扣：到账不变（输入金额 × 充值倍率），实付金额按对应百分比打折；百分比必须小于 100。",
          "modeLabel": "优惠方式",
          "noticeHint": "支持 Markdown，展示在充值页金额选择区顶部；留空则不展示。",
          "noticeLabel": "充值赠送活动文案",
          "noticePlaceholder": "例如：🎁 限时活动：单笔充值满 $100 送 20%，满 $500 送 30%……",
          "percentLabel": "赠送",
          "percentLabelDiscount": "优惠",
          "previewOpen": "≥ {from}：赠送 {percent}%",
          "previewOpenDiscount": "≥ {from}：{percent}% OFF",
          "previewOpenNone": "≥ {from}：不赠送",
          "previewRange": "{from} ~ {to}：赠送 {percent}%",
          "previewRangeDiscount": "{from} ~ {to}：{percent}% OFF",
          "previewRangeNone": "{from} ~ {to}：不赠送",
          "previewTitle": "区间预览",
          "removeTier": "删除档位"
        }
      },
      "registration": {
        "emailDomainQuota": "非白名单域名限量注册",
        "emailDomainQuotaHint": "开启后，白名单非空时，其他可注册主域名各限注册一个账户；关闭时非白名单域名直接拒绝注册。白名单为空时本开关无效果"
      },
      "site": {
        "compactHome": "简洁首页",
        "compactHomeHint": "未设置自定义首页内容时，展示简洁的站点信息页面。"
      },
      "tencentCaptcha": {
        "aidEncryptedHint": "当前版本暂不支持 aidEncrypted，请先不要在验证码控制台开启 CaptchaAppId 强制校验。",
        "appCredentialsHint": "CaptchaAppId 与 AppSecretKey 来自验证码控制台的验证管理页面。",
        "appCredentialsTitle": "验证码应用密钥",
        "appId": "CaptchaAppId",
        "appSecretKey": "AppSecretKey",
        "camPermissionHint": "推荐创建 CAM 子用户并授予 QcloudCaptchaFullAccess，避免使用主账号永久密钥。",
        "cloudCredentialsHint": "SecretId 与 SecretKey 用于服务端调用 DescribeCaptchaResult 接口。",
        "cloudCredentialsTitle": "云 API 调用密钥",
        "cloudSecretId": "腾讯云 SecretId",
        "cloudSecretKey": "腾讯云 SecretKey",
        "configured": "已配置，留空不会覆盖。",
        "createCloudKeys": "创建 SecretId / SecretKey",
        "keepExisting": "留空以保留当前值",
        "openCaptchaConsole": "打开验证码控制台",
        "openWebDocs": "查看 Web 接入文档",
        "region": "服务站点",
        "regionCn": "中国站",
        "regionHint": "决定前端加载的 SDK 脚本与服务端校验接入点，需与 CaptchaAppId 所属站点一致；国际站请在 tencentcloud.com 控制台创建验证",
        "regionIntl": "国际站",
        "required": "启用前必须填写此项。"
      },
      "upstreamBillingProbe": {
        "description": "定期获取 API Key 账号所连接上游 Sub2API 站点声明的计费倍率；只有另行开启“同步上游声明倍率”的账号才会更新账号倍率。",
        "enabled": "启用全局自动探测",
        "enabledHint": "开启后，仅对账号自身已启用自动检测的账号执行定时探测；关闭后停止所有定时探测，手动探测不受影响。",
        "intervalHint": "范围 5–1440 分钟。成功探测结果的有效期为两个探测周期。",
        "intervalMinutes": "探测周期（分钟）",
        "saveFailed": "保存上游倍率自动探测设置失败",
        "saved": "上游倍率自动探测设置已保存",
        "title": "上游倍率自动探测"
      }
    },
    "subscriptions": {
      "batchAssign": {
        "enable": "批量分配订阅",
        "hint": "搜索并添加多个用户，统一分配所选分组和有效期。每次最多 100 人。",
        "removeUser": "移除 {email}",
        "result": "分配完成：成功 {success} 人，失败 {failed} 人",
        "retryHint": "已移除成功项，可检查失败原因后重新提交剩余用户。",
        "selected": "已添加 {count} 位用户"
      },
      "bulk": {
        "clearSelection": "清空选择",
        "confirm": "确认执行",
        "confirmTargets": "本次将处理以下 {count} 条订阅",
        "extendHint": "正整数延长，负整数缩短，最多调整 36500 天。已过期订阅从当前时间起延长，不能缩短；缩短后的到期时间必须在未来。",
        "groupFallback": "分组 #{id}",
        "invalidDays": "请输入 -36500 到 36500 之间的非零整数天数",
        "itemFailed": "操作失败",
        "requestFailed": "批量操作请求失败，请重试",
        "resetHint": "所选窗口的用量将归零，并从今天开始重新计算。",
        "resetWindows": "选择要重置的配额窗口",
        "result": "处理完成：成功 {success} 条，失败 {failed} 条",
        "retry": "重试原操作",
        "retryHint": "暂未确认操作结果。重试会继续原操作，避免重复处理。也可关闭后重新选择相同订阅和参数重试。",
        "selectSubscription": "选择订阅 #{id}",
        "selectWindow": "请至少选择一个配额窗口",
        "selected": "已选择 {count} 条订阅",
        "selectionHint": "当前页选择，翻页或筛选会清空，最多 100 条。各操作仅处理适用状态的订阅。",
        "selectionLimit": "每次最多操作 100 条订阅",
        "selectionRequired": "请至少选择一条订阅"
      },
      "failedToRestore": "恢复订阅失败",
      "hoursMinutesRemaining": "剩余 {hours} 小时 {minutes} 分钟",
      "minutesRemaining": "剩余 {minutes} 分钟",
      "restore": "恢复",
      "restoreConfirm": "确定要恢复 '{user}' 的订阅吗？如果原订阅已过期，恢复后将显示为已过期。",
      "restoreSubscription": "恢复订阅",
      "subscriptionRestored": "订阅已恢复"
    },
    "usage": {
      "upstreamRequestId": "上游ID",
      "upstreamRequestIdCopied": "上游ID已复制"
    },
    "users": {
      "bulkDelete": {
        "confirm": "确定删除已选的 {count} 个用户吗？此操作无法撤销。管理员账号无法删除。",
        "failed": "{count} 个用户删除失败，已保留选中，可重试。",
        "success": "已删除 {count} 个用户",
        "title": "删除已选用户"
      },
      "concurrencyNonNegative": "并发数不能为负数，0 表示不限制",
      "form": {
        "concurrencyHint": "该用户的最大并发请求数，0 = 不限制",
        "concurrencyPlaceholder": "0 表示不限制"
      },
      "publicGroupsRestricted": "公开分组（已限制）",
      "restrictPublicGroups": "限制可访问的公开分组",
      "restrictPublicGroupsHint": "开启后，该用户仅能使用下方勾选的公开分组；关闭则可使用全部公开分组。"
    }
  },
  "auth": {
    "captchaClickToVerify": "点击完成人机验证",
    "captchaLoading": "正在加载验证码…",
    "captchaVerified": "验证已完成",
    "captchaVerifying": "验证中…",
    "emailDomainRegistrationLimit": "该邮箱域名无法注册新账户。请使用主流邮箱注册；如需使用企业邮箱，请联系客服添加域名白名单。",
    "passkeyCancelled": "已取消 Passkey 登录。",
    "passkeyFailed": "Passkey 登录失败，请重试。"
  },
  "channelMonitorV2": {
    "bootstrap": {
      "description": "首次启用被动监控时，系统会在后台静默聚合 90 分钟、24 小时、7 天与 30 天窗口；完成后可切换全部时间范围。",
      "progress": "进度 {percent}%",
      "title": "正在补齐历史监控数据",
      "working": "后台聚合中…"
    },
    "bucket": {
      "days": "{count} 天粒度",
      "hours": "{count} 小时粒度",
      "minutes": "{count} 分钟粒度"
    },
    "chart": {
      "cacheDataset": "缓存率趋势 %",
      "cacheLegend": "缓存率（左轴 %）",
      "description": "平滑趋势：错误率 · 首 Token P50 · 缓存率",
      "emptyTitle": "当前筛选窗口没有趋势数据",
      "errorDataset": "错误率趋势 %",
      "errorLegend": "错误率（左轴 %）",
      "percentAxis": "比率 %",
      "resetZoom": "重置缩放",
      "title": "可用性趋势",
      "ttftDataset": "首 Token 趋势 P50 (ms)",
      "ttftLegend": "首 Token P50（右轴）"
    },
    "clearFilters": "重置",
    "currentUser": "当前用户",
    "detailLoadFailed": "渠道监控明细加载失败",
    "empty": {
      "description": "尝试调整时间范围或筛选条件",
      "title": "没有可展示的数据"
    },
    "errorDetail": {
      "empty": "仅展示分类占比（样本消息仅管理员可见）",
      "http": "HTTP {code}",
      "noMessage": "无错误消息",
      "upstream": "上游 {code}"
    },
    "filters": {
      "allGroups": "全部",
      "allModels": "全部",
      "allPlatforms": "全部",
      "empty": "暂无可选项",
      "group": "分组",
      "labelValue": "{label}：{value}",
      "model": "模型",
      "platform": "平台",
      "selectedCount": "{count} 项"
    },
    "groupBy": {
      "label": "展示维度",
      "platform": "平台",
      "platformGroup": "平台 / 分组",
      "platformGroupModel": "平台 / 分组 / 模型",
      "platformModel": "平台 / 模型"
    },
    "healthMode": {
      "cache": "缓存率",
      "label": "健康显示",
      "overall": "综合",
      "success": "错误率",
      "ttft": "首 Token"
    },
    "ignored": "忽略",
    "loadFailed": "渠道监控加载失败",
    "matrix": {
      "bad": "差",
      "criticalLegend": "异常 (<50)",
      "description": "每行是一种渠道组合，每个色块代表一个统计区间；悬停查看明细",
      "dimension": "渠道维度",
      "emptyTitle": "当前筛选窗口没有矩阵数据",
      "good": "好",
      "healthyLegend": "健康 (≥80)",
      "legendAria": "健康分数图例",
      "noTraffic": "该区间无流量",
      "noTrafficAt": "{time} · 无流量",
      "resetZoom": "重置缩放",
      "scoreLine": "健康分 {score}",
      "title": "可用性趋势",
      "unknownLegend": "无流量 / 样本不足",
      "warningLegend": "需关注 (50–79)",
      "wheelZoomX": "在色块上滚轮放大（区间变窄、色块变宽）"
    },
    "metrics": {
      "cacheDetail": "读缓存占比",
      "cacheRate": "缓存率",
      "cacheRateValue": "缓存率 {value}",
      "durationValue": "请求时长 {value}",
      "errorRateValue": "错误率 {value}",
      "rpm": "RPM",
      "rpmDetail": "每分钟请求数",
      "rpmValue": "RPM {value}",
      "successRate": "成功率",
      "successRateValue": "成功率 {value}",
      "tps": "每秒 Token",
      "tpsDetail": "由 TPM ÷ 60 换算",
      "tpsValue": "每秒 Token {value}",
      "ttft": "首 Token",
      "ttftP50": "首 Token P50",
      "ttftValue": "首 Token {value}"
    },
    "otherModels": "其他模型",
    "partialCoverage": "部分历史覆盖",
    "ranges": {
      "24h": "24h",
      "30d": "30d",
      "7d": "7d",
      "90m": "90m"
    },
    "rank": {
      "bronze": "第 3 名 铜",
      "gold": "第 1 名 金",
      "place": "第 {n} 名",
      "silver": "第 2 名 银",
      "unranked": "未上榜"
    },
    "settings": {
      "addPlatform": "添加平台",
      "badgeAllModels": "全部模型",
      "badgeOther": "+ 其他",
      "description": "配置被动用量汇总维度（平台 / 模型 / 分组）与刷新频率。健康色与明细在用户端 /monitor 以比例、RPM/TPM 展示，不暴露绝对请求量。",
      "enableHint": "在系统模式为 V2 时生效；关闭后仅停止本配置的汇总，系统模式开关仍在「功能开关」",
      "enableTitle": "启用 V2 汇总",
      "errorsHint": "勾选「忽略」的类别不计入错误率与健康分，仍在错误原因列表中以灰色显示并标记忽略。未匹配的错误归入「其他」。",
      "errorsTitle": "错误分类与忽略",
      "fields": {
        "criticalCache": "缓存率异常 %",
        "criticalError": "错误率异常 %",
        "criticalTtft": "TTFT 异常 ms",
        "minimumSample": "最小样本数",
        "targetTtft": "TTFT 目标 ms",
        "warningCache": "缓存率关注 %",
        "warningError": "错误率关注 %",
        "warningTtft": "TTFT 关注 ms"
      },
      "groupsAll": "全部分组",
      "groupsEmpty": "没有可选择的分组",
      "groupsSelected": "已选择 {count} 个分组",
      "groupsTitle": "监控分组",
      "healthHint": "控制用户端色块和整体评分。默认阈值较宽松，避免少量错误或低缓存率立即显示异常。",
      "healthTitle": "健康阈值",
      "ignoredSummary": "已忽略 {ignored} 类 · 计入错误率 {counted} 类",
      "loadFailed": "V2 配置加载失败",
      "loading": "加载中...",
      "modeBanner": "当前系统设置为 {mode}。V2 分钟聚合不会运行；此处配置可预先保存，切换到 {modeV2} 后立即生效。可在系统设置 → 功能开关调整。",
      "modeClosed": "渠道监控已关闭",
      "modeV1": "V1 主动探测",
      "modeV2": "V2 被动监控",
      "modelsPlaceholder": "留空=全部真实模型；或填写主流模型名单（其余归其他）",
      "namedModelsCount": "将展示 {count} 个命名模型维度；名单外模型归入各平台「其他」。",
      "namedModelsEmpty": "各平台模型列表为空：将展示全部真实模型名（不归入「其他」）。",
      "platformsHint": "留空 = 展示全部真实模型名；填写后仅名单内单独成行，其余归入「其他」",
      "platformsTitle": "平台与模型",
      "refreshAria": "汇总频率",
      "refreshHint": "影响矩阵时间粒度与刷新节奏",
      "refreshTitle": "汇总频率",
      "save": "保存",
      "saveFailed": "V2 配置保存失败",
      "saveSuccess": "V2 监控配置已保存",
      "selectPlatform": "选择平台",
      "title": "V2 数据监控配置",
      "userContract": {
        "health": "健康色三指标：错误率 60% + 首 Token P50 20% + 缓存率 20%（阈值可在上方配置）",
        "latency": "延迟展示 AVG · P50 · P90；不展示绝对请求数 / 错误数",
        "models": "模型列表留空时展示真实模型名，不会全部归入「其他」",
        "trend": "趋势可切换色块矩阵 / 折线图（错误率 · 缓存率 · 首 Token）"
      },
      "userContractTitle": "用户端展示约定"
    },
    "summaryAria": "筛选范围整体汇总",
    "table": {
      "platformModel": "平台 / 模型",
      "rank": "排名",
      "user": "用户"
    },
    "tabs": {
      "aria": "明细维度",
      "errors": "错误原因",
      "models": "模型",
      "users": "用户排行"
    },
    "timeRange": "时间范围",
    "title": "渠道监控",
    "trendView": {
      "label": "趋势视图",
      "line": "折线图",
      "pulse": "色块矩阵"
    },
    "updatedTo": "更新至 {time}",
    "updating": "正在更新数据"
  },
  "common": {
    "fileReadFailed": "读取文件失败",
    "fileTooLargeKb": "文件过大（{size} KB），上限 {max} KB",
    "noFileSelected": "未选择文件",
    "selectImageFile": "请选择图片文件",
    "selectedFile": "已选：{name}",
    "upload": "上传"
  },
  "keyUsage": {
    "billingType": "计费方式"
  },
  "keys": {
    "bulkEdit": {
      "apply": "应用到 {count} 个密钥",
      "clearSelection": "取消选择",
      "failureHint": "以下密钥更新失败，可修改设置后重试。再次提交只会更新失败的密钥。",
      "hint": "勾选需要修改的字段，未勾选的字段保持原值。",
      "invalidExpiration": "请选择有效的过期时间，或勾选永久有效。",
      "invalidLimit": "请输入大于或等于 0 的有效金额。",
      "ipHint": "每行一个 IP 或 CIDR；留空将清空所选密钥的此项名单。",
      "limitHint": "输入 0 表示不限制；已用额度保持不变。",
      "partialFailure": "已更新 {success} 个密钥，{failed} 个失败",
      "selectKey": "选择密钥 {name}",
      "selectedCount": "已选择 {count} 个密钥",
      "success": "已更新 {count} 个密钥",
      "title": "批量编辑"
    },
    "providerLabel": "厂商",
    "useKeyModal": {
      "claudeSettingsHint": "用户级持久配置。此文件包含 API 密钥，请勿提交到项目仓库。",
      "cliTabs": {
        "systemOne": "System One"
      },
      "codexModelCatalog": {
        "description": "Codex 会使用配置中的认证信息加载并刷新远程目录。使用本地文件模式时，请在下方获取目录并保存到配置中的路径。",
        "download": "下载目录",
        "errorDescription": "无法使用当前 API Key 获取模型目录。",
        "fetch": "获取目录",
        "local": "本地文件（旧版客户端）",
        "mode": "目录来源",
        "modelsCount": "已获取 {count} 个模型",
        "oversized": "完整目录超过远程加载的 1 MiB 限制，已改为本地文件。请下载目录并保存到配置中的路径。",
        "remote": "远程目录（Codex 0.156.0+）",
        "retry": "重试",
        "title": "Codex 模型目录"
      },
      "routedCodex": {
        "description": "使用当前路由分组的完整模型目录配置 Codex。",
        "note": "启动 Codex 前先导出 SUB2API_API_KEY。下载的目录只包含模型元数据，不包含 API Key。"
      }
    }
  },
  "modelPlaza": {
    "anonymousHint": "登录后可查看你的专属分组与专属倍率",
    "badges": {
      "exclusive": "专属分组",
      "subscription": "订阅"
    },
    "description": "按分组浏览可用模型与价格",
    "detail": {
      "longContextDisabledNote": "该分组未启用长上下文阶梯计费，超阈值请求仍按基础档计费，官方阶梯仅供参考",
      "noModels": "该分组暂未配置模型",
      "peakNote": "高峰时段 {window} 计费倍率 ×{multiplier}"
    },
    "empty": "暂无可展示的分组",
    "filters": {
      "all": "全部",
      "groupLabel": "分组",
      "modelLabel": "模型",
      "platformLabel": "平台",
      "rateLabel": "倍率",
      "searchPlaceholder": "搜索模型名称"
    },
    "loadFailed": "加载模型广场失败",
    "nav": {
      "backToDashboard": "回到后台",
      "login": "登录"
    },
    "noSearchResult": "没有匹配的模型",
    "table": {
      "cache": "缓存",
      "cacheRead": "读取",
      "cacheReadShort": "读",
      "cacheWrite": "写入",
      "cacheWriteShort": "写",
      "input": "输入",
      "marginalBadge": "超出部分计价",
      "model": "模型",
      "officialPrice": "官方价格",
      "output": "输出",
      "paidPrice": "实付价格(折后)",
      "perImage": "按图片计费",
      "perRequest": "按次计费",
      "perUnitImage": "/ 张",
      "perUnitRequest": "/ 次",
      "rate": "折扣倍率",
      "reasoningMultiplierBadge": "{effort} ×{multiplier}",
      "reasoningMultiplierHint": "最终转发的思考等级为 {effort} 时，整次请求的计费与额度消耗乘以 {multiplier}；未配置的等级按 1 倍计费",
      "tierHint": "按单次请求的总上下文（输入 + 缓存写入 + 缓存读取）所在档位对整单计价",
      "tierHintMarginal": "仅超过阈值的部分按该档计价，输出不加价",
      "timePricingRateHint": "生效倍率 {rate} × 时段倍率 {multiplier}",
      "timePricingRowHintPeak": "；本行价格未含高峰倍率，与高峰时段 {window} 重叠的部分实付再乘 ×{multiplier}",
      "timePricingWeekdays": "工作日",
      "unitPerMillion": "$ / 1M token"
    },
    "title": "模型广场"
  },
  "nav": {
    "modelPlaza": "模型广场",
    "plugins": "插件管理",
    "recharge": "充值",
    "subscribe": "订阅"
  },
  "payment": {
    "billingUnavailable": "充值与订阅均暂未开放，请联系管理员。",
    "orders": {
      "bonusAmount": "赠送额度"
    },
    "qr": {
      "alipayContinueInApp": "请在支付宝中完成支付",
      "alipayFallbackHint": "可重新打开支付宝，或保存下方二维码后从支付宝相册识别",
      "alipayFallbackTitle": "打开支付宝未成功",
      "alipayOpening": "正在打开支付宝",
      "alipaySaveAndScanHint": "保存二维码后，打开支付宝扫一扫，从相册选择二维码",
      "alipayWaitingHint": "支付结果将由服务端确认，本页面会自动更新",
      "reopenAlipay": "重新打开支付宝",
      "saveQRCode": "保存二维码"
    },
    "rechargeBonus": {
      "creditedShort": "到账 {amount}",
      "giftTag": "送 {amount}",
      "payShort": "实付 {amount}"
    }
  },
  "profile": {
    "passkey": {
      "add": "添加 Passkey",
      "addFailed": "添加 Passkey 失败。",
      "added": "Passkey 已添加。",
      "continue": "创建 Passkey",
      "createdAt": "创建于 {date}",
      "deleteConfirm": "删除“{name}”？删除后将无法再使用它登录。",
      "deleteFailed": "删除 Passkey 失败。",
      "deleteTitle": "删除 Passkey",
      "deleted": "Passkey 已删除。",
      "description": "使用面容 ID、触控 ID、Windows Hello 或安全密钥免密码登录。",
      "empty": "尚未添加任何 Passkey。",
      "featureDisabled": "管理员尚未配置 Passkey 功能。",
      "lastUsed": "上次使用 {date}",
      "loadFailed": "加载 Passkey 失败。",
      "name": "Passkey 名称",
      "namePlaceholder": "例如：MacBook 触控 ID",
      "passwordPlaceholder": "输入当前登录密码以确认",
      "renameFailed": "重命名 Passkey 失败。",
      "renamePrompt": "请输入新的 Passkey 名称",
      "renamed": "Passkey 已重命名。",
      "synced": "已同步",
      "title": "Passkey",
      "unsupported": "当前浏览器或设备不支持 Passkey。"
    }
  },
  "purchase": {
    "rechargeDescription": "通过内嵌页面完成充值",
    "subscriptionDescription": "通过内嵌页面完成订阅"
  },
  "redeem": {
    "historyLoadFailed": "加载兑换记录失败，请重试。",
    "userRefreshFailed": "兑换成功，但账户信息刷新失败。"
  },
  "setup": {
    "redis": {
      "username": "用户名（可选）",
      "usernamePlaceholder": "默认用户留空"
    }
  },
  "usage": {
    "live": "Live",
    "modelMismatch": "模型不一致",
    "modelVariant": "疑似版本变体",
    "nativeCompactionV2": "压缩",
    "requestedReasoningEffort": "请求推理强度",
    "sentUpstreamModel": "发往上游",
    "upstreamModelMismatch": "上游响应模型不一致",
    "upstreamResponseModel": "上游响应"
  }
}
