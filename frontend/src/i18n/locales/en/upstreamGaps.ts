// Fills locale leaves missing after the 0.2.17 merge.
// Upstream strings come from Wei-Shaw/sub2api b0a02a330. Keys with no upstream copy use a readable English label.
// Loaded last so existing fork translations stay in place.
export default {
  "admin": {
    "accounts": {
      "accountCreatedSuccess": "Account Created Success",
      "accountDeletedSuccess": "Account Deleted Success",
      "accountSchedulingThresholdOverride": "Account Auto-Pause Threshold Override",
      "accountSchedulingThresholdOverrideDisabledHint": "Use 1-100. The account becomes temporarily unschedulable after reaching this usage percent; 100 disables it for this account.",
      "accountSchedulingThresholdOverrideHint": "Override the platform auto-pause threshold for this account only. Disable to use platform settings.",
      "accountSchedulingThresholdOverrideValue": "Account threshold percent",
      "accountUpdatedSuccess": "Account Updated Success",
      "audioPreview": "Generated audio:",
      "autoResetCredit": {
        "thresholdInvalid": "Automatic reset-credit thresholds must be between 0.1% and 100%."
      },
      "bulkActions": {
        "confirmDelete": "Delete the selected {count} account(s)? This action cannot be undone.",
        "deleteSuccess": "Deleted {count} account(s)",
        "selectAllFailed": "Failed to load all accounts. The previous selection was kept.",
        "selectAllResults": "Select all results ({count})",
        "selectedAll": "All {count} account(s) selected",
        "selectingAll": "Selecting all results..."
      },
      "claudeResetCredits": {
        "clears": "Clears windows: {windows}",
        "confirmMessage": "This will consume 1 reset credit to immediately restore the {windows} window(s) ({count} remaining). This action cannot be undone. Continue?",
        "confirmTitle": "Confirm Claude Reset",
        "cooldown": "Cooldown until {time}",
        "count": "Resets",
        "countTooltipLoad": "Check remaining Claude resets (read-only, never consumes one)",
        "countTooltipRefresh": "Refresh remaining Claude resets (read-only, never consumes one)",
        "error": "Could not check reset credits",
        "expiresAt": "Expires {time}",
        "expiresAtFull": "Reset credit expires at: {time}",
        "fetched": "Checked at {time}",
        "ineligible": "This account cannot use resets right now",
        "notUsableNow": "Not usable now",
        "outcome": {
          "busy": "Another reset is in progress; try again later",
          "failed": "Reset request failed",
          "inProgress": "This reset request is still processing; check again shortly",
          "notAvailable": "No reset can be used right now; no credit was used",
          "retryBackoff": "This reset request just failed; retry after a moment",
          "unavailable": "Reset service is temporarily unavailable; retry after a while",
          "unknown": "Result unconfirmed; further redemption is blocked for now. Check again later"
        },
        "requiresLimit": "Usable only after hitting a limit",
        "reset": "Reset",
        "resetTooltipNeedQuery": "Check the count first; reset is available once a usable credit is found",
        "resetTooltipNone": "No reset can be used right now",
        "resetTooltipReady": "Consume 1 reset to clear limit windows (asks for confirmation)"
      },
      "cookieRefreshedSuccess": "Cookie Refreshed Success",
      "deleteConfirmMessage": "Delete Confirm Message",
      "errorPrefix": "Error: {message}",
      "expiresAtTimezoneHint": "Input is interpreted in your browser time zone ({timezone}).",
      "failedToSave": "Failed To Save",
      "filters": {
        "allPlatforms": "All Platforms",
        "allStatuses": "All Statuses",
        "allTypes": "All Types",
        "platform": "Platform",
        "status": "Status",
        "type": "Type"
      },
      "form": {
        "credentialsLabel": "Credentials Label",
        "credentialsPlaceholder": "Credentials Placeholder",
        "nameLabel": "Name Label",
        "namePlaceholder": "Name Placeholder",
        "platformLabel": "Platform Label",
        "priorityHint": "Priority Hint",
        "priorityLabel": "Priority Label",
        "selectPlatform": "Select Platform",
        "selectType": "Select Type",
        "statusLabel": "Status Label",
        "typeLabel": "Type Label",
        "weightHint": "Weight Hint",
        "weightLabel": "Weight Label"
      },
      "grok": {
        "audioUploadHint": "Upload a real audio clip to transcribe. Without a file, a silent WAV is used for connectivity only.",
        "audioUploadLabel": "Audio file (optional for STT)",
        "chooseAudioFile": "Choose audio",
        "chooseImageFile": "Choose image",
        "fileReadFailed": "Failed to read the selected file",
        "imageUploadHint": "PNG/JPEG recommended, both sides ≥ 8 px, under ~4 MB for edits. Uploading a source image switches to /images/edits (image-to-image). Leave empty for text-to-image /images/generations.",
        "imageUploadLabel": "Source image (optional, for edits)",
        "mediaTooLarge": "File is too large (max ~6 MB for admin test uploads).",
        "noResponseBody": "No response body from server",
        "realtimeTestHint": "Standalone WebSocket dial to /v1/realtime (model=grok-voice-latest). Handshake success = connectivity OK; may also show the first server event.",
        "realtimeTestMode": "Mode: Realtime (WS /realtime)",
        "searchQueryDefault": "xAI Grok",
        "searchQueryLabel": "Search query",
        "searchQueryPlaceholder": "Example: xAI Grok",
        "searchTestHint": "Standalone web_search probe (same as gateway /v1/web_search). Not a free-form chat with tools.",
        "searchTestMode": "Mode: Web search (/web_search)",
        "selectedTestMode": "Test mode: {mode}",
        "sendingRealtimeRequest": "Dialing standalone /realtime WebSocket...",
        "sendingSTTRequest": "Sending standalone /stt request...",
        "sendingSearchRequest": "Sending standalone web_search request...",
        "sendingTTSRequest": "Sending standalone /tts request...",
        "sttTestHint": "Standalone /v1/stt with a synthetic silent WAV; success means the endpoint is reachable.",
        "sttTestMode": "Mode: STT (/stt)",
        "testMode": "Test mode",
        "testModeHint": "Text / image / video use the selected model. Web search, TTS, STT and Realtime hit standalone endpoints (not free-form chat tools).",
        "testModeImage": "Image (/images/generations)",
        "testModeRealtime": "Realtime (WS /realtime)",
        "testModeSTT": "STT (/stt)",
        "testModeSearch": "Web search (/web_search)",
        "testModeTTS": "TTS (/tts)",
        "testModeText": "Text (Responses)",
        "testModeVideo": "Video (/videos/generations)",
        "textTestMode": "Mode: Text (Responses)",
        "ttsTestHint": "Standalone /v1/tts with language=en; success reports audio byte size.",
        "ttsTestMode": "Mode: TTS (/tts)",
        "ttsTextDefault": "Hello from Sub2API account connectivity test.",
        "ttsTextLabel": "TTS text",
        "ttsTextPlaceholder": "Example: Hello from Sub2API connectivity test.",
        "uploadPreviewAlt": "Upload preview",
        "videoFirstFrameHint": "Optional first-frame / reference image for image-to-video. PNG/JPEG recommended, both sides ≥ 8 px.",
        "videoFirstFrameLabel": "First-frame / reference image (optional)"
      },
      "grokMediaEligibility": {
        "loadFailed": "Unable to load media eligibility",
        "partialSave": "Other account settings may have been saved, but media eligibility was not updated. Please retry."
      },
      "imageLightboxAlt": "Image preview",
      "imagePreviewAlt": "Test image {index}",
      "modelMappingConflict": "A mapping already exists for {from} → {to}. Modify or remove it under Model Mapping before adding this whitelist model",
      "noAccounts": "No Accounts",
      "noAccountsDescription": "No Accounts Description",
      "oauth": {
        "grok": {
          "failedToValidateSSO": "Failed to validate Grok SSO",
          "pleaseEnterSSOToken": "Please enter an SSO token"
        }
      },
      "ollamaCloud": {
        "autoRefresh": "Automatic usage refresh",
        "autoRefreshFailed": "Failed to update automatic usage refresh",
        "autoRefreshHint": "Runs only when the account switch and the global switch are both enabled.",
        "balance": "Balance",
        "configured": "Configured",
        "deleteConfirm": "Delete the stored Ollama browser session and its usage snapshot?",
        "deleteSession": "Delete session",
        "encryptionKeyRequired": "Set a persistent TOTP_ENCRYPTION_KEY before storing a browser session.",
        "failed": "Refresh failed",
        "fiveHour": "5 hour",
        "fiveHourShort": "5h",
        "loadFailed": "Failed to load Ollama Cloud usage settings",
        "models": "Models",
        "notConfigured": "Not configured",
        "notRefreshed": "Not refreshed",
        "ok": "Current",
        "plan": "Plan",
        "refreshFailed": "Failed to refresh Ollama Cloud usage",
        "refreshNow": "Refresh usage",
        "refreshSuccess": "Ollama Cloud usage refreshed",
        "sessionDeleteFailed": "Failed to delete Ollama browser session",
        "sessionDeleted": "Ollama browser session deleted",
        "sessionLabel": "Ollama browser Cookie",
        "sessionPlaceholder": "wos-session=...; __Secure-authjs.session-token.0=...",
        "sessionSaveFailed": "Failed to save Ollama browser session",
        "sessionSaved": "Ollama browser session saved",
        "sessionSecurityHint": "The browser session is encrypted at rest and sent only to the fixed official settings URL.",
        "sevenDay": "7 day",
        "sevenDayShort": "7d",
        "status": "Status",
        "title": "Ollama Cloud usage",
        "unauthorized": "Session expired",
        "updatedAt": "Updated",
        "windowWithReset": "{percent} used, resets {reset}",
        "writeOnlyHint": "Write-only. The saved value cannot be viewed and an empty value never replaces it."
      },
      "openai": {
        "flattenNamespaces": "Flatten Codex namespace tools (compatibility)",
        "flattenNamespacesDesc": "Disabled by default: Codex namespace tool declarations are forwarded as-is on /responses, which is what the ChatGPT Codex backend expects. Enable only when this OAuth account is routed to a relay that rejects namespace tools — flattening renames them to namespace__tool, which breaks models that address collaboration tools as functions.<namespace>.<tool>. Compaction requests always flatten regardless of this switch."
      },
      "openaiReferral": {
        "available": "Invites left",
        "cacheFailed": "Live capacity was fetched, but the cache could not be saved. Query again.",
        "checkedAt": "Checked: {time}. Click to refresh.",
        "consent": "I have this person’s consent to send them an invitation.",
        "email": "Recipient email",
        "fromAccount": "Inviting account:",
        "invite": "Invite user",
        "queryHint": "Click to query remaining invitations",
        "refreshFailed": "The invitation was sent, but remaining capacity could not be refreshed. Query again.",
        "send": "Send invitation",
        "sendUnknown": "The invitation outcome is unknown. Check its status in Codex before deciding whether to retry.",
        "sending": "Sending…",
        "sent": "Invitation sent to {email}",
        "shadowHint": "Send invitations from the parent account.",
        "unavailable": "Invitations are unavailable. Eligibility requirements may not be met, or the limit has been reached."
      },
      "opencodeGo": {
        "failed": "Refresh failed",
        "monthlyShort": "1m",
        "ok": "Current",
        "protocolRules": {
          "add": "Add rule",
          "alsoSupports": "Also supports",
          "alsoSupportsHint": "Requests arriving on one of these protocols are passed through unchanged, avoiding protocol conversion",
          "hint": "In adaptive mode, each model is sent to a native upstream protocol. Use an exact ID or a trailing * glob (e.g. grok-*, qwen*). The first matching rule wins. If the inbound protocol is one the model also supports, the request passes through on that protocol without conversion; otherwise the selected protocol is used.",
          "patternPlaceholder": "grok-* or deepseek-v4-flash",
          "remove": "Remove rule",
          "restoreDefaults": "Restore defaults",
          "title": "Model protocol routing"
        },
        "rollingShort": "5h",
        "unauthorized": "Session expired",
        "weeklyShort": "7d"
      },
      "platforms": {
        "composite": "Composite"
      },
      "priorityQuick": {
        "editHint": "Click to type a value; lower is used first",
        "failed": "Failed to update priority",
        "lower": "Lower priority (value +1)",
        "raise": "Raise priority (value -1)"
      },
      "refreshCookie": "Refresh Cookie",
      "refreshing": "Refreshing",
      "saving": "Saving",
      "sendingVideoRequest": "Sending video generation request...",
      "status": {
        "tempUnschedulableUntil": "Resumes {time}"
      },
      "tempUnschedulable": {
        "multipleErrorCount": "{count} matching errors contributed to this block.",
        "multipleErrorCountInWindow": "{count} matching errors occurred within {minutes} minutes.",
        "multipleErrorTrigger": "{count} matching errors in {minutes} minutes reached the trigger threshold ({threshold}).",
        "multipleErrorTriggerNoWindow": "{count} matching errors reached the trigger threshold ({threshold}).",
        "scope": "Scope",
        "scopeAccount": "Account",
        "scopeAccountDesc": "Pause scheduling for the whole account",
        "scopeHint": "Model-scoped rules fall back to account scope when the request model cannot be identified.",
        "scopeModel": "Model",
        "scopeModelDesc": "Cool down only the requested model"
      },
      "testAccount": "Test Account",
      "testSuccess": "Test Success",
      "types": {
        "api_key": "Api key",
        "cookie": "Cookie"
      },
      "upstreamRequestIdHeader": "Upstream ID",
      "upstreamRequestIdHeaderHelp": {
        "examplesTitle": "Common values",
        "intro": "Name of the response header in which the direct upstream declares its request ID. The value is recorded in the \"Upstream ID\" column of the usage log; leave empty to record nothing.",
        "official": "{platform} official API",
        "sub2apiNote": "Matches the request ID column of its usage log"
      },
      "upstreamRequestIdHeaderPlaceholder": "Leave empty to record nothing",
      "usageWindow": {
        "estimatedTotalCost": "Est. total ${cost}",
        "estimatedTotalCostTooltip": "Estimated total cost at 100% utilization, based on current window cost and utilization"
      },
      "vertexApiKeyHint": "Create it on the Agent Platform API Keys page in the Google Cloud console. Google's explicit context caching API does not accept API keys, so these accounts do not serve explicit caching or batch image generation; implicit caching still applies.",
      "vertexApiKeyProjectIdHint": "Leave empty to use the express mode global endpoint (location is ignored). For a standard API key bound to a service account, enter the Project ID to call the project endpoint with Project ID + Location.",
      "vertexApiKeyProjectIdPlaceholder": "Optional",
      "vertexApiKeyRequired": "Please enter an API key",
      "vertexAuthMode": "Authentication",
      "vertexAuthModeApiKey": "API Key",
      "vertexAuthModeServiceAccount": "Service Account JSON",
      "vertexGeminiDesc": "Service Account / API Key",
      "videoPreview": "Generated video:",
      "videoPromptDefault": "A red ball bouncing once on a white floor, short simple motion.",
      "videoPromptLabel": "Video prompt",
      "videoPromptPlaceholder": "Example: A red ball bouncing once on a white floor, short simple motion.",
      "videoTestHint": "Calls standalone /v1/videos/generations, polls until done, then downloads the finished video for on-page preview.",
      "videoTestMode": "Mode: Video generation test"
    },
    "affiliates": {
      "signupBonusesDescription": "Review signup-bonus grants, rejection reasons, and risk-control details.",
      "withdraw": {
        "amount": "Withdrawal amount (USD)",
        "amountExceeds": "Amount cannot exceed the available quota",
        "amountHint": "Enter the amount already paid to this user outside the site",
        "amountRequired": "Enter an amount greater than 0",
        "availableQuota": "Available quota",
        "changeUser": "Change user",
        "fillAll": "All",
        "frozenHint": "Rebates still in the freeze period are not included in the available quota",
        "noUserFound": "No matching users",
        "submit": "Confirm",
        "submitting": "Recording...",
        "title": "Record Offline Withdrawal",
        "uncertainHint": "The last submission returned no result and may already be recorded. The user and amount are locked; submitting again retries the same registration and never deducts twice.",
        "user": "User",
        "userPlaceholder": "Search by email or username",
        "warning": "Recording deducts this amount from the user's available affiliate quota and cannot be undone. Make sure the off-site payment is complete."
      }
    },
    "announcements": {
      "preview": "Preview"
    },
    "backup": {
      "archive": {
        "badge": "Monthly archive",
        "copies": "copies",
        "count": "Number of archives to keep",
        "dates": "Archive dates (select multiple)",
        "datesHint": "Archive the first successful scheduled backup for each selected date, using its start date and the schedule time zone.",
        "day": "Day {day}",
        "disabledHint": "Disabling stops new archives. Existing archives keep their previous retention policy.",
        "done": "Done",
        "enabled": "Enable",
        "fallbackHint": "If no backup succeeds on a selected date, use the next successful backup in that month. Missing dates use month end. Count each backup only once.",
        "forever": "Keep forever",
        "independentHint": "Archives reuse scheduled backups and do not count toward ordinary retention. No archive is created if the month has no later successful backup.",
        "invalidRetention": "Retention days and counts must be non-negative integers. Non-permanent archives require at least 1 copy.",
        "monthEnd": "Month end",
        "preview": "Archive one backup on {dates} each month: {retention}.",
        "retainLatest": "keep the latest {count} archives in total",
        "retention": "Archive retention",
        "selectDates": "Select at least one archive date",
        "selectedDates": "{count} dates selected",
        "title": "Monthly archives"
      },
      "columns": {
        "parts": "Parts"
      },
      "imageStorage": {
        "bucket": "Bucket",
        "bucketInherited": "Leave empty to use the backup bucket",
        "description": "Enables the asynchronous image endpoints and offloads generated images to object storage, keeping only short links in Redis. Shares the S3 client with backups and takes effect on save — no restart needed.",
        "enabled": "Enable async image tasks",
        "prefix": "Key prefix",
        "presignExpiryHours": "Presigned link TTL (hours)",
        "publicBaseUrl": "Public base URL",
        "publicBaseUrlPlaceholder": "Leave empty to return presigned links",
        "reuseBackupS3": "Reuse the backup S3 configuration above (different bucket/prefix only)",
        "saved": "Async image object storage saved",
        "title": "Async image object storage"
      },
      "schedule": {
        "ordinaryHint": "Clean up the oldest ordinary backups when either limit is reached. Monthly archives are retained separately.",
        "ordinaryRetention": "Ordinary backup retention",
        "preview": "Retention preview",
        "previewBoth": "Keep up to {count} ordinary backups from the last {days} days.",
        "previewCount": "Keep the latest {count} ordinary backups, with no age limit.",
        "previewDays": "Keep ordinary backups from the last {days} days, with no count limit.",
        "previewUnlimited": "Ordinary backups are not automatically deleted."
      }
    },
    "channels": {
      "form": {
        "cacheWrite1hPrice": "Cache Write (1h)",
        "cacheWrite5mPrice": "Cache Write (5m)",
        "clearReasoningEffortMultipliers": "Clear multipliers",
        "inclusive": "(inclusive)",
        "maxTokens": "Max",
        "minTokens": "Min",
        "reasoningEffortMultiplierDefault": "Default: 1",
        "reasoningEffortMultiplierLabel": "{effort} reasoning effort multiplier",
        "reasoningEffortMultiplierPositive": "The {effort} reasoning effort multiplier must be a finite number greater than 0",
        "reasoningEffortLevelInvalid": "Unsupported reasoning effort: {effort}",
        "reasoningEffortMultipliers": "Custom Reasoning Effort Multipliers (optional)",
        "reasoningEffortMultipliersHint": "Apply the multiplier for the final forwarded reasoning effort to the entire request cost and quota usage. Unconfigured levels use 1×. These multiply with other billing multipliers."
      }
    },
    "groups": {
      "accountsUnit": "accounts",
      "codexModelsManifest": {
        "accounts": "Pinned accounts",
        "disabledHint": "Disabled: ordinary lists use local mappings or defaults; Codex uses a local catalog when configured, otherwise scheduler discovery.",
        "enable": "Fetch model lists with specific accounts",
        "enabledHint": "Accounts are limited to OpenAI accounts bound to this group, at most 10.",
        "fallback": "Fall back to the scheduler when all pinned accounts are unavailable",
        "fallbackHint": "Off: return 503 / the upstream error. On: fall back to the existing scheduler path.",
        "hint": "When enabled, ordinary model lists and Codex Model Manifest are discovered from the pinned accounts first, then merged and filtered using account mappings and the group model list. Rate-limited or overloaded pinned accounts are still used.",
        "searchEmpty": "No matching accounts",
        "searchPlaceholder": "Search accounts (OpenAI accounts in this group)",
        "selectAtLeastOne": "Select at least one account after enabling pinned accounts",
        "title": "Pinned Accounts for Model Lists"
      },
      "explicitPricing": {
        "description": "Optional per-group prices for web_search (per 1k calls) and Voice realtime / TTS / STT (USD). Leave empty if unused.",
        "pricePlaceholder": "optional",
        "searchPricePer1k": "Search price per 1k calls (USD)",
        "title": "Grok Search & Voice Pricing"
      },
      "form": {
        "addReasoningEffortMapping": "Add mapping",
        "addReasoningEffortPair": "Add request value",
        "maxReasoningEffort": "Max reasoning effort",
        "maxReasoningEffortHint": "Limits explicit Anthropic and OpenAI reasoning effort requests. For Composite groups, it applies to the resolved target platform. Omitted effort stays omitted. The ceiling takes precedence over reasoning effort mappings.",
        "maxReasoningEffortOverLimit": "Over-limit access control",
        "maxReasoningEffortOverLimitDeny": "Deny access",
        "maxReasoningEffortOverLimitDowngrade": "Automatically downgrade when over limit",
        "maxReasoningEffortOverLimitHint": "Applies after a ceiling is set. Downgrade rewrites values above the ceiling to the ceiling. Deny rejects the request.",
        "maxReasoningEffortUnlimited": "Unlimited (follow request)",
        "reasoningEffortFrom": "Request value",
        "reasoningEffortFromPlaceholder": "Select A",
        "reasoningEffortMappings": "Reasoning effort mappings",
        "reasoningEffortMappingsHint": "Type and model can both be left empty to match every model. One type and model can hold multiple request mappings, for example prefix gpt mapping both high and xhigh to medium. Choose Deny as the forwarded value to reject that request value. Exact matches beat affixes, and longer affixes beat shorter ones.",
        "reasoningEffortMatchExact": "Exact",
        "reasoningEffortMatchPrefix": "Prefix",
        "reasoningEffortMatchSuffix": "Suffix",
        "reasoningEffortMatchType": "Type",
        "reasoningEffortMatchTypePlaceholder": "All models",
        "reasoningEffortModel": "Model",
        "reasoningEffortModelPlaceholder": "Empty = all / gpt / gpt-5.4",
        "reasoningEffortTo": "Forwarded value",
        "reasoningEffortToDeny": "Deny",
        "reasoningEffortToPlaceholder": "Select B",
        "removeReasoningEffortMapping": "Remove mapping",
        "removeReasoningEffortPair": "Remove request value"
      },
      "i18nField": {
        "default": "Default",
        "placeholder": "{language} version; leave empty to show the default text"
      },
      "modelAllowlist": {
        "emptySelectionError": "The model allowlist is enabled; select or add at least one model entry"
      },
      "openaiFast": {
        "force": "Force Fast (priority)",
        "free": "Free Fast",
        "freeHint": "Fast requests in this group still use the priority tier, but customers are charged the equivalent Standard price.",
        "hint": "Forces service_tier=priority on OpenAI requests in this group. The global Fast/Flex policy can still filter or block it. New requests update immediately after saving; existing WebSocket sessions must reconnect.",
        "title": "OpenAI Fast mode"
      },
      "openaiLive": {
        "allow": "Allow Live access",
        "enableAnyway": "Enable anyway",
        "hint": "When enabled, API keys in this OpenAI group can create and control Live voice sessions. Disabled by default. The Sub2API server must run on Apple Silicon macOS with the official ChatGPT app installed; client platforms are unrestricted.",
        "title": "OpenAI Live",
        "unsupportedMessage": "This Sub2API server cannot generate the required Live attestation. Live will not work even if enabled. Continue anyway?",
        "unsupportedTitle": "Current server does not support Live"
      },
      "voicePricing": {
        "audioRealtimePerMin": "Realtime price per minute (USD)",
        "audioSttPerHour": "STT price per hour (USD)",
        "audioTtsPerMillionChars": "TTS price per million chars (USD)",
        "pricePlaceholder": "optional"
      }
    },
    "ops": {
      "errorDetail": {
        "backToList": "Back to List"
      },
      "outputTps": "Per-request output TPS",
      "outputTpsSamples": "Valid samples: {count}",
      "requestDetails": {
        "table": {
          "account": "Account"
        }
      },
      "systemLogs": {
        "retentionDaysCustom": "Custom days",
        "retentionDaysInvalid": "Keep operations logs for 1–3650 days; keep request logs for 1–3650 days or choose Forever.",
        "retentionDaysOption": "{days} days",
        "retentionForever": "Forever",
        "runtimeConfigLoadFailed": "Failed to load log configuration. Refresh and try again."
      },
      "tooltips": {
        "outputTps": "Percentiles of each valid usage record’s output tokens / total duration, including first-token wait, within the selected time, platform and group. Output may include reasoning tokens; they are not added again. P50 is the median; P5/P10 show slower requests. Higher is faster. Excludes images, Live and records without positive output or duration. Samples come from retained usage logs; — means no samples or temporarily unavailable statistics."
      }
    },
    "plugins": {
      "bridgeRejected": "Plugin UI message validation failed",
      "compatibility": "Version compatibility",
      "configTitle": "{name} configuration",
      "configure": "Configure",
      "confirmDisable": "Disable this plugin? New OAuth requests immediately return to the built-in Sub2API path.",
      "confirmUninstall": "Uninstall this plugin? It must be disabled first. Installed files and configuration will be removed.",
      "confirmUntested": "This plugin is compatible but has not declared the current Sub2API version as tested. Enable it anyway?",
      "currentVersion": "Current Sub2API",
      "description": "Install and manage isolated OAuth outbound transport plugins. API Key flows are unchanged.",
      "disable": "Disable",
      "disableSuccess": "Plugin disabled",
      "empty": "No plugins installed",
      "emptyHint": "Select a local .s2plugin package. Sub2API never downloads third-party plugins automatically.",
      "enable": "Enable",
      "enableSuccess": "Plugin enabled",
      "fileRequired": "Select a .s2plugin file",
      "healthy": "Healthy",
      "loadingUI": "Loading plugin configuration UI...",
      "menuNotice": "The Plugin Management switch in System Settings controls only sidebar visibility; it does not stop loaded or running plugins.",
      "noAccountCoupling": "The scope is platform and account type. Account records are not changed and no per-account toggle is required.",
      "onlyOpenAI": "Initial capability: OpenAI OAuth outbound transport only",
      "recommendedVersion": "Recommended version",
      "requiredVersion": "Required range",
      "rollout": "OAuth traffic percentage",
      "runtime": "Runtime",
      "runtimeNotice": "Plugin installation, enable/disable, and configuration are handled dynamically by the Sub2API host and normally do not require a host restart. Restart only when the host version or host configuration changes according to your deployment process.",
      "signature": "Package signature",
      "test": "Test",
      "testSuccess": "Plugin test passed",
      "title": "Plugin Management",
      "uiUnavailable": "Unable to load the plugin configuration UI",
      "unhealthy": "Not running",
      "uninstall": "Uninstall",
      "uninstallSuccess": "Plugin uninstalled",
      "upload": "Install plugin",
      "uploadHint": "Only .s2plugin packages are accepted; trusted publisher signatures are required by default.",
      "uploadSuccess": "Plugin installed and kept disabled"
    },
    "proxies": {
      "columns": {
        "hostLabel": "Host",
        "hostPlaceholder": "Enter host address",
        "nameLabel": "Name",
        "namePlaceholder": "Enter proxy name",
        "passwordLabel": "Password (Optional)",
        "passwordPlaceholder": "Enter password",
        "portLabel": "Port",
        "portPlaceholder": "Enter port",
        "priorityLabel": "Priority",
        "protocolLabel": "Protocol",
        "selectProtocol": "Select protocol",
        "statusLabel": "Status",
        "usernameLabel": "Username (Optional)",
        "usernamePlaceholder": "Enter username"
      },
      "deleteConfirmMessage": "Are you sure you want to delete proxy '{name}'?",
      "failedToSave": "Failed to save proxy",
      "filters": {
        "allProtocols": "All Protocols",
        "allStatuses": "All Status",
        "protocol": "Protocol",
        "status": "Status"
      },
      "noProxies": "No proxies yet",
      "noProxiesDescription": "Add a proxy server to improve API access stability.",
      "proxyCreatedSuccess": "Proxy created successfully",
      "proxyDeletedSuccess": "Proxy deleted successfully",
      "proxyUpdatedSuccess": "Proxy updated successfully",
      "saving": "Saving...",
      "testProxy": "Test Proxy",
      "testSuccess": "Proxy test passed",
      "testing": "Testing..."
    },
    "redeem": {
      "codeDeletedSuccess": "Redeem code deleted successfully",
      "codeDisabledSuccess": "Redeem code disabled",
      "codeEnabledSuccess": "Redeem code enabled",
      "codesGeneratedSuccess": "Redeem codes generated successfully, {count} total",
      "columns": {
        "createdAt": "Created At"
      },
      "copyCode": "Copy",
      "deleteConfirmMessage": "Are you sure you want to delete this redeem code?",
      "disableCode": "Disable",
      "enableCode": "Enable",
      "expiryDateRequired": "Please enter a valid expiry date and time",
      "failedToUpdate": "Failed to update redeem code",
      "filters": {
        "allStatuses": "All Status",
        "allTypes": "All Types",
        "search": "Search codes",
        "status": "Status",
        "type": "Type"
      },
      "form": {
        "balanceHint": "Balance amount (USD)",
        "concurrencyHint": "Concurrency increment",
        "countHint": "Number of redeem codes to generate",
        "countLabel": "Count",
        "countPlaceholder": "Enter count",
        "expiresLabel": "Expires At (Optional)",
        "prefixLabel": "Prefix (Optional)",
        "prefixPlaceholder": "e.g., GIFT",
        "selectType": "Select type",
        "typeLabel": "Type",
        "valueLabel": "Value",
        "valuePlaceholder": "Enter value"
      },
      "localTimeZoneHint": "Custom time is interpreted in your browser time zone ({timezone}).",
      "noCodes": "No redeem codes yet",
      "noCodesDescription": "Generate redeem codes to distribute balance or concurrency to users."
    },
    "settings": {
      "aliyunCaptcha": {
        "accessKeyId": "AccessKey ID",
        "accessKeyIdHint": "Alibaba Cloud AccessKey ID used for server-side verification; a captcha-only RAM user is recommended",
        "accessKeySecret": "AccessKey Secret",
        "accessKeySecretConfiguredHint": "Secret configured. Leave empty to keep the current value.",
        "accessKeySecretHint": "Server-side verification secret (keep this secret)",
        "prefix": "Prefix",
        "prefixHint": "Found in the instance information on the console overview page",
        "region": "Region",
        "regionCn": "Mainland China",
        "regionHint": "Determines the frontend script region and the server endpoint; must match your captcha instance region",
        "regionSgp": "Singapore",
        "sceneId": "Scene ID",
        "sceneIdHint": "Create a verification scene in the Alibaba Cloud Captcha console; the captcha type (invisible/slider/puzzle) is configured per scene there"
      },
      "apiKeyAcl": {
        "forwardedClientIpHeaders": "Custom client-IP headers",
        "forwardedClientIpHeadersHint": "Add CDN or proxy header names to check before the built-in headers.",
        "forwardedClientIpHeadersPlaceholder": "X-Client-IP",
        "forwardedClientIpHeadersRiskHint": "These raw headers can be spoofed when the origin is reachable directly. Restrict origin access before trusting them.",
        "forwardedClientIpHeaderInvalid": "Enter a valid HTTP header name.",
        "forwardedClientIpHeadersLimit": "At most {max} custom client-IP headers are allowed.",
        "removeForwardedClientIpHeader": "Remove {header}"
      },
      "captcha": {
        "description": "Bot protection for login and registration",
        "enable": "Enable CAPTCHA",
        "enableHint": "Require human verification on login, registration and related flows",
        "provider": "Provider",
        "providerAliyun": "Aliyun Captcha 2.0",
        "providerTencent": "Tencent Captcha",
        "providerTurnstile": "Cloudflare Turnstile",
        "title": "CAPTCHA"
      },
      "customMenu": {
        "hideOpenButton": "Hide the “Open in new tab” button"
      },
      "features": {
        "channelMonitor": {
          "hideThroughput": "Hide throughput rates from users (RPM / TPM)",
          "hideThroughputHint": "When on, the user Channel Monitor page and user APIs omit RPM and TPM so fleet volume cannot be reverse-estimated from rates × window. Admins still see full metrics. Error rates, latency, and cache rates remain visible.",
          "hideUserRanking": "Hide user ranking from users",
          "hideUserRankingHint": "When on, the user Channel Monitor V2 page hides the user ranking tab and the user API returns no ranking rows. Admins still see the ranking.",
          "mode": "Monitor mode",
          "modeHint": "Default is V1 (active probes). Switch to V2 only when you want passive aggregation; only one implementation can be active at a time.",
          "modeV1": "V1 active probes",
          "modeV1Hint": "Default: runs scheduled upstream health checks for configured channel monitors (probe traffic).",
          "modeV2": "V2 passive monitoring",
          "modeV2Hint": "Opt-in: aggregates health metrics from real gateway traffic without upstream probe traffic. V1 probes stop while V2 is active."
        }
      },
      "gatewayForwarding": {
        "claudeCodeClientVersion": "Claude Code client version",
        "claudeCodeClientVersionHint": "The client version this gateway declares upstream when impersonating the official Claude Code CLI. Leave empty to use the auto-synced latest official release; setting a value pins it and stops following auto-sync. The SUB2API_CLAUDE_CLI_VERSION environment variable or built-in version is used only when neither the manual nor synced value is valid.",
        "claudeCodeVersionAutoSync": "Auto-sync Claude Code version",
        "claudeCodeVersionAutoSyncHint": "Fetches the latest Claude Code client version from the official release channel every hour, so you never need to upgrade this service just to keep the version current. When disabled, fetching stops but the previously synced version remains available. The manual version above always takes priority.",
        "claudeCodeVersionSyncedValue": "Currently synced: {version}",
        "openaiCodexClientVersion": "Codex client version",
        "openaiCodexClientVersionHint": "The Codex client version this gateway declares upstream, shared by the User-Agent and the version header. Leave empty to use the auto-synced latest stable release; setting a value pins it and stops following auto-sync.",
        "openaiCodexClientVersionPlaceholder": "Leave empty to follow auto-sync",
        "openaiCodexVersionAutoSync": "Auto-sync Codex version",
        "openaiCodexVersionAutoSyncHint": "Fetches the latest stable client version from the official repository every 6 hours, so you never need to upgrade this service just to keep the version current. When disabled, only the version above or the built-in default is used.",
        "openaiCodexVersionSyncedValue": "Currently synced: {version}"
      },
      "ollamaCloudUsage": {
        "debounceHint": "Range: 1–60 minutes. Refresh after the latest model request has been quiet for this long.",
        "debounceMinutes": "Quiet period after last request (minutes)",
        "description": "Refresh official Ollama settings-page usage driven by model requests for individually opted-in accounts. Disabled by default. Idle accounts are not polled.",
        "enabled": "Enable global automatic refresh",
        "enabledHint": "Only accounts with a stored browser session and their own automatic refresh switch enabled are refreshed, and only after subsequent model requests. Manual refresh remains available.",
        "intervalHint": "Range: 15–1440 minutes. When continuous requests keep sliding the debounce, force a refresh after this wait.",
        "intervalMinutes": "Max wait while requests continue (minutes)",
        "saveFailed": "Failed to save Ollama Cloud usage refresh settings",
        "saved": "Ollama Cloud usage refresh settings saved",
        "title": "Ollama Cloud Usage Refresh"
      },
      "openaiExperimentalScheduler": {
        "oauthRateInvalid": "The OAuth scheduling reference rate must be a non-negative number, or blank to use account rates."
      },
      "openaiFastPolicy": {
        "summaryOtherModels": "Other models",
        "tierMissing": "Omitted tier",
        "tierUltrafast": "ultrafast"
      },
      "opencodeGoUsage": {
        "debounceHint": "Range: 1–60 minutes, and must be less than the refresh interval. Refresh after the latest model request has been quiet for this long.",
        "debounceMinutes": "Quiet period after last request (minutes)",
        "description": "Refresh usage windows reported by the upstream OpenCode Go account for individually opted-in accounts. Disabled by default.",
        "enabled": "Enable global automatic refresh",
        "enabledHint": "Only accounts with their own automatic refresh switch enabled are refreshed. Manual refresh remains available.",
        "intervalHint": "Range: 5–1440 minutes. When continuous requests keep sliding the debounce, force a refresh after this wait.",
        "intervalMinutes": "Max wait while requests continue (minutes)",
        "saveFailed": "Failed to save OpenCode Go usage refresh settings",
        "saved": "OpenCode Go usage refresh settings saved",
        "title": "OpenCode Go Usage Refresh"
      },
      "panelRateLimit": {
        "description": "Throttle panel API requests to keep high-frequency polling (usage stats, dashboard queries) from overwhelming the database",
        "enabled": "Enable panel rate limiting",
        "enabledHint": "Limits authenticated panel endpoints per account. Requests over the threshold get HTTP 429 and recover automatically when the window resets.",
        "exemptAdmin": "Exempt administrators",
        "exemptAdminHint": "When enabled, admin accounts bypass panel rate limits so bulk operations are never throttled.",
        "heavyRpm": "Heavy queries per account",
        "heavyRpmHint": "Usage/dashboard aggregation queries allowed per account per minute (these are the most expensive for the database). 0 = unlimited.",
        "perMinute": "req/min",
        "proxySafeNote": "Authenticated endpoints are counted per user account, independent of the source IP — reverse proxies and shared NAT egress are never falsely blocked. Public endpoints are counted per real client IP, and loopback/private addresses (internal proxy hops) are skipped automatically.",
        "publicIpRpm": "Public endpoints per IP",
        "publicIpRpmHint": "Requests per minute allowed per real client IP for unauthenticated endpoints (e.g. public site settings). 0 = unlimited.",
        "title": "Panel API Rate Limiting",
        "saved": "Panel rate limit settings saved",
        "saveFailed": "Failed to save panel rate limit settings",
        "userRpm": "Requests per account",
        "userRpmHint": "Total panel API requests allowed per account per minute. Normal UI usage stays far below this. 0 = unlimited."
      },
      "payment": {
        "alipayMobilePrecreateDeepLink": "Mobile Alipay Precreate Handoff",
        "alipayMobilePrecreateDeepLinkHint": "Use official Alipay precreate on mobile, open the Alipay app, and show the dynamic QR only if handoff fails. This takes priority over Force Alipay QR Code",
        "rechargeBonus": {
          "addTier": "Add Tier",
          "empty": "No promotion tiers configured; top-ups are credited at face value.",
          "hint": "Balance top-ups match a tier by the amount the user enters (highest threshold not above it). Leave empty for no promotion. Subscriptions are not affected.",
          "incompleteRow": "Fill in both amount and percent for this tier to take effect",
          "label": "Recharge Promotion Tiers",
          "minAmountLabel": "Amount ≥",
          "modeBonus": "Bonus",
          "modeBonusHint": "Bonus: the payment stays the same and the credited base (amount × multiplier) gets the percentage added on top.",
          "modeDiscount": "Discount OFF",
          "modeDiscountHint": "Discount: the credit stays the same (amount × multiplier) and the payment is reduced by the percentage; must be below 100.",
          "modeLabel": "Promotion type",
          "noticeHint": "Markdown supported. Shown at the top of the amount picker on the recharge page; leave empty to hide.",
          "noticeLabel": "Recharge Bonus Notice",
          "noticePlaceholder": "e.g. 🎁 Limited offer: get 20% extra on top-ups of $100+, 30% on $500+ …",
          "percentLabel": "Bonus",
          "percentLabelDiscount": "Discount",
          "previewOpen": "≥ {from}: {percent}% bonus",
          "previewOpenDiscount": "≥ {from}: {percent}% OFF",
          "previewOpenNone": "≥ {from}: no bonus",
          "previewRange": "{from} ~ {to}: {percent}% bonus",
          "previewRangeDiscount": "{from} ~ {to}: {percent}% OFF",
          "previewRangeNone": "{from} ~ {to}: no bonus",
          "previewTitle": "Range preview",
          "removeTier": "Remove tier"
        }
      },
      "registration": {
        "emailDomainQuota": "Non-allowlist Domain Quota",
        "emailDomainQuotaHint": "When enabled and the allowlist is not empty, every other registrable domain can register one account. When disabled, non-allowlist domains are rejected. Has no effect while the allowlist is empty"
      },
      "site": {
        "compactHome": "Compact Home Page",
        "compactHomeHint": "Show a restrained site identity page when no custom home page content is set."
      },
      "tencentCaptcha": {
        "aidEncryptedHint": "aidEncrypted is not supported yet. Keep CaptchaAppId mandatory verification disabled in the Captcha console.",
        "appCredentialsHint": "Get CaptchaAppId and AppSecretKey from Verification Management in the Captcha console.",
        "appCredentialsTitle": "Captcha application credentials",
        "appId": "CaptchaAppId",
        "appSecretKey": "AppSecretKey",
        "camPermissionHint": "Create a CAM sub-user with QcloudCaptchaFullAccess instead of using permanent root-account credentials.",
        "cloudCredentialsHint": "SecretId and SecretKey authorize server-side DescribeCaptchaResult requests.",
        "cloudCredentialsTitle": "Cloud API credentials",
        "cloudSecretId": "Tencent Cloud SecretId",
        "cloudSecretKey": "Tencent Cloud SecretKey",
        "configured": "Configured. Leave empty to keep it.",
        "createCloudKeys": "Create SecretId / SecretKey",
        "keepExisting": "Leave empty to keep current value",
        "openCaptchaConsole": "Open Captcha console",
        "openWebDocs": "View Web integration guide",
        "region": "Service site",
        "regionCn": "Chinese mainland",
        "regionHint": "Selects the SDK script and the server-side verification endpoint. It must match the site that issued your CaptchaAppId; international apps are created in the tencentcloud.com console.",
        "regionIntl": "International",
        "required": "Required before enabling."
      },
      "upstreamBillingProbe": {
        "description": "Periodically retrieve rates declared by upstream Sub2API sites. Account rates change only when the separate sync switch is enabled.",
        "enabled": "Enable global auto detection",
        "enabledHint": "When enabled, scheduled detection runs only for accounts that also enable automatic detection. Disabling stops all scheduled detection; manual detection remains available.",
        "intervalHint": "Range: 5–1440 minutes. A successful result remains valid for two detection intervals.",
        "intervalMinutes": "Detection interval (minutes)",
        "saveFailed": "Failed to save upstream rate auto detection settings",
        "saved": "Upstream rate auto detection settings saved",
        "title": "Upstream Rate Auto Detection"
      }
    },
    "subscriptions": {
      "batchAssign": {
        "enable": "Assign to multiple users",
        "hint": "Search and add up to 100 users to assign the same group and validity period.",
        "removeUser": "Remove {email}",
        "result": "Assignment complete: {success} succeeded, {failed} failed",
        "retryHint": "Successful users have been removed. Resolve any errors and submit the remaining users again.",
        "selected": "{count} users added"
      },
      "bulk": {
        "clearSelection": "Clear Selection",
        "confirm": "Confirm Action",
        "confirmTargets": "This action will process the following {count} subscriptions",
        "extendHint": "Enter a positive whole number to extend or a negative one to shorten, up to 36500 days. Expired subscriptions are extended from now and cannot be shortened. The new expiration must be in the future.",
        "groupFallback": "Group #{id}",
        "invalidDays": "Enter a nonzero whole number of days between -36500 and 36500",
        "itemFailed": "Action failed",
        "requestFailed": "The bulk request failed. Please retry.",
        "resetHint": "Usage in the selected windows will be zeroed and restarted from today.",
        "resetWindows": "Select quota windows to reset",
        "result": "Completed: {success} succeeded, {failed} failed",
        "retry": "Retry Original Action",
        "retryHint": "The outcome is not yet confirmed. Retrying continues the original operation to avoid duplicate changes. You can also close this dialog, then select the same subscriptions and settings to retry.",
        "selectSubscription": "Select subscription #{id}",
        "selectWindow": "Select at least one quota window",
        "selected": "{count} subscriptions selected",
        "selectionHint": "Select up to 100 subscriptions on this page. Changing pages or filters clears the selection. Each action only processes subscriptions with an applicable status.",
        "selectionLimit": "You can process up to 100 subscriptions at a time",
        "selectionRequired": "Select at least one subscription"
      },
      "failedToRestore": "Failed to restore subscription",
      "hoursMinutesRemaining": "{hours}h {minutes}m remaining",
      "minutesRemaining": "{minutes}m remaining",
      "restore": "Restore",
      "restoreConfirm": "Restore the subscription for '{user}'? If the original subscription has expired, it will be restored as expired.",
      "restoreSubscription": "Restore Subscription",
      "subscriptionRestored": "Subscription restored successfully"
    },
    "usage": {
      "upstreamRequestId": "Upstream ID",
      "upstreamRequestIdCopied": "Upstream ID copied"
    },
    "users": {
      "bulkDelete": {
        "confirm": "Delete the {count} selected users? This action cannot be undone. Administrator accounts cannot be deleted.",
        "failed": "Failed to delete {count} users. They remain selected for retry.",
        "success": "Deleted {count} users",
        "title": "Delete selected users"
      },
      "concurrencyNonNegative": "Concurrency cannot be negative; 0 = unlimited",
      "form": {
        "concurrencyHint": "Max concurrent requests for this user; 0 = unlimited.",
        "concurrencyPlaceholder": "0 = unlimited"
      },
      "publicGroupsRestricted": "Public Groups (Restricted)",
      "restrictPublicGroups": "Restrict accessible public groups",
      "restrictPublicGroupsHint": "When on, this user may only use the public groups checked below. When off, every public group stays available."
    }
  },
  "auth": {
    "captchaClickToVerify": "Click to complete verification",
    "captchaLoading": "Loading verification…",
    "captchaVerified": "Verification completed",
    "captchaVerifying": "Verifying…",
    "emailDomainRegistrationLimit": "This email domain cannot register another account. Please use a mainstream email, or contact support to add your enterprise domain to the allowlist.",
    "passkeyCancelled": "Passkey sign-in was cancelled.",
    "passkeyFailed": "Passkey sign-in failed. Please try again."
  },
  "channelMonitorV2": {
    "bootstrap": {
      "description": "On first enable, passive aggregation silently fills the 90m, 24h, 7d, and 30d windows in the background. All ranges become complete once this finishes.",
      "progress": "{percent}% complete",
      "title": "Building historical monitor data",
      "working": "Aggregating in the background…"
    },
    "bucket": {
      "days": "{count}-day buckets",
      "hours": "{count}-hour buckets",
      "minutes": "{count}-minute buckets"
    },
    "chart": {
      "cacheDataset": "Cache rate trend %",
      "cacheLegend": "Cache rate (left axis %)",
      "description": "Smoothed trend: error rate · first token P50 · cache rate",
      "emptyTitle": "No trend data for the selected window",
      "errorDataset": "Error rate trend %",
      "errorLegend": "Error rate (left axis %)",
      "percentAxis": "Rate %",
      "resetZoom": "Reset zoom",
      "title": "Availability trend",
      "ttftDataset": "First token trend P50 (ms)",
      "ttftLegend": "First token P50 (right axis)"
    },
    "clearFilters": "Reset",
    "currentUser": "Current user",
    "detailLoadFailed": "Failed to load channel monitor details",
    "empty": {
      "description": "Try changing the time range or filters",
      "title": "No data to display"
    },
    "errorDetail": {
      "empty": "Category rates only (sample messages are admin-only)",
      "http": "HTTP {code}",
      "noMessage": "No error message",
      "upstream": "Upstream {code}"
    },
    "filters": {
      "allGroups": "All",
      "allModels": "All",
      "allPlatforms": "All",
      "empty": "No options",
      "group": "Group",
      "labelValue": "{label}: {value}",
      "model": "Model",
      "platform": "Platform",
      "selectedCount": "{count}"
    },
    "groupBy": {
      "label": "Group by",
      "platform": "Platform",
      "platformGroup": "Platform / Group",
      "platformGroupModel": "Platform / Group / Model",
      "platformModel": "Platform / Model"
    },
    "healthMode": {
      "cache": "Cache rate",
      "label": "Health display",
      "overall": "Overall",
      "success": "Error rate",
      "ttft": "First token"
    },
    "ignored": "Ignored",
    "loadFailed": "Failed to load channel monitor",
    "matrix": {
      "bad": "Bad",
      "criticalLegend": "Critical (<50)",
      "description": "Each row is a channel dimension and each block is an aggregate interval; hover for details",
      "dimension": "Channel dimension",
      "emptyTitle": "No matrix data for the selected window",
      "good": "Good",
      "healthyLegend": "Healthy (≥80)",
      "legendAria": "Health score legend",
      "noTraffic": "No traffic in this interval",
      "noTrafficAt": "{time} · no traffic",
      "resetZoom": "Reset zoom",
      "scoreLine": "Health score {score}",
      "title": "Availability trend",
      "unknownLegend": "No traffic / insufficient samples",
      "warningLegend": "Watch (50–79)",
      "wheelZoomX": "Scroll over blocks to zoom in (narrower range, wider blocks)"
    },
    "metrics": {
      "cacheDetail": "Read cache share",
      "cacheRate": "Cache rate",
      "cacheRateValue": "Cache rate {value}",
      "durationValue": "Duration {value}",
      "errorRateValue": "Error rate {value}",
      "rpm": "RPM",
      "rpmDetail": "Requests per minute",
      "rpmValue": "RPM {value}",
      "successRate": "Success rate",
      "successRateValue": "Success rate {value}",
      "tps": "Tokens/s",
      "tpsDetail": "Derived as TPM ÷ 60",
      "tpsValue": "Tokens/s {value}",
      "ttft": "First token",
      "ttftP50": "First token P50",
      "ttftValue": "First token {value}"
    },
    "otherModels": "Other models",
    "partialCoverage": "Partial historical coverage",
    "ranges": {
      "24h": "24h",
      "30d": "30d",
      "7d": "7d",
      "90m": "90m"
    },
    "rank": {
      "bronze": "Rank 3 bronze",
      "gold": "Rank 1 gold",
      "place": "Rank {n}",
      "silver": "Rank 2 silver",
      "unranked": "Unranked"
    },
    "settings": {
      "addPlatform": "Add platform",
      "badgeAllModels": "All models",
      "badgeOther": "+ Other",
      "description": "Configure passive usage aggregation dimensions (platform / model / group) and refresh cadence. Health colors and details on the user /monitor page show rates, RPM, and TPM — not absolute request volume.",
      "enableHint": "Applies when system mode is V2. Turning this off only stops this config’s aggregation; the system mode switch remains under Feature switches.",
      "enableTitle": "Enable V2 aggregation",
      "errorsHint": "Checked “ignore” categories are excluded from error rate and health score, but still appear greyed in the error breakdown. Unmatched errors roll into “Other”.",
      "errorsTitle": "Error categories and ignores",
      "fields": {
        "criticalCache": "Cache rate critical %",
        "criticalError": "Error rate critical %",
        "criticalTtft": "TTFT critical ms",
        "minimumSample": "Minimum samples",
        "targetTtft": "TTFT target ms",
        "warningCache": "Cache rate watch %",
        "warningError": "Error rate watch %",
        "warningTtft": "TTFT watch ms"
      },
      "groupsAll": "All groups",
      "groupsEmpty": "No groups available",
      "groupsSelected": "{count} groups selected",
      "groupsTitle": "Monitored groups",
      "healthHint": "Controls user-facing color bands and overall score. Defaults are tolerant so small error rates or low cache do not immediately show as unhealthy.",
      "healthTitle": "Health thresholds",
      "ignoredSummary": "Ignored {ignored} categories · counted in error rate {counted} categories",
      "loadFailed": "Failed to load V2 config",
      "loading": "Loading…",
      "modeBanner": "System mode is currently {mode}. V2 minute aggregation will not run; this config can be prepared now and takes effect after switching to {modeV2}. Change mode under System Settings → Feature switches.",
      "modeClosed": "Channel monitor disabled",
      "modeV1": "V1 active probes",
      "modeV2": "V2 passive monitoring",
      "modelsPlaceholder": "Empty = all real models; or list popular models (rest → Other)",
      "namedModelsCount": "Showing {count} named model dimensions; unlisted models fold into per-platform “Other”.",
      "namedModelsEmpty": "Platform model lists are empty: every real model name will be shown (not folded into “Other”).",
      "platformsHint": "Leave empty = show all real model names; when filled, only listed models get their own rows and the rest roll into “Other”",
      "platformsTitle": "Platforms and models",
      "refreshAria": "Aggregation interval",
      "refreshHint": "Affects matrix time granularity and refresh cadence",
      "refreshTitle": "Aggregation interval",
      "save": "Save",
      "saveFailed": "Failed to save V2 config",
      "saveSuccess": "V2 monitor config saved",
      "selectPlatform": "Select a platform",
      "title": "V2 data monitor config",
      "userContract": {
        "health": "Health color weights: error rate 60% + first-token P50 20% + cache rate 20% (thresholds configurable above)",
        "latency": "Latency shows AVG · P50 · P90; absolute request / error counts are not shown",
        "models": "Empty model lists show real names and never dump everything into “Other”",
        "trend": "Trend can switch between pulse matrix and line chart (error · cache · first token)"
      },
      "userContractTitle": "User-facing display contract"
    },
    "summaryAria": "Selected range summary",
    "table": {
      "platformModel": "Platform / Model",
      "rank": "Rank",
      "user": "User"
    },
    "tabs": {
      "aria": "Detail dimension",
      "errors": "Error reasons",
      "models": "Models",
      "users": "User ranking"
    },
    "timeRange": "Time range",
    "title": "Channel Monitor",
    "trendView": {
      "label": "Trend view",
      "line": "Line chart",
      "pulse": "Pulse matrix"
    },
    "updatedTo": "Updated to {time}",
    "updating": "Updating data"
  },
  "common": {
    "fileReadFailed": "Failed to read file",
    "fileTooLargeKb": "File too large ({size} KB), max {max} KB",
    "noFileSelected": "No file selected",
    "selectImageFile": "Please select an image file",
    "selectedFile": "Selected: {name}",
    "upload": "Upload"
  },
  "customPage": {
    "copiedCode": "Copied",
    "copyCode": "Copy",
    "copyCodeFailed": "Failed",
    "tableOfContents": "Contents"
  },
  "keyUsage": {
    "billingType": "Billing Type"
  },
  "keys": {
    "bulkEdit": {
      "apply": "Apply to {count} keys",
      "clearSelection": "Clear selection",
      "failureHint": "These keys could not be updated. Adjust the settings and retry. Only failed keys will be retried.",
      "hint": "Check the fields to update. Unchecked fields keep their current values.",
      "invalidExpiration": "Choose a valid expiration date or select Never expires.",
      "invalidLimit": "Enter a valid amount greater than or equal to 0.",
      "ipHint": "One IP or CIDR per line. Leave empty to clear this list on the selected keys.",
      "limitHint": "Enter 0 for no limit. Existing usage is preserved.",
      "partialFailure": "Updated {success} keys; {failed} failed",
      "selectKey": "Select key {name}",
      "selectedCount": "{count} keys selected",
      "success": "Updated {count} keys",
      "title": "Bulk Edit"
    },
    "providerLabel": "Provider",
    "providers": {
      "anthropic": "Anthropic",
      "openai": "OpenAI",
      "domestic": "Chinese AI",
      "other": "Other"
    },
    "providerHints": {
      "anthropic": "Choose an available Anthropic / Claude group",
      "openai": "Choose an available OpenAI / GPT group",
      "domestic": "Includes DeepSeek, Kimi, Zhipu GLM and MiniMax",
      "other": "Includes Gemini, Grok, Antigravity, OpenCode and mixed groups"
    },
    "useKeyModal": {
      "claudeSettingsHint": "User-level persistent configuration. Do not commit this file containing your API key to a project repository.",
      "cliTabs": {
        "systemOne": "System One"
      },
      "codexModelCatalog": {
        "description": "Codex loads and refreshes the remote catalog using your configured authentication. For local file mode, fetch the catalog below and save it at the configured path.",
        "download": "Download catalog",
        "errorDescription": "The catalog could not be fetched with this API key.",
        "fetch": "Fetch catalog",
        "local": "Local file (older clients)",
        "mode": "Catalog source",
        "modelsCount": "{count} models ready to download",
        "oversized": "The complete catalog exceeds the 1 MiB remote limit. Local file mode is selected; download it to the configured path.",
        "remote": "Remote catalog (Codex 0.156.0+)",
        "retry": "Retry",
        "title": "Codex model catalog"
      },
      "routedCodex": {
        "description": "Configure Codex with the complete model catalog for the current routed group.",
        "note": "Export SUB2API_API_KEY before starting Codex. The downloaded catalog contains model metadata only, not your API key."
      }
    }
  },
  "modelPlaza": {
    "anonymousHint": "Sign in to see your exclusive groups and personal rates",
    "badges": {
      "exclusive": "Exclusive",
      "subscription": "Subscription"
    },
    "description": "Browse available models and pricing by group",
    "detail": {
      "longContextDisabledNote": "Long-context tier pricing is disabled for this group: requests above the threshold are billed at the base tier; official tiers are for reference only",
      "noModels": "No models configured for this group",
      "peakNote": "Peak hours {window}: billing rate ×{multiplier}"
    },
    "empty": "No groups to display",
    "filters": {
      "all": "All",
      "groupLabel": "Group",
      "modelLabel": "Model",
      "platformLabel": "Platform",
      "rateLabel": "Rate",
      "searchPlaceholder": "Search models"
    },
    "loadFailed": "Failed to load model plaza",
    "nav": {
      "backToDashboard": "Back to Console",
      "login": "Sign In"
    },
    "noSearchResult": "No matching models",
    "table": {
      "cache": "Cache",
      "cacheRead": "Read",
      "cacheReadShort": "R",
      "cacheWrite": "Write",
      "cacheWriteShort": "W",
      "input": "Input",
      "marginalBadge": "excess-only tiers",
      "model": "Model",
      "officialPrice": "Official Price",
      "output": "Output",
      "paidPrice": "Your Price (Discounted)",
      "perImage": "Per image",
      "perRequest": "Per request",
      "perUnitImage": "/ image",
      "perUnitRequest": "/ request",
      "rate": "Rate",
      "reasoningMultiplierBadge": "{effort} ×{multiplier}",
      "reasoningMultiplierHint": "When the forwarded reasoning effort is {effort}, billing and quota usage for the request are multiplied by {multiplier}. Unconfigured levels use 1×",
      "tierHint": "The whole request is billed at the tier matching its total context (input + cache write + cache read)",
      "tierHintMarginal": "Only the portion above the threshold is billed at this tier; output is unaffected",
      "timePricingRateHint": "Effective rate {rate} × period multiplier {multiplier}",
      "timePricingRowHintPeak": "; prices in this row exclude the peak-hour rate — where this period overlaps the peak hours {window}, the overlapping portion is additionally multiplied by ×{multiplier}",
      "timePricingWeekdays": "Weekdays",
      "unitPerMillion": "$ / 1M tokens"
    },
    "title": "Model Plaza"
  },
  "nav": {
    "modelPlaza": "Model Plaza",
    "plugins": "Plugins",
    "recharge": "Recharge",
    "subscribe": "Subscription"
  },
  "payment": {
    "admin": {
      "queryRefundStatus": "Query refund status",
      "refundPending": "Refund pending gateway confirmation",
      "subscriptionCnyPayPreview": "CNY channel charge preview: {amount}",
      "subscriptionCnyPayPreviewWithFee": "({feeRate}% fee included: {total})"
    },
    "billingUnavailable": "Neither top-up nor subscriptions are currently available. Please contact the administrator.",
    "orders": {
      "bonusAmount": "Bonus"
    },
    "planCard": {
      "peakRate": "Peak Rate"
    },
    "qr": {
      "alipayContinueInApp": "Complete payment in Alipay",
      "alipayFallbackHint": "Try opening Alipay again, or save the QR code and scan it from your Alipay photo album",
      "alipayFallbackTitle": "Alipay did not open",
      "alipayOpening": "Opening Alipay",
      "alipaySaveAndScanHint": "Save the QR code, open Alipay Scan, then select it from your photo album",
      "alipayWaitingHint": "The server will confirm the payment and update this page automatically",
      "reopenAlipay": "Open Alipay Again",
      "saveQRCode": "Save QR Code"
    },
    "rechargeBonus": {
      "creditedShort": "Get {amount}",
      "giftTag": "Bonus {amount}",
      "payShort": "Pay {amount}"
    },
    "status": {
      "refund_pending": "Refund Pending"
    }
  },
  "profile": {
    "passkey": {
      "add": "Add passkey",
      "addFailed": "Failed to add passkey.",
      "added": "Passkey added.",
      "continue": "Create passkey",
      "createdAt": "Created {date}",
      "deleteConfirm": "Delete “{name}”? You will no longer be able to sign in with it.",
      "deleteFailed": "Failed to delete passkey.",
      "deleteTitle": "Delete passkey",
      "deleted": "Passkey deleted.",
      "description": "Use Face ID, Touch ID, Windows Hello, or a security key to sign in without a password.",
      "empty": "No passkeys are registered yet.",
      "featureDisabled": "Passkeys have not been configured by the administrator.",
      "lastUsed": "Last used {date}",
      "loadFailed": "Failed to load passkeys.",
      "name": "Passkey name",
      "namePlaceholder": "For example, MacBook Touch ID",
      "passwordPlaceholder": "Enter your current password to confirm",
      "renameFailed": "Failed to rename passkey.",
      "renamePrompt": "Enter a new name for this passkey",
      "renamed": "Passkey renamed.",
      "synced": "Synced",
      "title": "Passkeys",
      "unsupported": "This browser or device does not support passkeys."
    }
  },
  "purchase": {
    "rechargeDescription": "Recharge balance via the embedded page",
    "subscriptionDescription": "Purchase subscription via the embedded page"
  },
  "redeem": {
    "historyLoadFailed": "Failed to load activity. Please try again.",
    "userRefreshFailed": "Redeemed successfully, but failed to refresh account information."
  },
  "setup": {
    "redis": {
      "username": "Username (optional)",
      "usernamePlaceholder": "Leave empty for default user"
    }
  },
  "usage": {
    "live": "Live",
    "modelMismatch": "Different model",
    "modelVariant": "Possible version variant",
    "nativeCompactionV2": "Compaction",
    "requestedReasoningEffort": "Requested reasoning effort",
    "sentUpstreamModel": "Sent upstream",
    "upstreamModelMismatch": "Response model mismatch",
    "upstreamResponseModel": "Upstream response"
  },
  "version": {
    "copied": "Copied",
    "copyCommand": "Copy",
    "deployDocker": "Docker",
    "deployScript": "Script",
    "dockerEditCompose": "Edit the image tag in docker-compose.yml",
    "dockerRecreate": "Recreate the container",
    "loadVersionsFailed": "Failed to load versions",
    "manualRollbackCommand": "Manual rollback",
    "noRollbackVersions": "No versions available for rollback",
    "rollback": "Version Rollback",
    "rollbackComplete": "Rollback Complete",
    "rollbackConfirm": "Roll back to {version}",
    "rollbackFailed": "Rollback Failed",
    "rollbackSelectVersion": "Select a version to roll back to (last 3 versions)",
    "rollbackSourceHint": "Online rollback is not available for source builds",
    "rollbackWarning": "Rollback downloads the selected version and replaces the current binary. A service restart is required afterwards.",
    "rollingBack": "Rolling back..."
  }
}
