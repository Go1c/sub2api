package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"golang.org/x/sync/singleflight"
)

// Geo-block runtime cache: in-process atomic.Value with TTL + singleflight, mirroring
// the other runtime setting getters (e.g. version bounds / balance usage gate).
// Refreshed eagerly by UpdateGeoBlockRuntime so admin saves take effect immediately.
type cachedGeoBlockRuntime struct {
	value     config.GeoBlockConfig
	expiresAt int64 // unix nano
}

var (
	geoBlockRuntimeCache atomic.Value // *cachedGeoBlockRuntime
	geoBlockRuntimeSF    singleflight.Group
)

const (
	geoBlockRuntimeCacheKey  = "geo_block_runtime"
	geoBlockRuntimeCacheTTL  = 60 * time.Second
	geoBlockRuntimeErrorTTL  = 5 * time.Second
	geoBlockRuntimeDBTimeout = 5 * time.Second
)

// GetGeoBlockRuntime returns the effective geo-block configuration, reading from the
// settings store with a short TTL cache. Resolution order per field:
//
//	DB value -> cfg.GeoBlock seed -> built-in default (enabled=false, countries=["CN"], whitelist=empty).
//
// Fail-open: on DB error it falls back to the config seed (then built-in defaults) and
// caches that briefly so the hot path never blocks.
func (s *SettingService) GetGeoBlockRuntime(ctx context.Context) config.GeoBlockConfig {
	if cached, ok := geoBlockRuntimeCache.Load().(*cachedGeoBlockRuntime); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.value
		}
	}

	result, _, _ := geoBlockRuntimeSF.Do(geoBlockRuntimeCacheKey, func() (any, error) {
		// Double-check after acquiring singleflight slot.
		if cached, ok := geoBlockRuntimeCache.Load().(*cachedGeoBlockRuntime); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return cached.value, nil
			}
		}

		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), geoBlockRuntimeDBTimeout)
		defer cancel()

		vals, err := s.settingRepo.GetMultiple(dbCtx, []string{
			SettingKeyGeoBlockEnabled,
			SettingKeyGeoBlockCountries,
			SettingKeyGeoBlockWhitelist,
		})
		if err != nil {
			slog.Warn("failed to read geo_block runtime settings, falling back to config seed", "error", err)
			rt := s.geoBlockSeed()
			geoBlockRuntimeCache.Store(&cachedGeoBlockRuntime{
				value:     rt,
				expiresAt: time.Now().Add(geoBlockRuntimeErrorTTL).UnixNano(),
			})
			return rt, nil
		}

		rt := s.resolveGeoBlockRuntime(vals)
		geoBlockRuntimeCache.Store(&cachedGeoBlockRuntime{
			value:     rt,
			expiresAt: time.Now().Add(geoBlockRuntimeCacheTTL).UnixNano(),
		})
		return rt, nil
	})

	rt, ok := result.(config.GeoBlockConfig)
	if !ok {
		return s.geoBlockSeed()
	}
	return rt
}

// UpdateGeoBlockRuntime normalizes, validates and persists the three geo-block keys,
// then refreshes the runtime cache so the change takes effect immediately.
func (s *SettingService) UpdateGeoBlockRuntime(ctx context.Context, cfg config.GeoBlockConfig) error {
	countries, err := normalizeGeoBlockCountries(cfg.Countries)
	if err != nil {
		return err
	}
	whitelist, err := normalizeGeoBlockWhitelist(cfg.Whitelist)
	if err != nil {
		return err
	}

	countriesJSON, err := json.Marshal(countries)
	if err != nil {
		return fmt.Errorf("marshal geo_block countries: %w", err)
	}
	whitelistJSON, err := json.Marshal(whitelist)
	if err != nil {
		return fmt.Errorf("marshal geo_block whitelist: %w", err)
	}

	updates := map[string]string{
		SettingKeyGeoBlockEnabled:   strconv.FormatBool(cfg.Enabled),
		SettingKeyGeoBlockCountries: string(countriesJSON),
		SettingKeyGeoBlockWhitelist: string(whitelistJSON),
	}
	if err := s.settingRepo.SetMultiple(ctx, updates); err != nil {
		return fmt.Errorf("persist geo_block settings: %w", err)
	}

	refreshGeoBlockRuntimeCache(config.GeoBlockConfig{
		Enabled:   cfg.Enabled,
		Countries: countries,
		Whitelist: whitelist,
	})
	return nil
}

// geoBlockSeed returns the config seed normalized to built-in defaults.
func (s *SettingService) geoBlockSeed() config.GeoBlockConfig {
	var seed config.GeoBlockConfig
	if s != nil && s.cfg != nil {
		seed = s.cfg.GeoBlock
	}
	countries := parseGeoBlockCountries("")
	if c, err := normalizeGeoBlockCountries(seed.Countries); err == nil && len(c) > 0 {
		countries = c
	}
	whitelist := []string{}
	if w, err := normalizeGeoBlockWhitelist(seed.Whitelist); err == nil && len(w) > 0 {
		whitelist = w
	}
	return config.GeoBlockConfig{
		Enabled:   seed.Enabled,
		Countries: countries,
		Whitelist: whitelist,
	}
}

// resolveGeoBlockRuntime merges DB values over the config seed over built-in defaults.
// A key absent from the map (never written) falls back to the seed; a present-but-empty
// value is treated as an explicit reset to the built-in default for that field.
func (s *SettingService) resolveGeoBlockRuntime(vals map[string]string) config.GeoBlockConfig {
	seed := s.geoBlockSeed()
	rt := seed

	if raw, ok := vals[SettingKeyGeoBlockEnabled]; ok {
		rt.Enabled = raw == "true"
	}
	if raw, ok := vals[SettingKeyGeoBlockCountries]; ok {
		rt.Countries = parseGeoBlockCountries(raw)
	}
	if raw, ok := vals[SettingKeyGeoBlockWhitelist]; ok {
		rt.Whitelist = parseGeoBlockWhitelist(raw)
	}
	return rt
}

// refreshGeoBlockRuntimeCache invalidates the singleflight slot then stores the fresh value.
func refreshGeoBlockRuntimeCache(rt config.GeoBlockConfig) {
	geoBlockRuntimeSF.Forget(geoBlockRuntimeCacheKey)
	geoBlockRuntimeCache.Store(&cachedGeoBlockRuntime{
		value:     rt,
		expiresAt: time.Now().Add(geoBlockRuntimeCacheTTL).UnixNano(),
	})
}

// normalizeGeoBlockCountries trims/uppercases entries, drops blanks, and rejects any
// item that is not a 2-letter A-Z code. Empty input normalizes to the built-in ["CN"].
func normalizeGeoBlockCountries(raw []string) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, c := range raw {
		c = strings.ToUpper(strings.TrimSpace(c))
		if c == "" {
			continue
		}
		if !isISOAlpha2(c) {
			return nil, fmt.Errorf("invalid country code: %q (expected ISO 3166-1 alpha-2)", c)
		}
		if _, dup := seen[c]; dup {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	if len(out) == 0 {
		return []string{"CN"}, nil
	}
	return out, nil
}

// normalizeGeoBlockWhitelist trims entries, drops blanks, and rejects any item that is
// neither a valid IP nor a valid CIDR. Empty input normalizes to an empty slice.
func normalizeGeoBlockWhitelist(raw []string) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, w := range raw {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		if net.ParseIP(w) == nil {
			if _, _, err := net.ParseCIDR(w); err != nil {
				return nil, fmt.Errorf("invalid whitelist entry: %q (expected IP or CIDR)", w)
			}
		}
		if _, dup := seen[w]; dup {
			continue
		}
		seen[w] = struct{}{}
		out = append(out, w)
	}
	if len(out) == 0 {
		return []string{}, nil
	}
	return out, nil
}

// parseGeoBlockCountries parses persisted JSON into normalized codes; invalid entries are
// dropped (tolerant read) and an empty/invalid result falls back to ["CN"].
func parseGeoBlockCountries(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{"CN"}
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{"CN"}
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, c := range items {
		c = strings.ToUpper(strings.TrimSpace(c))
		if c == "" || !isISOAlpha2(c) {
			continue
		}
		if _, dup := seen[c]; dup {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	if len(out) == 0 {
		return []string{"CN"}
	}
	return out
}

// parseGeoBlockWhitelist parses persisted JSON into normalized entries; invalid entries are
// dropped (tolerant read) and an empty/invalid result yields an empty slice.
func parseGeoBlockWhitelist(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, w := range items {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		if net.ParseIP(w) == nil {
			if _, _, err := net.ParseCIDR(w); err != nil {
				continue
			}
		}
		if _, dup := seen[w]; dup {
			continue
		}
		seen[w] = struct{}{}
		out = append(out, w)
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func isISOAlpha2(s string) bool {
	if len(s) != 2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 'A' || s[i] > 'Z' {
			return false
		}
	}
	return true
}
