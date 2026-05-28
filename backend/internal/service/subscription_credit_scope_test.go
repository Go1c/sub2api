package service

import "testing"

func TestSubscriptionCoversGroup(t *testing.T) {
	tests := []struct {
		name  string
		sub   *UserSubscription
		group *Group
		want  bool
	}{
		{
			name: "all available covers active group",
			sub: &UserSubscription{
				ScopeType:   SubscriptionScopeAllAvailableGroups,
				ScopeConfig: map[string]any{},
			},
			group: &Group{ID: 10, Status: StatusActive, Platform: PlatformAnthropic},
			want:  true,
		},
		{
			name: "selected groups accepts numeric json group id",
			sub: &UserSubscription{
				ScopeType:   SubscriptionScopeSelectedGroups,
				ScopeConfig: map[string]any{"group_ids": []any{float64(10), float64(11)}},
			},
			group: &Group{ID: 11, Status: StatusActive, Platform: PlatformOpenAI},
			want:  true,
		},
		{
			name: "selected groups rejects missing group id",
			sub: &UserSubscription{
				ScopeType:   SubscriptionScopeSelectedGroups,
				ScopeConfig: map[string]any{"group_ids": []any{float64(10)}},
			},
			group: &Group{ID: 12, Status: StatusActive, Platform: PlatformOpenAI},
			want:  false,
		},
		{
			name: "platforms accepts matching platform",
			sub: &UserSubscription{
				ScopeType:   SubscriptionScopePlatforms,
				ScopeConfig: map[string]any{"platforms": []any{PlatformAnthropic, PlatformGemini}},
			},
			group: &Group{ID: 12, Status: StatusActive, Platform: PlatformGemini},
			want:  true,
		},
		{
			name: "platforms rejects inactive group",
			sub: &UserSubscription{
				ScopeType:   SubscriptionScopePlatforms,
				ScopeConfig: map[string]any{"platforms": []any{PlatformGemini}},
			},
			group: &Group{ID: 12, Status: StatusDisabled, Platform: PlatformGemini},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SubscriptionCoversGroup(tt.sub, tt.group, nil)
			if got != tt.want {
				t.Fatalf("SubscriptionCoversGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}
