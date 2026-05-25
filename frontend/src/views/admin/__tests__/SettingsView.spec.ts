import { beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent, h } from "vue";
import { flushPromises, mount } from "@vue/test-utils";

import SettingsView from "../SettingsView.vue";

const {
  getSettings,
  updateSettings,
  getWebSearchEmulationConfig,
  updateWebSearchEmulationConfig,
  getAdminApiKey,
  getOverloadCooldownSettings,
  getRateLimit429CooldownSettings,
  updateRateLimit429CooldownSettings,
  getStreamTimeoutSettings,
  getRectifierSettings,
  getBetaPolicySettings,
  getModelMarket,
  updateModelMarket,
  getGroups,
  listProxies,
  getProviders,
  updateProvider,
  createProvider,
  deleteProvider,
  fetchPublicSettings,
  adminSettingsFetch,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  getSettings: vi.fn(),
  updateSettings: vi.fn(),
  getWebSearchEmulationConfig: vi.fn(),
  updateWebSearchEmulationConfig: vi.fn(),
  getAdminApiKey: vi.fn(),
  getOverloadCooldownSettings: vi.fn(),
  getRateLimit429CooldownSettings: vi.fn(),
  updateRateLimit429CooldownSettings: vi.fn(),
  getStreamTimeoutSettings: vi.fn(),
  getRectifierSettings: vi.fn(),
  getBetaPolicySettings: vi.fn(),
  getModelMarket: vi.fn(),
  updateModelMarket: vi.fn(),
  getGroups: vi.fn(),
  listProxies: vi.fn(),
  getProviders: vi.fn(),
  updateProvider: vi.fn(),
  createProvider: vi.fn(),
  deleteProvider: vi.fn(),
  fetchPublicSettings: vi.fn(),
  adminSettingsFetch: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}));

const localeRef = vi.hoisted(() => ({ value: "zh-CN" }));

vi.mock("@/i18n", () => ({
  i18n: {
    global: {
      locale: localeRef,
      t: (key: string) => key,
    },
  },
  getLocale: () => localeRef.value,
  setLocale: vi.fn(),
}));

vi.mock("@/api", () => ({
  adminAPI: {
    settings: {
      getSettings,
      updateSettings,
      getWebSearchEmulationConfig,
      updateWebSearchEmulationConfig,
      getAdminApiKey,
      getOverloadCooldownSettings,
      getRateLimit429CooldownSettings,
      updateRateLimit429CooldownSettings,
      getStreamTimeoutSettings,
      getRectifierSettings,
      getBetaPolicySettings,
      getModelMarket,
      updateModelMarket,
    },
    groups: {
      getAll: getGroups,
    },
    proxies: {
      list: listProxies,
    },
    payment: {
      getProviders,
      updateProvider,
      createProvider,
      deleteProvider,
    },
  },
}));

vi.mock("@/api/admin/settings", () => {
  const authSourceTypes = ["email", "linuxdo", "oidc", "wechat", "github", "google"] as const;
  const FRONTEND_LOCALE_OPTIONS = [
    { value: "en", labelZh: "English", labelEn: "English" },
    { value: "zh", labelZh: "简体中文", labelEn: "Simplified Chinese" },
    { value: "zh-Hant", labelZh: "繁體中文", labelEn: "Traditional Chinese" },
  ];
  const normalizeDefaultSubscriptionSettings = (
    subscriptions: Array<{ group_id: number; validity_days: number }> | null | undefined,
  ) =>
    Array.isArray(subscriptions)
      ? subscriptions
          .filter((item) => item.group_id > 0 && item.validity_days > 0)
          .map((item) => ({
            group_id: Math.floor(item.group_id),
            validity_days: Math.min(36500, Math.max(1, Math.floor(item.validity_days))),
          }))
      : [];
  const normalizeWeChatConnectMode = (source: unknown) => {
    const normalized = String(source || "").trim().toLowerCase();
    return normalized === "mp" || normalized === "mobile" ? normalized : "open";
  };
  const settingsAPI = {
    getSettings,
    updateSettings,
    testSmtpConnection: vi.fn(),
    sendTestEmail: vi.fn(),
    getAdminApiKey,
    regenerateAdminApiKey: vi.fn(),
    deleteAdminApiKey: vi.fn(),
    getModelMarket,
    updateModelMarket,
    getOverloadCooldownSettings,
    updateOverloadCooldownSettings: vi.fn(),
    getStreamTimeoutSettings,
    updateStreamTimeoutSettings: vi.fn(),
    getRectifierSettings,
    updateRectifierSettings: vi.fn(),
    getBetaPolicySettings,
    updateBetaPolicySettings: vi.fn(),
    getWebSearchEmulationConfig,
    updateWebSearchEmulationConfig,
    testWebSearchEmulation: vi.fn(),
    resetWebSearchUsage: vi.fn(),
  };

  return {
    default: settingsAPI,
    settingsAPI,
    FRONTEND_LOCALE_OPTIONS,
    normalizeDefaultSubscriptionSettings,
    buildAuthSourceDefaultsState: (settings: Record<string, unknown>) =>
      authSourceTypes.reduce(
        (acc, source) => {
          acc[source] = {
            balance: Number(settings[`auth_source_default_${source}_balance`] ?? 0),
            concurrency: Math.max(1, Number(settings[`auth_source_default_${source}_concurrency`] ?? 5)),
            subscriptions: normalizeDefaultSubscriptionSettings(
              settings[`auth_source_default_${source}_subscriptions`] as Array<{
                group_id: number;
                validity_days: number;
              }>,
            ),
            grant_on_signup: settings[`auth_source_default_${source}_grant_on_signup`] === true,
            grant_on_first_bind: settings[`auth_source_default_${source}_grant_on_first_bind`] === true,
          };
          return acc;
        },
        {} as Record<string, Record<string, unknown>>,
      ),
    appendAuthSourceDefaultsToUpdateRequest: (
      payload: Record<string, unknown>,
      authSourceDefaults: Record<string, Record<string, unknown>>,
    ) => {
      for (const source of authSourceTypes) {
        const current = authSourceDefaults[source] || {};
        payload[`auth_source_default_${source}_balance`] = Number(current.balance) || 0;
        payload[`auth_source_default_${source}_concurrency`] = Math.max(
          1,
          Math.floor(Number(current.concurrency) || 5),
        );
        payload[`auth_source_default_${source}_subscriptions`] = normalizeDefaultSubscriptionSettings(
          current.subscriptions as Array<{ group_id: number; validity_days: number }>,
        );
        payload[`auth_source_default_${source}_grant_on_signup`] = current.grant_on_signup === true;
        payload[`auth_source_default_${source}_grant_on_first_bind`] = current.grant_on_first_bind === true;
      }
      return payload;
    },
    defaultWeChatConnectScopesForMode: (mode: unknown) => {
      const normalized = normalizeWeChatConnectMode(mode);
      if (normalized === "mp") return "snsapi_userinfo";
      if (normalized === "mobile") return "";
      return "snsapi_login";
    },
    deriveWeChatConnectStoredMode: (
      openEnabled: boolean,
      mpEnabled: boolean,
      mobileEnabled: boolean,
      legacyMode: unknown,
    ) => {
      if (mpEnabled) return "mp";
      if (mobileEnabled) return "mobile";
      if (openEnabled) return "open";
      return normalizeWeChatConnectMode(legacyMode);
    },
    resolveWeChatConnectModeCapabilities: (
      openEnabled: unknown,
      mpEnabled: unknown,
      mobileEnabled: unknown,
      legacyMode: unknown,
    ) => {
      if (
        typeof openEnabled === "boolean" ||
        typeof mpEnabled === "boolean" ||
        typeof mobileEnabled === "boolean"
      ) {
        return {
          openEnabled: openEnabled === true,
          mpEnabled: mpEnabled === true,
          mobileEnabled: mobileEnabled === true,
        };
      }
      const normalized = normalizeWeChatConnectMode(legacyMode);
      return {
        openEnabled: normalized === "open",
        mpEnabled: normalized === "mp",
        mobileEnabled: normalized === "mobile",
      };
    },
  };
});

vi.mock("@/stores", () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning: vi.fn(),
    showInfo: vi.fn(),
    fetchPublicSettings,
  }),
}));

vi.mock("@/stores/adminSettings", () => ({
  useAdminSettingsStore: () => ({
    fetch: adminSettingsFetch,
  }),
}));

vi.mock("@/composables/useClipboard", () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(),
  }),
}));

vi.mock("@/utils/apiError", () => ({
  extractApiErrorMessage: () => "error",
}));

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  const translations: Record<string, string> = {
    "admin.settings.wechatConnect.title": "微信登录",
    "admin.settings.wechatConnect.description": "用于微信开放平台或公众号/小程序的第三方登录配置。",
    "admin.settings.wechatConnect.enabledLabel": "启用微信登录",
    "admin.settings.wechatConnect.enabledHint": "开启后可使用微信第三方登录回调与授权配置。",
    "admin.settings.wechatConnect.appIdLabel": "AppID",
    "admin.settings.wechatConnect.appIdPlaceholder": "微信开放平台 AppID",
    "admin.settings.wechatConnect.appSecretLabel": "AppSecret",
    "admin.settings.wechatConnect.appSecretConfiguredPlaceholder": "密钥已配置，留空以保留当前值。",
    "admin.settings.wechatConnect.appSecretPlaceholder": "微信开放平台 AppSecret",
    "admin.settings.wechatConnect.appSecretConfiguredHint": "密钥已配置，留空以保留当前值。",
    "admin.settings.wechatConnect.appSecretHint": "填写后会覆盖当前微信密钥。",
    "admin.settings.wechatConnect.modeLabel": "模式",
    "admin.settings.wechatConnect.openModeLabel": "非微信环境使用开放平台",
    "admin.settings.wechatConnect.openModeHint": "浏览器不在微信内时，自动走开放平台扫码授权。",
    "admin.settings.wechatConnect.mpModeLabel": "微信环境使用公众号",
    "admin.settings.wechatConnect.mpModeHint": "浏览器在微信内时，自动走公众号授权。",
    "admin.settings.wechatConnect.redirectUrlLabel": "回调地址",
    "admin.settings.wechatConnect.redirectUrlPlaceholder": "https://your-site.com/api/v1/auth/oauth/wechat/callback",
    "admin.settings.wechatConnect.generateAndCopy": "使用当前站点生成并复制",
    "admin.settings.wechatConnect.redirectUrlSetAndCopied": "已使用当前站点生成回调地址并复制到剪贴板",
    "admin.settings.wechatConnect.frontendRedirectUrlLabel": "前端回调地址",
    "admin.settings.wechatConnect.frontendRedirectUrlPlaceholder": "/auth/wechat/callback",
    "admin.settings.wechatConnect.frontendRedirectUrlHint": "通常用于前端路由回调地址，需与后端配置保持一致。",
    "admin.settings.authSourceDefaults.title": "认证来源默认值",
    "admin.settings.authSourceDefaults.description": "按注册来源配置新用户默认余额、并发、订阅与授权策略。",
    "admin.settings.authSourceDefaults.requireEmailLabel": "第三方注册强制补充邮箱",
    "admin.settings.authSourceDefaults.requireEmailHint": "启用后，Linux DO、OIDC、微信注册缺少邮箱时必须先补充邮箱地址。",
    "admin.settings.authSourceDefaults.enabledHint": "以下默认值会在该来源注册新用户时发放；首次绑定时授权仅作用于已有账号绑定该来源。",
    "admin.settings.authSourceDefaults.sources.email.title": "邮箱注册",
    "admin.settings.authSourceDefaults.sources.email.description": "适用于邮箱密码注册的新用户默认配额。",
    "admin.settings.authSourceDefaults.sources.linuxdo.title": "Linux DO 登录",
    "admin.settings.authSourceDefaults.sources.linuxdo.description": "适用于 Linux DO 第三方注册的新用户默认配额。",
    "admin.settings.authSourceDefaults.sources.oidc.title": "OIDC 登录",
    "admin.settings.authSourceDefaults.sources.oidc.description": "适用于 OIDC 第三方注册的新用户默认配额。",
    "admin.settings.authSourceDefaults.sources.wechat.title": "微信登录",
    "admin.settings.authSourceDefaults.sources.wechat.description": "适用于微信第三方注册的新用户默认配额。",
    "admin.settings.authSourceDefaults.grantOnFirstBindLabel": "首次绑定时授权",
    "admin.settings.authSourceDefaults.grantOnFirstBindHint": "已有账号首次绑定该来源时发放默认权益。",
    "admin.settings.authSourceDefaults.defaultSubscriptionsLabel": "默认订阅",
    "admin.settings.authSourceDefaults.defaultSubscriptionsHint": "仅对当前认证来源生效，未配置时不追加来源专属订阅。",
    "admin.settings.authSourceDefaults.noSourceSubscriptions": "当前来源未配置专属默认订阅。",
    "admin.settings.paymentVisibleMethods.methodLabel": "{title} 可见方式",
    "admin.settings.paymentVisibleMethods.methodHint": "控制前台结算页是否展示该方式，以及展示时使用的来源键。",
    "admin.settings.paymentVisibleMethods.sourceLabel": "支付来源",
    "admin.settings.paymentVisibleMethods.sourceHint": "启用后必须明确选择一个来源；未配置状态不会对外展示该支付方式。",
    "admin.settings.paymentVisibleMethods.sourceRequiredError": "{title} 已启用，请先选择支付来源。",
    "admin.settings.payment.configGuide": "查看支付配置说明",
    "admin.settings.payment.findProvider": "查看支持的支付方式",
    "admin.settings.openaiExperimentalScheduler.title": "OpenAI 实验调度策略",
    "admin.settings.openaiExperimentalScheduler.description": "默认关闭。开启后仅影响本网关在 OpenAI 账号间的实验性调度选择逻辑，不代表上游 OpenAI 官方能力。",
    "admin.settings.site.uploadImage": "上传图片",
    "admin.settings.site.remove": "移除",
    "admin.settings.platformQuota.platform": "平台",
    "admin.settings.platformQuota.daily": "日限额 (USD)",
    "admin.settings.platformQuota.weekly": "周限额 (USD)",
    "admin.settings.platformQuota.monthly": "月限额 (USD, 30天滚动)",
    "admin.settings.platformQuota.placeholder": "不限",
    "admin.settings.defaults.defaultPlatformQuotas": "默认平台限额（注册时分配）",
    "admin.settings.defaults.defaultPlatformQuotasHint": "新用户注册时自动写入平台限额记录；已有用户不受影响。留空 = 该平台该窗口不限制。",
    "admin.settings.defaults.platformQuotaNotice": "月限额为 30 天滚动窗口，非自然月",
    "admin.settings.authSourceDefaults.platformQuotasOverride": "平台限额覆盖",
    "admin.settings.authSourceDefaults.platformQuotasOverrideHint": "留空的字段继承「系统默认平台限额」；填 0 表示禁止该窗口使用。",
  };
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) =>
        (translations[key] ?? key).replace(/\{(\w+)\}/g, (_, token) => params?.[token] ?? `{${token}}`),
      locale: localeRef,
    }),
  };
});

const AppLayoutStub = { template: "<div><slot /></div>" };
const ToggleStub = defineComponent({
  props: {
    modelValue: {
      type: Boolean,
      default: false,
    },
  },
  emits: ["update:modelValue"],
  inheritAttrs: false,
  setup(props, { attrs, emit }) {
    return () =>
      h("input", {
        ...attrs,
        class: "toggle-stub",
        type: "checkbox",
        checked: props.modelValue,
        onChange: (event: Event) => {
          emit("update:modelValue", (event.target as HTMLInputElement).checked);
        },
      });
  },
});

const SelectStub = defineComponent({
  props: {
    modelValue: {
      type: [String, Number, Boolean, null],
      default: "",
    },
    options: {
      type: Array,
      default: () => [],
    },
    placeholder: {
      type: String,
      default: "",
    },
  },
  emits: ["update:modelValue", "change"],
  setup(props, { emit }) {
    const onChange = (event: Event) => {
      const target = event.target as HTMLSelectElement;
      emit("update:modelValue", target.value);
      const option =
        (props.options as Array<Record<string, unknown>>).find(
          (item) => String(item.value ?? "") === target.value,
        ) ?? null;
      emit("change", target.value, option);
    };

    return () =>
      h(
        "select",
        {
          class: "select-stub",
          value: props.modelValue ?? "",
          "data-placeholder": props.placeholder,
          onChange,
        },
        (props.options as Array<Record<string, unknown>>).map((option) =>
          h(
            "option",
            {
              key: `${String(option.value ?? "")}:${String(option.label ?? "")}`,
              value: option.value as string,
            },
            String(option.label ?? ""),
          ),
        ),
      );
  },
});

const ImageUploadStub = defineComponent({
  props: {
    modelValue: {
      type: String,
      default: "",
    },
    uploadLabel: {
      type: String,
      default: "",
    },
    removeLabel: {
      type: String,
      default: "",
    },
    placeholder: {
      type: String,
      default: "",
    },
    maxSize: {
      type: Number,
      default: undefined,
    },
  },
  setup(props) {
    return () =>
      h("div", {
        class: "image-upload-stub",
        "data-model-value": props.modelValue,
        "data-upload-label": props.uploadLabel,
        "data-remove-label": props.removeLabel,
        "data-placeholder": props.placeholder,
        "data-max-size": props.maxSize == null ? "" : String(props.maxSize),
      });
  },
});

const baseSettingsResponse = {
  registration_enabled: true,
  email_verify_enabled: false,
  registration_email_suffix_whitelist: [],
  promo_code_enabled: true,
  invitation_code_enabled: false,
  password_reset_enabled: false,
  totp_enabled: false,
  totp_encryption_key_configured: false,
  default_balance: 0,
  default_concurrency: 1,
  default_subscriptions: [],
  site_name: "Sub2API",
  site_logo: "",
  site_subtitle: "",
  api_base_url: "",
  contact_info: "",
  doc_url: "",
  home_content: "",
  hide_ccs_import_button: false,
  table_default_page_size: 20,
  table_page_size_options: [10, 20, 50, 100],
  backend_mode_enabled: false,
  custom_menu_items: [],
  custom_endpoints: [],
  frontend_url: "",
  smtp_host: "",
  smtp_port: 587,
  smtp_username: "",
  smtp_password_configured: false,
  smtp_from_email: "",
  smtp_from_name: "",
  smtp_use_tls: true,
  turnstile_enabled: false,
  turnstile_site_key: "",
  turnstile_secret_key_configured: false,
  linuxdo_connect_enabled: false,
  linuxdo_connect_client_id: "",
  linuxdo_connect_client_secret_configured: false,
  linuxdo_connect_redirect_url: "",
  wechat_connect_enabled: true,
  wechat_connect_app_id: "wx-app-id-123",
  wechat_connect_app_secret_configured: true,
  wechat_connect_open_enabled: false,
  wechat_connect_mp_enabled: true,
  wechat_connect_mode: "mp",
  wechat_connect_scopes: "",
  wechat_connect_redirect_url:
    "https://admin.example.com/api/v1/auth/oauth/wechat/callback",
  wechat_connect_frontend_redirect_url: "/auth/wechat/callback",
  oidc_connect_enabled: false,
  oidc_connect_provider_name: "OIDC",
  oidc_connect_client_id: "",
  oidc_connect_client_secret_configured: false,
  oidc_connect_issuer_url: "",
  oidc_connect_discovery_url: "",
  oidc_connect_authorize_url: "",
  oidc_connect_token_url: "",
  oidc_connect_userinfo_url: "",
  oidc_connect_jwks_url: "",
  oidc_connect_scopes: "openid email profile",
  oidc_connect_redirect_url: "",
  oidc_connect_frontend_redirect_url: "/auth/oidc/callback",
  oidc_connect_token_auth_method: "client_secret_post",
  oidc_connect_use_pkce: true,
  oidc_connect_validate_id_token: true,
  oidc_connect_allowed_signing_algs: "RS256,ES256,PS256",
  oidc_connect_clock_skew_seconds: 120,
  oidc_connect_require_email_verified: false,
  oidc_connect_userinfo_email_path: "",
  oidc_connect_userinfo_id_path: "",
  oidc_connect_userinfo_username_path: "",
  enable_model_fallback: false,
  fallback_model_anthropic: "",
  fallback_model_openai: "",
  fallback_model_gemini: "",
  fallback_model_antigravity: "",
  enable_identity_patch: false,
  identity_patch_prompt: "",
  ops_monitoring_enabled: false,
  ops_realtime_monitoring_enabled: false,
  ops_query_mode_default: "auto",
  ops_metrics_interval_seconds: 60,
  min_claude_code_version: "",
  max_claude_code_version: "",
  allow_ungrouped_key_scheduling: false,
  enable_fingerprint_unification: true,
  enable_metadata_passthrough: false,
  enable_cch_signing: false,
  enable_anthropic_cache_ttl_1h_injection: false,
  payment_enabled: true,
  payment_min_amount: 1,
  payment_max_amount: 10000,
  payment_daily_limit: 50000,
  payment_order_timeout_minutes: 30,
  payment_max_pending_orders: 3,
  payment_enabled_types: [],
  payment_balance_disabled: false,
  payment_subscription_balance_enabled: false,
  payment_balance_recharge_multiplier: 1,
  payment_recharge_fee_rate: 0,
  payment_load_balance_strategy: "round-robin",
  payment_product_name_prefix: "",
  payment_product_name_suffix: "",
  payment_help_image_url: "",
  payment_help_text: "",
  payment_cancel_rate_limit_enabled: false,
  payment_cancel_rate_limit_max: 10,
  payment_cancel_rate_limit_window: 1,
  payment_cancel_rate_limit_unit: "day",
  payment_cancel_rate_limit_window_mode: "rolling",
  user_subscriptions_visible: true,
  payment_visible_method_alipay_source: "alipay_direct",
  payment_visible_method_wxpay_source: "invalid-source",
  payment_visible_method_alipay_enabled: true,
  payment_visible_method_wxpay_enabled: true,
  openai_advanced_scheduler_enabled: false,
  balance_low_notify_enabled: false,
  balance_low_notify_threshold: 0,
  balance_low_notify_recharge_url: "",
  account_quota_notify_enabled: false,
  account_quota_notify_emails: [],
  site_messages_enabled: false,
  site_messages_daily_send_limit: 10,
  site_messages_retention_days: 30,
  site_messages_default_recipient_email: "",
  // 平台限额嵌套字段（新后端契约）
  default_platform_quotas: {
    anthropic:   { daily: null, weekly: null, monthly: null },
    openai:      { daily: null, weekly: 12.5, monthly: null },
    gemini:      { daily: null, weekly: null, monthly: 200 },
    antigravity: { daily: null, weekly: null, monthly: null },
  },
};

function mountView() {
  return mount(SettingsView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        RouterLink: { template: "<a><slot /></a>" },
        "router-link": { template: "<a><slot /></a>" },
        Select: SelectStub,
        Toggle: ToggleStub,
        Icon: true,
        ConfirmDialog: true,
        PaymentProviderList: true,
        PaymentProviderDialog: true,
        GroupBadge: true,
        GroupOptionItem: true,
        ProxySelector: true,
        ImageUpload: ImageUploadStub,
        BackupSettings: true,
      },
    },
  });
}

async function openPaymentTab(wrapper: ReturnType<typeof mountView>) {
  const paymentTabButton = wrapper
    .findAll("button")
    .find((node) => node.text().includes("admin.settings.tabs.payment"));

  expect(paymentTabButton).toBeDefined();
  await paymentTabButton?.trigger("click");
  await flushPromises();
}

async function openSecurityTab(wrapper: ReturnType<typeof mountView>) {
  const securityTabButton = wrapper
    .findAll("button")
    .find((node) => node.text().includes("admin.settings.tabs.security"));

  expect(securityTabButton).toBeDefined();
  await securityTabButton?.trigger("click");
  await flushPromises();
}

async function openUsersTab(wrapper: ReturnType<typeof mountView>) {
  const usersTabButton = wrapper
    .findAll("button")
    .find((node) => node.text().includes("admin.settings.tabs.users"));

  expect(usersTabButton).toBeDefined();
  await usersTabButton?.trigger("click");
  await flushPromises();
}

async function openFeaturesTab(wrapper: ReturnType<typeof mountView>) {
  const featuresTabButton = wrapper
    .findAll("button")
    .find((node) => node.text().includes("admin.settings.tabs.features"));

  expect(featuresTabButton).toBeDefined();
  await featuresTabButton?.trigger("click");
  await flushPromises();
}

async function openModelMarketTab(wrapper: ReturnType<typeof mountView>) {
  const modelMarketTabButton = wrapper
    .findAll("button")
    .find((node) => node.text().includes("admin.settings.tabs.pricing"));

  expect(modelMarketTabButton).toBeDefined();
  await modelMarketTabButton?.trigger("click");
  await flushPromises();
}

describe("admin SettingsView payment visible method controls", () => {
  beforeEach(() => {
    getSettings.mockReset();
    updateSettings.mockReset();
    getWebSearchEmulationConfig.mockReset();
    updateWebSearchEmulationConfig.mockReset();
    getAdminApiKey.mockReset();
    getOverloadCooldownSettings.mockReset();
    getRateLimit429CooldownSettings.mockReset();
    updateRateLimit429CooldownSettings.mockReset();
    getStreamTimeoutSettings.mockReset();
    getRectifierSettings.mockReset();
    getBetaPolicySettings.mockReset();
    getModelMarket.mockReset();
    updateModelMarket.mockReset();
    getGroups.mockReset();
    listProxies.mockReset();
    getProviders.mockReset();
    updateProvider.mockReset();
    createProvider.mockReset();
    deleteProvider.mockReset();
    fetchPublicSettings.mockReset();
    adminSettingsFetch.mockReset();
    showError.mockReset();
    showSuccess.mockReset();
    localeRef.value = "zh-CN";

    getSettings.mockResolvedValue({ ...baseSettingsResponse });
    updateSettings.mockImplementation(async (payload) => ({
      ...baseSettingsResponse,
      ...payload,
    }));
    getWebSearchEmulationConfig.mockResolvedValue({
      enabled: false,
      providers: [],
    });
    updateWebSearchEmulationConfig.mockResolvedValue({
      enabled: false,
      providers: [],
    });
    getAdminApiKey.mockResolvedValue({
      exists: false,
      masked_key: "",
    });
    getOverloadCooldownSettings.mockResolvedValue({
      enabled: true,
      cooldown_minutes: 10,
    });
    getRateLimit429CooldownSettings.mockResolvedValue({
      enabled: true,
      cooldown_seconds: 5,
    });
    updateRateLimit429CooldownSettings.mockImplementation(async (payload) => payload);
    getStreamTimeoutSettings.mockResolvedValue({
      enabled: true,
      action: "temp_unsched",
      temp_unsched_minutes: 5,
      threshold_count: 3,
      threshold_window_minutes: 10,
    });
    getRectifierSettings.mockResolvedValue({
      enabled: true,
      thinking_signature_enabled: true,
      thinking_budget_enabled: true,
      apikey_signature_enabled: false,
      apikey_signature_patterns: [],
    });
    getBetaPolicySettings.mockResolvedValue({
      rules: [],
    });
    getModelMarket.mockResolvedValue({
      config: {
        enabled: true,
        auto_sync: true,
        title: "模型广场",
        description: "按平台、分组和计费类型查看当前可用模型。",
        selected_models: [],
        custom_models: [],
      },
      candidates: [],
      models: [],
    });
    updateModelMarket.mockImplementation(async (payload) => ({
      config: payload,
      candidates: [],
      models: [],
    }));
    getGroups.mockResolvedValue([]);
    listProxies.mockResolvedValue({
      items: [],
    });
    getProviders.mockResolvedValue({
      data: [],
    });
    fetchPublicSettings.mockResolvedValue(undefined);
    adminSettingsFetch.mockResolvedValue(undefined);
  });

  it("does not render legacy visible payment method controls", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openPaymentTab(wrapper);

    expect(wrapper.text()).not.toContain("可见方式");
    expect(wrapper.text()).not.toContain("支付来源");
  });

  it("renders Mapay in the enabled payment types selector", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openPaymentTab(wrapper);

    const enabledTypeButtons = wrapper
      .findAll("button")
      .map((node) => node.text());

    expect(enabledTypeButtons).toContain("payment.methods.mapay");
  });

  it("links payment guidance to README sections instead of removed payment docs", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openPaymentTab(wrapper);

    const paymentLinks = wrapper
      .findAll("a")
      .filter((node) =>
        ["查看支付配置说明", "查看支持的支付方式"].includes(node.text()),
      );

    expect(paymentLinks).toHaveLength(2);
    expect(paymentLinks[0]?.attributes("href")).toBe(
      "https://github.com/Wei-Shaw/sub2api/blob/main/docs/PAYMENT_CN.md",
    );
    expect(paymentLinks[1]?.attributes("href")).toBe(
      "https://github.com/Wei-Shaw/sub2api/blob/main/docs/PAYMENT_CN.md#支持的支付方式",
    );
    for (const link of paymentLinks) {
      expect(link.attributes("href")).toContain("docs/PAYMENT");
    }
  });

  it("does not submit legacy visible payment method settings", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openPaymentTab(wrapper);
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    const payload = updateSettings.mock.calls[0]?.[0];
    expect(payload).not.toHaveProperty("payment_visible_method_alipay_source");
    expect(payload).not.toHaveProperty("payment_visible_method_wxpay_source");
    expect(payload).not.toHaveProperty("payment_visible_method_alipay_enabled");
    expect(payload).not.toHaveProperty("payment_visible_method_wxpay_enabled");
  });

  it("preserves public site page link mode when saving", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      site_pages: [
        {
          key: "docs",
          title: "Docs",
          slug: "doc/docs",
          mode: "link",
          content: "https://blog.lumio.games/docs/doc/api",
          enabled: true,
        },
      ],
    });

    const wrapper = mountView();

    await flushPromises();
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        site_pages: expect.arrayContaining([
          expect.objectContaining({
            key: "docs",
            mode: "link",
            content: "https://blog.lumio.games/docs/doc/api",
          }),
        ]),
      }),
    );
  });

  it("places user subscriptions visibility under feature switches, not general settings", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      user_subscriptions_visible: false,
    });

    const wrapper = mountView();

    await flushPromises();

    await openFeaturesTab(wrapper);

    const pageText = wrapper.text();

    expect(pageText).not.toContain("admin.settings.site.userSubscriptionsVisible");
    expect(pageText).toContain("admin.settings.features.userSubscriptions.title");
  });

  it("submits the user subscriptions visibility setting", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      user_subscriptions_visible: false,
    });

    const wrapper = mountView();

    await flushPromises();
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        user_subscriptions_visible: false,
      }),
    );
  });

  it("submits the multiple subscription purchases setting", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      subscription_multiple_purchases_enabled: true,
    });

    const wrapper = mountView();

    await flushPromises();
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        subscription_multiple_purchases_enabled: true,
      }),
    );
  });

  it("submits Anthropic cache TTL injection gateway setting", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      enable_anthropic_cache_ttl_1h_injection: true,
    });

    const wrapper = mountView();

    await flushPromises();
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        enable_anthropic_cache_ttl_1h_injection: true,
      }),
    );
  });

  it("submits trimmed site message default recipient email", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      site_messages_enabled: true,
      site_messages_default_recipient_email: " support@lumio.games ",
    });

    const wrapper = mountView();

    await flushPromises();
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        site_messages_default_recipient_email: "support@lumio.games",
      }),
    );
  });

  it("ignores duplicate save submissions while the settings request is in flight", async () => {
    let resolveUpdate: ((value: typeof baseSettingsResponse) => void) | undefined;
    updateSettings.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveUpdate = resolve;
        }),
    );

    const wrapper = mountView();

    await flushPromises();
    const form = wrapper.find("form");
    await form.trigger("submit.prevent");
    await form.trigger("submit.prevent");

    expect(updateSettings).toHaveBeenCalledTimes(1);

    resolveUpdate?.({ ...baseSettingsResponse });
    await flushPromises();
  });

  it("updates provider enablement immediately and reloads providers", async () => {
    const provider = {
      id: 7,
      provider_key: "alipay",
      name: "Official Alipay",
      config: {},
      supported_types: ["alipay"],
      enabled: false,
      payment_mode: "",
      refund_enabled: false,
      allow_user_refund: false,
      limits: "",
      sort_order: 0,
    };
    getProviders.mockReset();
    getProviders
      .mockResolvedValueOnce({ data: [provider] })
      .mockResolvedValueOnce({ data: [{ ...provider, enabled: true }] });
    updateProvider.mockResolvedValue({ data: { ...provider, enabled: true } });

    const PaymentProviderListStub = defineComponent({
      emits: ["toggleField"],
      setup(_, { emit }) {
        return () =>
          h(
            "button",
            {
              class: "provider-toggle-stub",
              onClick: () => emit("toggleField", provider, "enabled"),
            },
            "toggle provider",
          );
      },
    });

    const wrapper = mount(SettingsView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          Select: SelectStub,
          Toggle: ToggleStub,
          Icon: true,
          ConfirmDialog: true,
          PaymentProviderList: PaymentProviderListStub,
          PaymentProviderDialog: true,
          GroupBadge: true,
          GroupOptionItem: true,
          ProxySelector: true,
          ImageUpload: ImageUploadStub,
          BackupSettings: true,
        },
      },
    });

    await flushPromises();
    await openPaymentTab(wrapper);
    await wrapper.get(".provider-toggle-stub").trigger("click");
    await flushPromises();

    expect(updateProvider).toHaveBeenCalledWith(7, { enabled: true });
    expect(getProviders).toHaveBeenCalledTimes(2);
  });

  it("renders advanced scheduler copy as local experimental gateway policy", async () => {
    const wrapper = mountView();

    await flushPromises();

    expect(wrapper.text()).toContain("OpenAI 实验调度策略");
    expect(wrapper.text()).toContain(
      "默认关闭。开启后仅影响本网关在 OpenAI 账号间的实验性调度选择逻辑",
    );
    expect(wrapper.text()).not.toContain("OpenAI 高级调度器");
  });

  it("passes translated upload and remove labels to the payment help image uploader", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openPaymentTab(wrapper);

    const imageUploads = wrapper.findAll(".image-upload-stub");
    expect(imageUploads.length).toBeGreaterThan(0);

    const paymentHelpImageUpload = imageUploads.find(
      (node) => node.attributes("data-placeholder") === "admin.settings.payment.helpImagePlaceholder",
    );

    expect(paymentHelpImageUpload).toBeDefined();
    expect(paymentHelpImageUpload?.attributes("data-upload-label")).toBe("上传图片");
    expect(paymentHelpImageUpload?.attributes("data-remove-label")).toBe("移除");
    expect(paymentHelpImageUpload?.attributes("data-max-size")).toBe(String(1024 * 1024));
  });
});

describe("admin SettingsView wechat connect controls", () => {
  beforeEach(() => {
    getSettings.mockReset();
    updateSettings.mockReset();
    getWebSearchEmulationConfig.mockReset();
    updateWebSearchEmulationConfig.mockReset();
    getAdminApiKey.mockReset();
    getOverloadCooldownSettings.mockReset();
    getRateLimit429CooldownSettings.mockReset();
    updateRateLimit429CooldownSettings.mockReset();
    getStreamTimeoutSettings.mockReset();
    getRectifierSettings.mockReset();
    getBetaPolicySettings.mockReset();
    getModelMarket.mockReset();
    updateModelMarket.mockReset();
    getGroups.mockReset();
    listProxies.mockReset();
    getProviders.mockReset();
    updateProvider.mockReset();
    createProvider.mockReset();
    deleteProvider.mockReset();
    fetchPublicSettings.mockReset();
    adminSettingsFetch.mockReset();
    showError.mockReset();
    showSuccess.mockReset();

    getSettings.mockResolvedValue({
      ...baseSettingsResponse,
      payment_visible_method_wxpay_source: "official_wxpay",
    });
    updateSettings.mockImplementation(async (payload) => ({
      ...baseSettingsResponse,
      payment_visible_method_wxpay_source: "official_wxpay",
      ...payload,
    }));
    getWebSearchEmulationConfig.mockResolvedValue({
      enabled: false,
      providers: [],
    });
    updateWebSearchEmulationConfig.mockResolvedValue({
      enabled: false,
      providers: [],
    });
    getAdminApiKey.mockResolvedValue({
      exists: false,
      masked_key: "",
    });
    getOverloadCooldownSettings.mockResolvedValue({
      enabled: true,
      cooldown_minutes: 10,
    });
    getRateLimit429CooldownSettings.mockResolvedValue({
      enabled: true,
      cooldown_seconds: 5,
    });
    updateRateLimit429CooldownSettings.mockImplementation(async (payload) => payload);
    getStreamTimeoutSettings.mockResolvedValue({
      enabled: true,
      action: "temp_unsched",
      temp_unsched_minutes: 5,
      threshold_count: 3,
      threshold_window_minutes: 10,
    });
    getRectifierSettings.mockResolvedValue({
      enabled: true,
      thinking_signature_enabled: true,
      thinking_budget_enabled: true,
      apikey_signature_enabled: false,
      apikey_signature_patterns: [],
    });
    getBetaPolicySettings.mockResolvedValue({
      rules: [],
    });
    getModelMarket.mockResolvedValue({
      config: {
        enabled: true,
        auto_sync: true,
        title: "模型广场",
        description: "按平台、分组和计费类型查看当前可用模型。",
        selected_models: [],
        custom_models: [],
      },
      candidates: [],
      models: [],
    });
    updateModelMarket.mockImplementation(async (payload) => ({
      config: payload,
      candidates: [],
      models: [],
    }));
    getGroups.mockResolvedValue([]);
    listProxies.mockResolvedValue({
      items: [],
    });
    getProviders.mockResolvedValue({
      data: [],
    });
    fetchPublicSettings.mockResolvedValue(undefined);
    adminSettingsFetch.mockResolvedValue(undefined);
  });

  it("loads and echoes WeChat Connect fields from the backend payload", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openSecurityTab(wrapper);

    expect(
      (
        wrapper.get('[data-testid="wechat-connect-mp-app-id"]')
          .element as HTMLInputElement
      ).value,
    ).toBe("wx-app-id-123");
    expect(
      (
        wrapper.get('[data-testid="wechat-connect-open-enabled"]')
          .element as HTMLInputElement
      ).checked,
    ).toBe(false);
    expect(
      (
        wrapper.get('[data-testid="wechat-connect-mp-enabled"]')
          .element as HTMLInputElement
      ).checked,
    ).toBe(true);
    expect(wrapper.find('[data-testid="wechat-connect-scopes"]').exists()).toBe(
      false,
    );
    expect(
      wrapper
        .get('[data-testid="wechat-connect-mp-app-secret"]')
        .attributes("placeholder"),
    ).toContain("密钥已配置");
    expect(
      (
        wrapper.get('[data-testid="wechat-connect-frontend-redirect-url"]')
          .element as HTMLInputElement
      ).value,
    ).toBe("/auth/wechat/callback");
  });

  it("links GitHub OAuth Apps guide to GitHub developer settings", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      github_oauth_enabled: true,
    });

    const wrapper = mountView();

    await flushPromises();
    await openSecurityTab(wrapper);

    const link = wrapper.get('[data-testid="github-oauth-apps-guide-link"]');
    expect(link.text()).toContain("OAuth Apps");
    expect(link.attributes("href")).toBe("https://github.com/settings/developers");
    expect(link.attributes("target")).toBe("_blank");
    expect(link.attributes("rel")).toContain("noopener");
  });

  it("saves WeChat Connect fields using the backend contract and clears the secret after save", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openSecurityTab(wrapper);

    await wrapper
      .get('[data-testid="wechat-connect-mp-app-id"]')
      .setValue("wx-app-id-updated");
    await wrapper
      .get('[data-testid="wechat-connect-mp-app-secret"]')
      .setValue("new-secret");
    await wrapper
      .get('[data-testid="wechat-connect-open-enabled"]')
      .setValue(true);
    await wrapper
      .get('[data-testid="wechat-connect-mp-enabled"]')
      .setValue(true);
    await wrapper
      .get('[data-testid="wechat-connect-redirect-url"]')
      .setValue("https://admin.example.com/api/v1/auth/oauth/wechat/callback");
    await wrapper
      .get('[data-testid="wechat-connect-frontend-redirect-url"]')
      .setValue("/auth/wechat/callback");
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        wechat_connect_enabled: true,
        wechat_connect_app_id: "wx-app-id-updated",
        wechat_connect_open_enabled: true,
        wechat_connect_mp_enabled: true,
        wechat_connect_mp_app_id: "wx-app-id-updated",
        wechat_connect_mp_app_secret: "new-secret",
        wechat_connect_redirect_url:
          "https://admin.example.com/api/v1/auth/oauth/wechat/callback",
        wechat_connect_frontend_redirect_url: "/auth/wechat/callback",
      }),
    );
    expect(
      (
        wrapper.get('[data-testid="wechat-connect-mp-app-secret"]')
          .element as HTMLInputElement
      ).value,
    ).toBe("");
    expect(
      wrapper
        .get('[data-testid="wechat-connect-mp-app-secret"]')
        .attributes("placeholder"),
    ).toContain("密钥已配置");
  });

  it("omits unchanged site_logo from the settings save payload", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      site_logo: "data:image/png;base64," + "a".repeat(1024),
    });

    const wrapper = mountView();

    await flushPromises();
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(updateSettings.mock.calls[0][0]).not.toHaveProperty("site_logo");
  });

  it("collapses auth source defaults until the source is enabled", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openUsersTab(wrapper);

    expect(
      (
        wrapper.get('[data-testid="auth-source-email-enabled"]')
          .element as HTMLInputElement
      ).checked,
    ).toBe(false);
    expect(
      wrapper.find('[data-testid="auth-source-email-panel"]').exists(),
    ).toBe(false);
    expect(wrapper.text()).not.toContain("注册即授权");

    await wrapper
      .get('[data-testid="auth-source-email-enabled"]')
      .setValue(true);

    expect(
      wrapper.find('[data-testid="auth-source-email-panel"]').exists(),
    ).toBe(true);
    expect(wrapper.text()).toContain("首次绑定时授权");
  });

  it("preserves optional OIDC compatibility flags instead of forcing them on save", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      oidc_connect_enabled: true,
      oidc_connect_use_pkce: false,
      oidc_connect_validate_id_token: false,
    });

    const wrapper = mountView();

    await flushPromises();
    await openSecurityTab(wrapper);
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        oidc_connect_use_pkce: false,
        oidc_connect_validate_id_token: false,
      }),
    );
  });

  it("saves model market manual selections without duplicating price config", async () => {
    getModelMarket.mockResolvedValueOnce({
      config: {
        enabled: true,
        auto_sync: true,
        title: "模型广场",
        description: "按平台、分组和计费类型查看当前可用模型。",
        selected_models: [],
        custom_models: [],
      },
      candidates: [
        {
          key: "openai:gpt-5.4",
          name: "gpt-5.4",
          platform: "openai",
          billing_mode: "token",
          pricing: {
            billing_mode: "token",
            input_price: 0.000001,
            output_price: 0.000002,
            cache_write_price: null,
            cache_read_price: null,
            image_output_price: null,
            per_request_price: null,
            intervals: [],
          },
          groups: [
            {
              id: 1,
              name: "public",
              platform: "openai",
              subscription_type: "balance",
              rate_multiplier: 0.2,
              is_exclusive: false,
            },
          ],
          channels: ["OpenAI Public"],
          sort_order: 0,
        },
      ],
      models: [],
    });

    const wrapper = mountView();

    await flushPromises();
    await openModelMarketTab(wrapper);
    await wrapper.get('[data-testid="model-market-auto-sync"]').setValue(false);
    await flushPromises();
    await wrapper.get('[data-testid="model-market-select-openai:gpt-5.4"]').setValue(true);
    await flushPromises();
    await wrapper.get('[data-testid="model-market-sort-openai:gpt-5.4"]').setValue("20");
    await wrapper.get('[data-testid="model-market-save"]').trigger("click");
    await flushPromises();

    expect(updateModelMarket).toHaveBeenCalledTimes(1);
    expect(updateModelMarket).toHaveBeenCalledWith({
      enabled: true,
      auto_sync: false,
      title: "模型广场",
      description: "按平台、分组和计费类型查看当前可用模型。",
      selected_models: [
        {
          key: "openai:gpt-5.4",
          platform: "openai",
          model: "gpt-5.4",
          enabled: true,
          sort_order: 20,
        },
      ],
      custom_models: [],
    });
    expect(updateModelMarket.mock.calls[0]?.[0]).not.toHaveProperty("rows");
    expect(updateModelMarket.mock.calls[0]?.[0]).not.toHaveProperty("currency");
  });

  it("saves model market candidate billing overrides for token pricing", async () => {
    getModelMarket.mockResolvedValueOnce({
      config: {
        enabled: true,
        auto_sync: true,
        title: "模型广场",
        description: "按平台、分组和计费类型查看当前可用模型。",
        selected_models: [],
        custom_models: [],
      },
      candidates: [
        {
          key: "gemini:gemini-3.5-flash-high",
          name: "gemini-3.5-flash-high",
          platform: "gemini",
          billing_mode: "token",
          pricing: null,
          groups: [
            {
              id: 1,
              name: "Gemini",
              platform: "gemini",
              subscription_type: "balance",
              rate_multiplier: 0.35,
              is_exclusive: false,
            },
          ],
          channels: ["Gemini"],
          sort_order: 0,
        },
      ],
      models: [],
    });

    const wrapper = mountView();

    await flushPromises();
    await openModelMarketTab(wrapper);
    expect(wrapper.text()).not.toContain("admin.channels.form");
    await wrapper.get('[data-testid="model-market-candidate-input-price-gemini:gemini-3.5-flash-high"]').setValue("1.5");
    await wrapper.get('[data-testid="model-market-candidate-output-price-gemini:gemini-3.5-flash-high"]').setValue("9");
    await wrapper.get('[data-testid="model-market-save"]').trigger("click");
    await flushPromises();

    expect(updateModelMarket).toHaveBeenCalledTimes(1);
    expect(updateModelMarket).toHaveBeenCalledWith(
      expect.objectContaining({
        selected_models: [
          expect.objectContaining({
            key: "gemini:gemini-3.5-flash-high",
            platform: "gemini",
            model: "gemini-3.5-flash-high",
            enabled: true,
            billing_mode: "token",
            pricing: expect.objectContaining({
              billing_mode: "token",
              input_price: 0.0000015,
              output_price: 0.000009,
            }),
          }),
        ],
      }),
    );
  });

  it("saves model market candidate billing overrides for per-image tier pricing", async () => {
    getModelMarket.mockResolvedValueOnce({
      config: {
        enabled: true,
        auto_sync: true,
        title: "模型广场",
        description: "按平台、分组和计费类型查看当前可用模型。",
        selected_models: [],
        custom_models: [],
      },
      candidates: [
        {
          key: "openai:gpt-image-2",
          name: "gpt-image-2",
          platform: "openai",
          billing_mode: "token",
          pricing: {
            billing_mode: "token",
            input_price: 0.000005,
            output_price: 0.00001,
            cache_write_price: null,
            cache_read_price: 0.00000125,
            image_output_price: 0.00003,
            per_request_price: null,
            intervals: [],
          },
          groups: [
            {
              id: 1,
              name: "public",
              platform: "openai",
              subscription_type: "balance",
              rate_multiplier: 0.2,
              is_exclusive: false,
            },
          ],
          channels: ["OpenAI Public"],
          sort_order: 0,
        },
      ],
      models: [],
    });

    const wrapper = mountView();

    await flushPromises();
    await openModelMarketTab(wrapper);
    await wrapper.get('[data-testid="model-market-candidate-billing-openai:gpt-image-2"]').setValue("image");
    await flushPromises();
    await wrapper.get('[data-testid="model-market-candidate-image-tier-1k-openai:gpt-image-2"]').setValue("0.05");
    await wrapper.get('[data-testid="model-market-candidate-image-tier-4k-openai:gpt-image-2"]').setValue("0.15");
    await wrapper.get('[data-testid="model-market-save"]').trigger("click");
    await flushPromises();

    expect(updateModelMarket).toHaveBeenCalledTimes(1);
    expect(updateModelMarket).toHaveBeenCalledWith(
      expect.objectContaining({
        enabled: true,
        auto_sync: true,
        selected_models: [
          expect.objectContaining({
            key: "openai:gpt-image-2",
            platform: "openai",
            model: "gpt-image-2",
            enabled: true,
            billing_mode: "image",
            pricing: expect.objectContaining({
              billing_mode: "image",
              intervals: [
                expect.objectContaining({
                  tier_label: "1K",
                  per_request_price: 0.05,
                }),
                expect.objectContaining({
                  tier_label: "4K",
                  per_request_price: 0.15,
                }),
              ],
            }),
          }),
        ],
      }),
    );
  });

  it("adds custom model market entries with group, multiplier and pricing", async () => {
    getGroups.mockResolvedValue([
      {
        id: 42,
        name: "OpenAI Pro",
        platform: "openai",
        status: "active",
        subscription_type: "standard",
        rate_multiplier: 0.3,
        is_exclusive: false,
      },
    ]);

    const wrapper = mountView();

    await flushPromises();
    await openModelMarketTab(wrapper);
    await wrapper.get('[data-testid="model-market-add-custom"]').trigger("click");
    await flushPromises();

    await wrapper.get('[data-testid="model-market-custom-platform-0"]').setValue("openai");
    await wrapper.get('[data-testid="model-market-custom-model-0"]').setValue("gpt-custom");
    await wrapper.get('[data-testid="model-market-custom-group-0"]').setValue("42");
    await wrapper.get('[data-testid="model-market-custom-rate-0"]').setValue("0.55");
    await wrapper.get('[data-testid="model-market-custom-input-price-0"]').setValue("4");
    await wrapper.get('[data-testid="model-market-custom-output-price-0"]').setValue("12");
    await wrapper.get('[data-testid="model-market-custom-sort-0"]').setValue("30");
    await wrapper.get('[data-testid="model-market-save"]').trigger("click");
    await flushPromises();

    expect(updateModelMarket).toHaveBeenCalledTimes(1);
    expect(updateModelMarket).toHaveBeenCalledWith(
      expect.objectContaining({
        custom_models: [
          expect.objectContaining({
            key: "custom:openai:gpt-custom",
            platform: "openai",
            model: "gpt-custom",
            enabled: true,
            sort_order: 30,
            billing_mode: "token",
            pricing: expect.objectContaining({
              billing_mode: "token",
              input_price: 0.000004,
              output_price: 0.000012,
            }),
            groups: [
              expect.objectContaining({
                id: 42,
                name: "OpenAI Pro",
                platform: "openai",
                subscription_type: "standard",
                rate_multiplier: 0.55,
                is_exclusive: false,
              }),
            ],
          }),
        ],
      }),
    );
  });

  it("blocks model market custom save when the model name is empty", async () => {
    getGroups.mockResolvedValue([
      {
        id: 42,
        name: "OpenAI Pro",
        platform: "openai",
        status: "active",
        subscription_type: "standard",
        rate_multiplier: 0.3,
        is_exclusive: false,
      },
    ]);

    const wrapper = mountView();

    await flushPromises();
    await openModelMarketTab(wrapper);
    await wrapper.get('[data-testid="model-market-add-custom"]').trigger("click");
    await flushPromises();
    await wrapper.get('[data-testid="model-market-save"]').trigger("click");
    await flushPromises();

    expect(updateModelMarket).not.toHaveBeenCalled();
    expect(showError).toHaveBeenCalledWith("admin.settings.modelMarket.customModelRequired");
    expect(wrapper.find('[data-testid="model-market-custom-model-0"]').exists()).toBe(true);
  });

  it("blocks enabled model market custom save when no group is selected", async () => {
    getGroups.mockResolvedValue([]);

    const wrapper = mountView();

    await flushPromises();
    await openModelMarketTab(wrapper);
    await wrapper.get('[data-testid="model-market-add-custom"]').trigger("click");
    await flushPromises();
    await wrapper.get('[data-testid="model-market-custom-model-0"]').setValue("gpt-custom");
    await wrapper.get('[data-testid="model-market-save"]').trigger("click");
    await flushPromises();

    expect(updateModelMarket).not.toHaveBeenCalled();
    expect(showError).toHaveBeenCalledWith("admin.settings.modelMarket.customGroupRequired");
  });
});

describe("admin SettingsView platform quota matrix", () => {
  beforeEach(() => {
    getSettings.mockReset();
    updateSettings.mockReset();
    getWebSearchEmulationConfig.mockReset();
    updateWebSearchEmulationConfig.mockReset();
    getAdminApiKey.mockReset();
    getOverloadCooldownSettings.mockReset();
    getRateLimit429CooldownSettings.mockReset();
    updateRateLimit429CooldownSettings.mockReset();
    getStreamTimeoutSettings.mockReset();
    getRectifierSettings.mockReset();
    getBetaPolicySettings.mockReset();
    getGroups.mockReset();
    listProxies.mockReset();
    getProviders.mockReset();
    updateProvider.mockReset();
    createProvider.mockReset();
    deleteProvider.mockReset();
    fetchPublicSettings.mockReset();
    adminSettingsFetch.mockReset();
    showError.mockReset();
    showSuccess.mockReset();
    localeRef.value = "zh-CN";

    getSettings.mockResolvedValue({ ...baseSettingsResponse });
    updateSettings.mockImplementation(async (payload) => ({
      ...baseSettingsResponse,
      ...payload,
    }));
    getWebSearchEmulationConfig.mockResolvedValue({ enabled: false, providers: [] });
    updateWebSearchEmulationConfig.mockResolvedValue({ enabled: false, providers: [] });
    getAdminApiKey.mockResolvedValue({ exists: false, masked_key: "" });
    getOverloadCooldownSettings.mockResolvedValue({});
    getRateLimit429CooldownSettings.mockResolvedValue({});
    updateRateLimit429CooldownSettings.mockResolvedValue({});
    getStreamTimeoutSettings.mockResolvedValue({});
    getRectifierSettings.mockResolvedValue({});
    getBetaPolicySettings.mockResolvedValue({});
    getGroups.mockResolvedValue([]);
    listProxies.mockResolvedValue({ items: [] });
    getProviders.mockResolvedValue({ data: [] });
  });

  it("从 baseSettings 加载默认平台配额数据并在 Users tab 渲染 4 平台行", async () => {
    const wrapper = mountView();
    await flushPromises();
    await openUsersTab(wrapper);

    expect(getSettings).toHaveBeenCalled();

    const html = wrapper.html();
    // 表格行的平台字段：font-mono 渲染纯英文 platform key
    expect(html).toContain("anthropic");
    expect(html).toContain("openai");
    expect(html).toContain("gemini");
    expect(html).toContain("antigravity");
  });

  it("保存时 updateSettings payload 应包含嵌套 default_platform_quotas 对象（含全 4 平台）", async () => {
    const wrapper = mountView();
    await flushPromises();
    await openUsersTab(wrapper);

    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalled();
    const lastCallArgs = updateSettings.mock.calls.at(-1);
    expect(lastCallArgs).toBeDefined();
    const payload = lastCallArgs![0] as Record<string, unknown>;

    // 应携带嵌套对象，而非扁平字段
    expect(payload).toHaveProperty("default_platform_quotas");
    const quotas = payload["default_platform_quotas"] as Record<string, unknown>;
    const platforms = ["anthropic", "openai", "gemini", "antigravity"];
    for (const p of platforms) {
      expect(quotas).toHaveProperty(p);
      const pq = quotas[p] as Record<string, unknown>;
      expect(pq).toHaveProperty("daily");
      expect(pq).toHaveProperty("weekly");
      expect(pq).toHaveProperty("monthly");
    }

    // 不应存在旧扁平字段
    expect(payload).not.toHaveProperty("default_platform_quota_anthropic_daily");
    expect(payload).not.toHaveProperty("default_platform_quota_openai_weekly");
  });

  it("加载后 form.default_platform_quotas 含全 4 平台，从嵌套 JSON 正确读取数值", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      default_platform_quotas: {
        anthropic: { daily: 5, weekly: null, monthly: null },
        openai:    { daily: null, weekly: 12.5, monthly: null },
        // gemini / antigravity 缺失 → 应被归一化为全 null
      },
    });

    const wrapper = mountView();
    await flushPromises();
    await openUsersTab(wrapper);

    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    const payload = updateSettings.mock.calls.at(-1)![0] as Record<string, unknown>;
    const quotas = payload["default_platform_quotas"] as Record<string, Record<string, unknown>>;

    expect(quotas["anthropic"]?.["daily"]).toBe(5);
    expect(quotas["openai"]?.["weekly"]).toBe(12.5);
    // 缺失平台应补全为 null
    expect(quotas["gemini"]).toEqual({ daily: null, weekly: null, monthly: null });
    expect(quotas["antigravity"]).toEqual({ daily: null, weekly: null, monthly: null });
  });

  it("空输入（v-model.number 产出 \"\"）在提交时清洗为 null 而非空字符串", async () => {
    // 模拟后端返回带有 anthropic daily 值的配额
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      default_platform_quotas: {
        anthropic: { daily: 10, weekly: null, monthly: null },
        openai:    { daily: null, weekly: null, monthly: null },
        gemini:    { daily: null, weekly: null, monthly: null },
        antigravity: { daily: null, weekly: null, monthly: null },
      },
    });

    const wrapper = mountView();
    await flushPromises();
    await openUsersTab(wrapper);

    // 找到 anthropic daily 输入框并清空（模拟用户删除值）
    const inputs = wrapper.findAll('input[type="number"]');
    const anthropicDailyInput = inputs.find((i) => {
      const parent = i.element.closest("tr");
      return parent?.textContent?.includes("anthropic");
    });

    if (anthropicDailyInput) {
      // 设置为空字符串，模拟 v-model.number 在清空时产出 ""
      await anthropicDailyInput.setValue("");
    }

    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    const payload = updateSettings.mock.calls.at(-1)![0] as Record<string, unknown>;
    const quotas = payload["default_platform_quotas"] as Record<string, Record<string, unknown>>;
    // 不管输入是什么，提交值应为 null（而非 "" 或 NaN）
    expect(quotas["anthropic"]?.["daily"]).toBe(null);
  });
});
