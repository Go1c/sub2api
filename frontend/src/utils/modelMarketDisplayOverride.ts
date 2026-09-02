import type { ModelMarketSelection } from "@/api/modelMarket";
import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_TOKEN,
  BILLING_MODE_VIDEO,
} from "@/constants/channel";

function isBillingMode(mode: unknown): boolean {
  return (
    mode === BILLING_MODE_TOKEN ||
    mode === BILLING_MODE_PER_REQUEST ||
    mode === BILLING_MODE_IMAGE ||
    mode === BILLING_MODE_VIDEO
  );
}

export function selectionHasDisplayOverride(selection: ModelMarketSelection): boolean {
  return isBillingMode(selection.billing_mode) || selection.pricing != null;
}

export function clearModelMarketSelectionDisplayOverride(
  selection: ModelMarketSelection,
): ModelMarketSelection {
  return {
    key: selection.key,
    platform: selection.platform,
    model: selection.model,
    enabled: selection.enabled,
    sort_order: selection.sort_order,
  };
}

export function restoreModelMarketCandidateSelection(
  selections: ModelMarketSelection[],
  key: string,
  autoSync: boolean,
): ModelMarketSelection[] {
  const index = selections.findIndex((selection) => selection.key === key);
  if (index < 0) return selections;
  if (autoSync) {
    return selections.filter((selection) => selection.key !== key);
  }
  const next = selections.slice();
  next[index] = clearModelMarketSelectionDisplayOverride(selections[index]);
  return next;
}
