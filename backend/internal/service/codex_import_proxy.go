package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

type CodexImportProxy struct {
	ProxyID        *int64 `json:"proxy_id"`
	ProxyIPGroupID *int64 `json:"proxy_ip_group_id"`
	Mode           string `json:"mode"`
}

// Shared by JSON, session JSON and 2FA import. Explicit bindings bypass defaults.
func SelectCodexImportProxy(groups []ProxyIPGroup, proxies []Proxy, dynamicEnabled bool, now time.Time) CodexImportProxy {
	groups = append([]ProxyIPGroup(nil), groups...)
	proxies = append([]Proxy(nil), proxies...)
	sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })
	sort.Slice(proxies, func(i, j int) bool { return proxies[i].ID < proxies[j].ID })
	live := make(map[int64]Proxy)
	for _, p := range proxies {
		if proxyIsLive(&p, now) {
			live[p.ID] = p
		}
	}
	for _, sticky := range []bool{true, false} {
		if sticky && !dynamicEnabled {
			continue
		}
		for _, g := range groups {
			if (g.StickyMinutes > 0) != sticky {
				continue
			}
			for _, id := range g.ProxyIDs {
				p, ok := live[id]
				if !ok {
					continue
				}
				if sticky && (strings.TrimSpace(p.Username) == "" || strings.TrimSpace(p.Password) == "") {
					continue
				}
				groupID := g.ID
				mode := "group"
				if sticky {
					mode = "dynamic"
				}
				return CodexImportProxy{ProxyIPGroupID: &groupID, Mode: mode}
			}
		}
	}
	for _, p := range proxies {
		if proxyIsLive(&p, now) {
			id := p.ID
			return CodexImportProxy{ProxyID: &id, Mode: "single"}
		}
	}
	return CodexImportProxy{Mode: "none"}
}

func (s *adminServiceImpl) CodexImportProxyDefault(ctx context.Context) (CodexImportProxy, error) {
	enabled := false
	if s.settingService != nil && s.settingService.settingRepo != nil {
		raw, err := s.settingService.settingRepo.GetValue(ctx, SettingKeyTurnStateProbePolicy)
		if err != nil && !errors.Is(err, ErrSettingNotFound) {
			return CodexImportProxy{}, err
		}
		if strings.TrimSpace(raw) != "" {
			policy, err := DecodeTurnStateProbePolicyJSON(raw)
			if err != nil {
				return CodexImportProxy{}, err
			}
			enabled = policy.Enabled
		}
	}
	groups, err := s.ListProxyIPGroups(ctx)
	if err != nil {
		return CodexImportProxy{}, err
	}
	proxies, err := s.GetAllProxies(ctx)
	if err != nil {
		return CodexImportProxy{}, err
	}
	return SelectCodexImportProxy(groups, proxies, enabled, time.Now()), nil
}

func ResolveCodexImportProxyDefault(ctx context.Context, admin AdminService) (CodexImportProxy, error) {
	if admin == nil {
		return CodexImportProxy{}, errors.New("import proxy defaults unavailable")
	}
	if provider, ok := admin.(interface {
		CodexImportProxyDefault(context.Context) (CodexImportProxy, error)
	}); ok {
		return provider.CodexImportProxyDefault(ctx)
	}
	// Compatibility for alternate AdminService implementations without the policy provider.
	groups, err := admin.ListProxyIPGroups(ctx)
	if err != nil {
		return CodexImportProxy{}, err
	}
	proxies, err := admin.GetAllProxies(ctx)
	if err != nil {
		return CodexImportProxy{}, err
	}
	return SelectCodexImportProxy(groups, proxies, false, time.Now()), nil
}
