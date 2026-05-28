package service

import (
	"strconv"
	"strings"
)

// SubscriptionCoversGroup reports whether a credit-pool subscription can be used
// for the request group. Status, expiry, and quota checks are handled separately.
func SubscriptionCoversGroup(sub *UserSubscription, group *Group, _ *User) bool {
	if sub == nil || !subscriptionScopeGroupAvailable(group) {
		return false
	}
	switch strings.TrimSpace(sub.ScopeType) {
	case "", SubscriptionScopeAllAvailableGroups:
		return true
	case SubscriptionScopeSelectedGroups:
		return int64SliceContains(scopeInt64Slice(sub.ScopeConfig["group_ids"]), group.ID)
	case SubscriptionScopePlatforms:
		return stringSliceContainsFold(scopeStringSlice(sub.ScopeConfig["platforms"]), strings.TrimSpace(group.Platform))
	default:
		return false
	}
}

func subscriptionScopeGroupAvailable(group *Group) bool {
	if group == nil || group.ID <= 0 {
		return false
	}
	return group.Status == "" || group.Status == StatusActive
}

func scopeInt64Slice(value any) []int64 {
	switch v := value.(type) {
	case []int64:
		return v
	case []int:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			out = append(out, int64(item))
		}
		return out
	case []float64:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			out = append(out, int64(item))
		}
		return out
	case []string:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			if parsed, err := strconv.ParseInt(strings.TrimSpace(item), 10, 64); err == nil {
				out = append(out, parsed)
			}
		}
		return out
	case []any:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			if parsed, ok := scopeInt64Value(item); ok {
				out = append(out, parsed)
			}
		}
		return out
	}
	if parsed, ok := scopeInt64Value(value); ok {
		return []int64{parsed}
	}
	return nil
}

func scopeInt64Value(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func scopeStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{v}
	}
	return nil
}

func int64SliceContains(values []int64, want int64) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func stringSliceContainsFold(values []string, want string) bool {
	if want == "" {
		return false
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), want) {
			return true
		}
	}
	return false
}
