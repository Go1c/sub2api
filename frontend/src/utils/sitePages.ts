import type { SitePage, SitePageMode } from "@/types";

export type NormalizedSitePage = SitePage & {
  mode: SitePageMode;
};

export interface SitePageNavigationTarget {
  kind: "route";
  target: string;
}

export function normalizeSitePageSlug(slug: string) {
  return slug.trim().replace(/^\/+|\/+$/g, "");
}

export function normalizeSitePageMode(mode: unknown): SitePageMode {
  return mode === "link" ? "link" : "markdown";
}

export function isHttpUrl(value: string) {
  try {
    const url = new URL(value.trim());
    return url.protocol === "http:" || url.protocol === "https:";
  } catch {
    return false;
  }
}

export function normalizeSitePages(
  pages: SitePage[] | null | undefined,
): NormalizedSitePage[] {
  if (!Array.isArray(pages)) return [];

  return pages.map((page) => {
    const mode = normalizeSitePageMode(page.mode);
    return {
      key: String(page.key || "").trim(),
      title: String(page.title || "").trim(),
      slug: normalizeSitePageSlug(String(page.slug || "")),
      mode,
      content:
        mode === "link"
          ? String(page.content || "").trim()
          : String(page.content || ""),
      enabled: page.enabled !== false,
    };
  });
}

export function resolveSitePageNavigationTarget(
  pages: SitePage[] | null | undefined,
  key: string,
): SitePageNavigationTarget | null {
  const page = normalizeSitePages(pages).find(
    (item) => item.enabled !== false && item.key === key,
  );
  if (!page) return null;

  const slug = normalizeSitePageSlug(page.slug);
  return slug ? { kind: "route", target: encodeURI(`/${slug}`) } : null;
}
