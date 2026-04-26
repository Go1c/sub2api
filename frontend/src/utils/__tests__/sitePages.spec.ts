import { describe, expect, it } from "vitest";

import {
  resolveSitePageNavigationTarget,
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

  it("resolves link pages as internal document routes", () => {
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
      kind: "route",
      target: "/doc/docs",
    });
  });
});
