import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";

import PublicMarkdownPageView from "../PublicMarkdownPageView.vue";

const routeState = vi.hoisted(() => ({
  params: {
    slug: ["api", "帮助文档"],
  },
}));

const routerPush = vi.hoisted(() => vi.fn());
const checkAuth = vi.hoisted(() => vi.fn());
const fetchPublicSettings = vi.hoisted(() => vi.fn());

const publicSettings = vi.hoisted(() => ({
  site_name: "LumioAPI",
  site_logo: "",
  site_pages: [
    {
      key: "docs",
      title: "帮助文档",
      slug: "doc/api/帮助文档",
      mode: "link",
      content:
        "https://blog.lumio.games/docs/doc/api/%E5%B8%AE%E5%8A%A9%E6%96%87%E6%A1%A3",
      enabled: true,
    },
  ],
}));

vi.mock("vue-router", () => ({
  useRoute: () => routeState,
  useRouter: () => ({
    push: routerPush,
  }),
}));

vi.mock("@/stores", () => ({
  useAppStore: () => ({
    cachedPublicSettings: publicSettings,
    publicSettingsLoaded: true,
    fetchPublicSettings,
    siteName: "Sub2API",
    siteLogo: "",
  }),
  useAuthStore: () => ({
    checkAuth,
    isAuthenticated: false,
    isAdmin: false,
  }),
}));

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: {
      value: "zh-CN",
    },
  }),
}));

describe("PublicMarkdownPageView", () => {
  const originalLocation = window.location;

  beforeEach(() => {
    routerPush.mockReset();
    checkAuth.mockReset();
    fetchPublicSettings.mockReset();
    Object.defineProperty(window, "location", {
      value: {
        ...originalLocation,
        assign: vi.fn(),
      },
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    Object.defineProperty(window, "location", {
      value: originalLocation,
      writable: true,
      configurable: true,
    });
    vi.restoreAllMocks();
  });

  it("renders link-mode public pages inside the current document page", async () => {
    const wrapper = mount(PublicMarkdownPageView, {
      global: {
        stubs: {
          Icon: true,
        },
      },
    });

    await flushPromises();

    const frame = wrapper.get("iframe.public-page-frame");
    expect(frame.attributes("src")).toBe(
      "https://blog.lumio.games/docs/doc/api/%E5%B8%AE%E5%8A%A9%E6%96%87%E6%A1%A3",
    );
    expect(window.location.assign).not.toHaveBeenCalled();
  });
});
