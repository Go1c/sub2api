import { describe, expect, it } from "vitest";

import {
  resolveSitePageNavigationTarget,
  resolveSitePageNavItem,
  resolveDocsNavItem,
  normalizeSitePages,
} from "../sitePages";

describe("site page helpers", () => {
  it("treats legacy pages without a mode as markdown routes", () => {
    const pages = normalizeSitePages([
      {
        key: "docs",
        title: "Docs",
        slug: "/doc/docs/",
        content: "# Docs",
        enabled: true,
      },
    ]);

    expect(pages[0]).toMatchObject({
      mode: "markdown",
      slug: "doc/docs",
    });
    expect(resolveSitePageNavigationTarget(pages, "docs")).toEqual({
      kind: "route",
      target: "/doc/docs",
    });
  });

  it("resolves link pages as external URLs", () => {
    const pages = normalizeSitePages([
      {
        key: "docs",
        title: "Docs",
        slug: "doc/docs",
        mode: "link",
        content: "https://blog.lumio.games/docs/doc/api",
        enabled: true,
      },
    ]);

    expect(resolveSitePageNavigationTarget(pages, "docs")).toEqual({
      kind: "external",
      target: "https://blog.lumio.games/docs/doc/api",
    });
    expect(resolveSitePageNavItem(pages, "docs")).toEqual({
      target: "https://blog.lumio.games/docs/doc/api",
      external: true,
    });
  });

  it("falls back to the slug route when a link page has no http URL", () => {
    const pages = normalizeSitePages([
      {
        key: "terms",
        title: "Terms",
        slug: "doc/服务条款",
        mode: "link",
        content: "not-a-url",
        enabled: true,
      },
    ]);

    expect(resolveSitePageNavigationTarget(pages, "terms")).toEqual({
      kind: "route",
      target: encodeURI("/doc/服务条款"),
    });
    expect(resolveSitePageNavItem(pages, "terms")).toEqual({
      target: encodeURI("/doc/服务条款"),
      external: false,
    });
  });

  it("does not resolve disabled pages", () => {
    expect(
      resolveSitePageNavigationTarget(
        [
          {
            key: "privacy",
            title: "Privacy",
            slug: "doc/隐私协议",
            mode: "link",
            content: "https://example.com/privacy",
            enabled: false,
          },
        ],
        "privacy",
      ),
    ).toBeNull();
    expect(resolveSitePageNavItem(null, "privacy")).toBeNull();
  });

  it("keeps the docs fallback URL when the site page is empty markdown", () => {
    const pages = normalizeSitePages([
      {
        key: "docs",
        title: "文档",
        slug: "doc/文档",
        mode: "markdown",
        content: "",
        enabled: true,
      },
    ]);

    expect(
      resolveDocsNavItem(pages, "https://fast-note.example.com/docs"),
    ).toEqual({
      target: "https://fast-note.example.com/docs",
      external: true,
    });
  });

  it("uses markdown docs content instead of the fallback URL", () => {
    const pages = normalizeSitePages([
      {
        key: "docs",
        title: "文档",
        slug: "doc/文档",
        mode: "markdown",
        content: "# 文档",
        enabled: true,
      },
    ]);

    expect(
      resolveDocsNavItem(pages, "https://fast-note.example.com/docs"),
    ).toEqual({
      target: encodeURI("/doc/文档"),
      external: false,
    });
  });
});
