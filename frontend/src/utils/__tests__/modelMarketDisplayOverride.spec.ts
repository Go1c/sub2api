import { describe, expect, it } from "vitest";

import type { ModelMarketSelection } from "@/api/modelMarket";
import { BILLING_MODE_TOKEN } from "@/constants/channel";
import {
  restoreModelMarketCandidateSelection,
  selectionHasDisplayOverride,
} from "@/utils/modelMarketDisplayOverride";

function overlaySelection(): ModelMarketSelection {
  return {
    key: "anthropic:claude-fable-5-1",
    platform: "anthropic",
    model: "claude-fable-5-1",
    enabled: true,
    sort_order: 10,
    billing_mode: BILLING_MODE_TOKEN,
    pricing: {
      billing_mode: BILLING_MODE_TOKEN,
      input_price: null,
      output_price: null,
      cache_write_price: 3.75e-6,
      cache_read_price: 0.3e-6,
      image_output_price: null,
      per_request_price: null,
      intervals: [],
    },
  };
}

describe("modelMarketDisplayOverride", () => {
  it("treats cache-only token snapshots as a display override", () => {
    expect(selectionHasDisplayOverride(overlaySelection())).toBe(true);
  });

  it("drops the auto-sync selection so live GetModelPricing is used again", () => {
    const kept: ModelMarketSelection = {
      key: "anthropic:claude-fable-5",
      platform: "anthropic",
      model: "claude-fable-5",
      enabled: true,
      sort_order: 0,
      billing_mode: BILLING_MODE_TOKEN,
      pricing: {
        billing_mode: BILLING_MODE_TOKEN,
        input_price: 10e-6,
        output_price: 50e-6,
        cache_write_price: 12.5e-6,
        cache_read_price: 1e-6,
        image_output_price: null,
        per_request_price: null,
        intervals: [],
      },
    };
    const restored = restoreModelMarketCandidateSelection(
      [kept, overlaySelection()],
      "anthropic:claude-fable-5-1",
      true,
    );
    expect(restored.map((item) => item.key)).toEqual(["anthropic:claude-fable-5"]);
    expect(restored.some((item) => item.key === "anthropic:claude-fable-5-1")).toBe(false);
  });

  it("keeps the manual-mode row but strips billing_mode and pricing", () => {
    const restored = restoreModelMarketCandidateSelection(
      [overlaySelection()],
      "anthropic:claude-fable-5-1",
      false,
    );
    expect(restored).toHaveLength(1);
    expect(restored[0]).toEqual({
      key: "anthropic:claude-fable-5-1",
      platform: "anthropic",
      model: "claude-fable-5-1",
      enabled: true,
      sort_order: 10,
    });
    expect(selectionHasDisplayOverride(restored[0])).toBe(false);
  });
});
