//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// geoBlockSettingRepoStub is a minimal read/write SettingRepository for geo-block tests.
type geoBlockSettingRepoStub struct {
	values map[string]string
}

func newGeoBlockSettingRepoStub() *geoBlockSettingRepoStub {
	return &geoBlockSettingRepoStub{values: map[string]string{}}
}

func (s *geoBlockSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *geoBlockSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", ErrSettingNotFound
}

func (s *geoBlockSettingRepoStub) Set(ctx context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *geoBlockSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if v, ok := s.values[key]; ok {
			result[key] = v
		}
	}
	return result, nil
}

func (s *geoBlockSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	for k, v := range settings {
		s.values[k] = v
	}
	return nil
}

func (s *geoBlockSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for k, v := range s.values {
		out[k] = v
	}
	return out, nil
}

func (s *geoBlockSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

// resetGeoBlockCache clears the process-wide runtime cache so tests do not bleed into each other.
func resetGeoBlockCache() {
	geoBlockRuntimeSF.Forget(geoBlockRuntimeCacheKey)
	geoBlockRuntimeCache.Store((*cachedGeoBlockRuntime)(nil))
}

func TestGetGeoBlockRuntime_DefaultsWhenDBEmpty(t *testing.T) {
	resetGeoBlockCache()
	repo := newGeoBlockSettingRepoStub()
	// config seed empty -> built-in defaults
	svc := NewSettingService(repo, &config.Config{})

	rt := svc.GetGeoBlockRuntime(context.Background())
	require.False(t, rt.Enabled)
	require.Equal(t, []string{"CN"}, rt.Countries)
	require.Empty(t, rt.Whitelist)
}

func TestGetGeoBlockRuntime_SeedsFromConfigWhenDBMissing(t *testing.T) {
	resetGeoBlockCache()
	repo := newGeoBlockSettingRepoStub()
	svc := NewSettingService(repo, &config.Config{GeoBlock: config.GeoBlockConfig{
		Enabled:   true,
		Countries: []string{"US", "JP"},
		Whitelist: []string{"203.0.113.0/24"},
	}})

	rt := svc.GetGeoBlockRuntime(context.Background())
	require.True(t, rt.Enabled)
	require.Equal(t, []string{"US", "JP"}, rt.Countries)
	require.Equal(t, []string{"203.0.113.0/24"}, rt.Whitelist)
}

func TestUpdateGeoBlockRuntime_WriteThenReadConsistent(t *testing.T) {
	resetGeoBlockCache()
	repo := newGeoBlockSettingRepoStub()
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateGeoBlockRuntime(context.Background(), config.GeoBlockConfig{
		Enabled:   true,
		Countries: []string{" cn ", "us"},
		Whitelist: []string{"1.2.3.4", "10.0.0.0/8"},
	})
	require.NoError(t, err)

	// Persisted values must be normalized JSON / bool.
	require.Equal(t, "true", repo.values[SettingKeyGeoBlockEnabled])
	var countries []string
	require.NoError(t, json.Unmarshal([]byte(repo.values[SettingKeyGeoBlockCountries]), &countries))
	require.Equal(t, []string{"CN", "US"}, countries)
	var whitelist []string
	require.NoError(t, json.Unmarshal([]byte(repo.values[SettingKeyGeoBlockWhitelist]), &whitelist))
	require.Equal(t, []string{"1.2.3.4", "10.0.0.0/8"}, whitelist)

	// Reading back (cache was refreshed on update) must reflect the new values immediately.
	rt := svc.GetGeoBlockRuntime(context.Background())
	require.True(t, rt.Enabled)
	require.Equal(t, []string{"CN", "US"}, rt.Countries)
	require.Equal(t, []string{"1.2.3.4", "10.0.0.0/8"}, rt.Whitelist)
}

func TestUpdateGeoBlockRuntime_RejectsInvalidCountry(t *testing.T) {
	resetGeoBlockCache()
	repo := newGeoBlockSettingRepoStub()
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateGeoBlockRuntime(context.Background(), config.GeoBlockConfig{
		Enabled:   true,
		Countries: []string{"CHINA"}, // not 2-letter
	})
	require.Error(t, err)
	require.Empty(t, repo.values[SettingKeyGeoBlockCountries], "nothing should be persisted on validation error")
}

func TestUpdateGeoBlockRuntime_RejectsInvalidWhitelist(t *testing.T) {
	resetGeoBlockCache()
	repo := newGeoBlockSettingRepoStub()
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateGeoBlockRuntime(context.Background(), config.GeoBlockConfig{
		Enabled:   true,
		Countries: []string{"CN"},
		Whitelist: []string{"not-an-ip"},
	})
	require.Error(t, err)
	require.Empty(t, repo.values[SettingKeyGeoBlockWhitelist], "nothing should be persisted on validation error")
}
