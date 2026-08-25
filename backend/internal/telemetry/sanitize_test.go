//go:build unit

package telemetry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSanitizePropsDropsSecretsUnknownKeysAndFullURLs(t *testing.T) {
	got := sanitizeProps(map[string]string{
		"email":                "leak@example.com",
		"password":             "s3cret",
		"token":                "tok_abc",
		"user_id":              "99",
		"url":                  "https://bestcodex.app/signup?next=1",
		"invite":               "INVITE99",
		"fingerprint":          "fp-device",
		"code":                 "123456",
		"unknown_key":          "nope",
		"client_source":        "bestcodex_web",
		"route":                "https://bestcodex.app/signup?utm=1#hash",
		"auth_method":          "email",
		"platform":             "mac_arm",
		"destination":          "cdn",
		"error_code":           "AUTH_INVALID",
		"attribution_id":       "bc_abcdefghijklmnop",
		"first_touch_source":   "sb.sb",
		"first_touch_medium":   "referral",
		"first_touch_campaign": "t1299",
		"last_touch_source":    "direct",
		"last_touch_medium":    "none",
		"last_touch_campaign":  "t1299",
		"bad_attribution_id":   "not-an-id",
	})

	require.Equal(t, "bestcodex_web", got.ClientSource)
	require.Equal(t, "/signup", got.Route)
	require.Equal(t, "email", got.AuthMethod)
	require.Equal(t, "mac_arm", got.Platform)
	require.Equal(t, "cdn", got.Destination)
	require.Equal(t, "AUTH_INVALID", got.ErrorCode)
	require.Equal(t, "bc_abcdefghijklmnop", got.AttributionID)
	require.Equal(t, "sb.sb", got.FirstTouchSource)
	require.Equal(t, "referral", got.FirstTouchMedium)
	require.Equal(t, "t1299", got.FirstTouchCampaign)
	require.Equal(t, "direct", got.LastTouchSource)
	require.Equal(t, "none", got.LastTouchMedium)
	require.Equal(t, "t1299", got.LastTouchCampaign)
}

func TestSanitizePropsAcceptsKnownTouchValuesAndDropsBadIDs(t *testing.T) {
	got := sanitizeProps(map[string]string{
		"client_source":        "not-a-client",
		"auth_method":          "sms",
		"platform":             "android",
		"destination":          "dropbox",
		"error_code":           "WAT_NO",
		"attribution_id":       "bc_SHORT",
		"first_touch_source":   "sb.sb",
		"first_touch_medium":   "referral",
		"first_touch_campaign": "t1299",
		"last_touch_source":    "Invalid Source",
		"route":                "/login/../admin",
	})

	require.Equal(t, "unknown", got.ClientSource)
	require.Empty(t, got.AuthMethod)
	require.Empty(t, got.Platform)
	require.Empty(t, got.Destination)
	require.Empty(t, got.ErrorCode)
	require.Empty(t, got.AttributionID)
	require.Equal(t, "sb.sb", got.FirstTouchSource)
	require.Equal(t, "referral", got.FirstTouchMedium)
	require.Equal(t, "t1299", got.FirstTouchCampaign)
	require.Empty(t, got.LastTouchSource)
	require.Empty(t, got.Route)
}

func TestNormalizeOccurredAt(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

	require.Equal(t, now, normalizeOccurredAt(0, now))
	require.Equal(t, now, normalizeOccurredAt(now.Add(-8*24*time.Hour).UnixMilli(), now))
	require.Equal(t, now, normalizeOccurredAt(now.Add(2*time.Hour).UnixMilli(), now))

	seconds := now.Unix()
	require.Equal(t, now.UnixMilli(), normalizeOccurredAt(seconds, now).UnixMilli())

	fresh := now.Add(-time.Minute).UnixMilli()
	require.Equal(t, fresh, normalizeOccurredAt(fresh, now).UnixMilli())
}
