# Model Pricing Data

This directory contains a local copy of the mirrored model pricing data as a fallback mechanism, plus an optional **local overrides** file for platform policy pricing.

## Local overrides (`pricing_overrides.json`)

Use this when you need to pin specific model base rates while still following the remote price table for everything else.

Priority (highest first):

1. Channel / group custom pricing (if configured)
2. **`pricing.overrides_file`** (this file by default)
3. Remote `model-price-repo` / local cache (`data/model_pricing.json`)
4. `fallback_file` (this directory's full JSON, only fills missing models)
5. Hardcoded Go fallbacks

Only models listed in `pricing_overrides.json` are overridden. Unlisted models keep remote auto-updates.

Configure path via:

```yaml
pricing:
  overrides_file: "./resources/model-pricing/pricing_overrides.json"
```

Default ships with `gpt-5.6-luna` pinned to pre-2026-07-31 rates, and Claude Fable 5 / 5.1 pinned to Anthropic official cache rates (`$12.50` 5m write / `$20` 1h write; Fable 5.1 cache read `$0.25`). Edit or empty the file to change policy; restart (or wait for next pricing reload after process restart) to apply.

## Source
The original file is maintained by the LiteLLM project and mirrored into the `price-mirror` branch of this repository via GitHub Actions:
- Mirror branch (configurable via `PRICE_MIRROR_REPO`): https://raw.githubusercontent.com/<your-repo>/price-mirror/model_prices_and_context_window.json
- Upstream source: https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json

## Purpose
This local copy serves as a fallback when the remote file cannot be downloaded due to:
- Network restrictions
- Firewall rules
- DNS resolution issues
- GitHub being blocked in certain regions
- Docker container network limitations

## Update Process
The pricingService will:
1. First attempt to download the latest version from GitHub
2. If download fails, use this local copy as fallback
3. Log a warning when using the fallback file

## Manual Update
To manually update this file with the latest pricing data (if automation is unavailable):
```bash
curl -s https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json -o model_prices_and_context_window.json
```

## File Format
The file contains JSON data with model pricing information including:
- Model names and identifiers
- Input/output token costs
- Context window sizes
- Model capabilities

Last updated: 2025-08-10
