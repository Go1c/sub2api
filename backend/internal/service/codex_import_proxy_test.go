package service

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSelectCodexImportProxyPriority(t *testing.T) {
	now := time.Now()
	expired := now.Add(-time.Minute)
	proxies := []Proxy{{ID: 1, Status: StatusActive}, {ID: 2, Status: StatusActive, Username: "dynamic", Password: "secret"}, {ID: 3, Status: StatusActive, ExpiresAt: &expired}}
	groups := []ProxyIPGroup{{ID: 10, ProxyIDs: []int64{1}}, {ID: 20, StickyMinutes: 20, ProxyIDs: []int64{2}}}
	tests := []struct {
		name    string
		enabled bool
		groups  []ProxyIPGroup
		proxies []Proxy
		mode    string
		id      int64
	}{
		{"enabled dynamic before ordinary", true, groups, proxies, "dynamic", 20},
		{"disabled dynamic falls back", false, groups, proxies, "group", 10},
		{"no group falls back to single", true, nil, proxies, "single", 1},
		{"empty dynamic falls back", true, []ProxyIPGroup{{ID: 20, StickyMinutes: 20}, {ID: 10, ProxyIDs: []int64{1}}}, proxies, "group", 10},
		{"missing credentials falls back", true, []ProxyIPGroup{{ID: 20, StickyMinutes: 20, ProxyIDs: []int64{1}}}, proxies, "single", 1},
		{"expired only yields none", true, nil, proxies[2:], "none", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectCodexImportProxy(tt.groups, tt.proxies, tt.enabled, now)
			require.Equal(t, tt.mode, got.Mode)
			if got.ProxyIPGroupID != nil {
				require.Equal(t, tt.id, *got.ProxyIPGroupID)
			}
			if got.ProxyID != nil {
				require.Equal(t, tt.id, *got.ProxyID)
			}
			if tt.mode == "none" {
				require.Nil(t, got.ProxyID)
				require.Nil(t, got.ProxyIPGroupID)
			}
		})
	}
}

func TestSelectCodexImportProxy916Before921(t *testing.T) {
	groups := []ProxyIPGroup{{ID: 2, ProxyIDs: []int64{22, 23}}, {ID: 1, ProxyIDs: []int64{3, 7, 8, 13, 14, 18, 19, 20, 22, 23}}}
	result := SelectCodexImportProxy(groups, []Proxy{{ID: 22, Status: StatusActive}, {ID: 23, Status: StatusActive}}, true, time.Now())
	require.Equal(t, int64(1), *result.ProxyIPGroupID)
}
