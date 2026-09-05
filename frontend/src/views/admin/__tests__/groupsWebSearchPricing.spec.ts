import { describe, expect, it } from "vitest";

import {
  DEFAULT_WEB_SEARCH_PRICE_PER_CALL,
  resolveWebSearchPreviewMultiplier,
  supportsWebSearchPricingPlatform,
} from "../groupsWebSearchPricing";

describe("groups web search pricing platform support", () => {
  it("enables per-call search pricing for Codex and Grok groups", () => {
    expect(supportsWebSearchPricingPlatform("openai")).toBe(true);
    expect(supportsWebSearchPricingPlatform("grok")).toBe(true);
  });

  it("keeps other platforms out of the search pricing controls", () => {
    expect(supportsWebSearchPricingPlatform("anthropic")).toBe(false);
    expect(supportsWebSearchPricingPlatform("gemini")).toBe(false);
  });
});

describe("resolveWebSearchPreviewMultiplier", () => {
  it("uses the search multiplier when independent mode is on", () => {
    expect(resolveWebSearchPreviewMultiplier(true, 2.5, 1.5)).toBe(2.5);
  });

  it("uses the group multiplier when independent mode is off", () => {
    expect(resolveWebSearchPreviewMultiplier(false, 2.5, 1.5)).toBe(1.5);
  });

  it("falls back to 1 for empty or invalid independent multipliers", () => {
    expect(resolveWebSearchPreviewMultiplier(true, "", 3)).toBe(1);
    expect(resolveWebSearchPreviewMultiplier(true, null, 3)).toBe(1);
    expect(resolveWebSearchPreviewMultiplier(true, "nope", 3)).toBe(1);
  });

  it("previews the official $0.01 default times the independent multiplier", () => {
    expect(DEFAULT_WEB_SEARCH_PRICE_PER_CALL).toBe(0.01);
    expect(
      DEFAULT_WEB_SEARCH_PRICE_PER_CALL *
        resolveWebSearchPreviewMultiplier(true, 2, 9),
    ).toBeCloseTo(0.02);
  });
});
