export const webSearchPricingPlatforms = new Set(["openai", "grok"]);

export const supportsWebSearchPricingPlatform = (platform: string): boolean =>
  webSearchPricingPlatforms.has(platform);

export const DEFAULT_WEB_SEARCH_PRICE_PER_CALL = 0.01;

const parsePreviewNumber = (value: number | string | null | undefined, fallback = 1): number => {
  if (value === null || value === undefined || value === "") {
    return fallback;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
};

export function resolveWebSearchPreviewMultiplier(
  independent: boolean,
  searchMultiplier: number | string | null | undefined,
  groupMultiplier: number | string | null | undefined,
): number {
  if (independent) {
    return parsePreviewNumber(searchMultiplier, 1);
  }
  return parsePreviewNumber(groupMultiplier, 1);
}
