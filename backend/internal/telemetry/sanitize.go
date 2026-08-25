package telemetry

import (
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	attributionIDRe = regexp.MustCompile(`^bc_[a-z0-9]{16,32}$`)
	errorCodeRe     = regexp.MustCompile(`^(AUTH|ACCOUNT|KEY|SERVICE)_[A-Z0-9_]{1,48}$`)
	touchRe         = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,31}$`)
)

var allowedClientSources = map[string]struct{}{
	"bestcodex_web":            {},
	"bestcodex_desktop_codex":  {},
	"bestcodex_desktop_claude": {},
	ClientSourceUnknown:        {},
}

var allowedAuthMethods = map[string]struct{}{
	"email": {},
	"2fa":   {},
}

var allowedPlatforms = map[string]struct{}{
	"mac_arm":   {},
	"mac_intel": {},
	"windows":   {},
}

var allowedDestinations = map[string]struct{}{
	"cdn":    {},
	"github": {},
}

func sanitizeProps(props map[string]string) sanitizedProps {
	out := sanitizedProps{ClientSource: ClientSourceUnknown}
	if len(props) == 0 {
		return out
	}
	if raw := strings.TrimSpace(props["client_source"]); raw != "" {
		if _, ok := allowedClientSources[raw]; ok {
			out.ClientSource = raw
		}
	}
	out.Route = sanitizeRoute(props["route"])
	if raw := strings.TrimSpace(props["auth_method"]); raw != "" {
		if _, ok := allowedAuthMethods[raw]; ok {
			out.AuthMethod = raw
		}
	}
	if raw := strings.TrimSpace(props["platform"]); raw != "" {
		if _, ok := allowedPlatforms[raw]; ok {
			out.Platform = raw
		}
	}
	if raw := strings.TrimSpace(props["destination"]); raw != "" {
		if _, ok := allowedDestinations[raw]; ok {
			out.Destination = raw
		}
	}
	if raw := strings.TrimSpace(props["error_code"]); raw != "" {
		if raw == "UNKNOWN" || errorCodeRe.MatchString(raw) {
			out.ErrorCode = raw
		}
	}
	if raw := strings.TrimSpace(props["attribution_id"]); attributionIDRe.MatchString(raw) {
		out.AttributionID = raw
	}
	out.FirstTouchSource = sanitizeTouch(props["first_touch_source"])
	out.FirstTouchMedium = sanitizeTouch(props["first_touch_medium"])
	out.FirstTouchCampaign = sanitizeTouch(props["first_touch_campaign"])
	out.LastTouchSource = sanitizeTouch(props["last_touch_source"])
	out.LastTouchMedium = sanitizeTouch(props["last_touch_medium"])
	out.LastTouchCampaign = sanitizeTouch(props["last_touch_campaign"])
	return out
}

func sanitizeTouch(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !touchRe.MatchString(raw) {
		return ""
	}
	return raw
}

func sanitizeRoute(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if i := strings.IndexByte(raw, '#'); i >= 0 {
		raw = raw[:i]
	}
	path := raw
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		path = parsed.Path
	} else if i := strings.IndexByte(raw, '?'); i >= 0 {
		path = raw[:i]
	}
	if strings.Contains(path, "..") {
		return ""
	}
	if len(path) > maxRouteLen {
		path = path[:maxRouteLen]
	}
	return path
}

func normalizeOccurredAt(ts int64, now time.Time) time.Time {
	now = now.UTC()
	if ts >= 1_000_000_000 && ts < 10_000_000_000 {
		ts *= 1000
	}
	if ts <= 0 {
		return now
	}
	occurred := time.UnixMilli(ts).UTC()
	if occurred.Before(now.Add(-7*24*time.Hour)) || occurred.After(now.Add(time.Hour)) {
		return now
	}
	return occurred
}

func firstTouchEmpty(props sanitizedProps) bool {
	return props.FirstTouchSource == "" && props.FirstTouchMedium == "" && props.FirstTouchCampaign == ""
}
