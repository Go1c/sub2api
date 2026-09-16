package service

import (
	"context"
	"log"
	"sort"
)

type AccountRequestHealthService struct {
	store RequestHealthStore
	dir   RequestHealthDirectory
}

func NewAccountRequestHealthService(store RequestHealthStore, dir RequestHealthDirectory) *AccountRequestHealthService {
	if store == nil {
		store = noopRequestHealthStore{}
	}
	return &AccountRequestHealthService{store: store, dir: dir}
}

func (s *AccountRequestHealthService) Record(ctx context.Context, in RequestHealthRecordInput) {
	if s == nil || s.store == nil || in.AccountID <= 0 {
		return
	}
	ev := in.toEvent()
	go func() {
		if err := s.store.Append(context.Background(), ev); err != nil {
			log.Printf("[request-health] append failed: %v", err)
		}
	}()
}

func (s *AccountRequestHealthService) recordSync(ctx context.Context, in RequestHealthRecordInput) error {
	if s == nil || s.store == nil || in.AccountID <= 0 {
		return nil
	}
	return s.store.Append(ctx, in.toEvent())
}

func (s *AccountRequestHealthService) ListForAccounts(ctx context.Context, accountIDs []int64, window int) ([]AccountRequestHealthDTO, error) {
	if s == nil {
		return []AccountRequestHealthDTO{}, nil
	}
	window = ClampRequestHealthWindow(window)
	ids := uniquePositiveIDs(accountIDs, requestHealthBatchLimit)
	if len(ids) == 0 {
		return []AccountRequestHealthDTO{}, nil
	}

	accounts := []*Account{}
	if s.dir != nil {
		found, err := s.dir.GetAccountsByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		accounts = found
	}
	accountByID := make(map[int64]*Account, len(accounts))
	for _, acc := range accounts {
		if acc != nil {
			accountByID[acc.ID] = acc
		}
	}

	groups, proxies := s.loadProxyDirectory(ctx)
	out := make([]AccountRequestHealthDTO, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.buildAccountHealth(ctx, accountByID[id], id, window, groups, proxies))
	}
	return out, nil
}

func (s *AccountRequestHealthService) loadProxyDirectory(ctx context.Context) (map[int64]ProxyIPGroup, map[int64]Proxy) {
	groups := map[int64]ProxyIPGroup{}
	proxies := map[int64]Proxy{}
	if s.dir == nil {
		return groups, proxies
	}
	listed, err := s.dir.ListProxyIPGroups(ctx)
	if err != nil {
		log.Printf("[request-health] list ip groups failed: %v", err)
		return groups, proxies
	}
	proxyIDs := make([]int64, 0)
	for _, group := range listed {
		groups[group.ID] = group
		proxyIDs = append(proxyIDs, group.ProxyIDs...)
	}
	if len(proxyIDs) == 0 {
		return groups, proxies
	}
	found, err := s.dir.GetProxiesByIDs(ctx, uniquePositiveIDs(proxyIDs, 0))
	if err != nil {
		log.Printf("[request-health] list proxies failed: %v", err)
		return groups, proxies
	}
	for _, proxy := range found {
		proxies[proxy.ID] = proxy
	}
	return groups, proxies
}

func (s *AccountRequestHealthService) buildAccountHealth(
	ctx context.Context,
	account *Account,
	accountID int64,
	window int,
	groups map[int64]ProxyIPGroup,
	proxies map[int64]Proxy,
) AccountRequestHealthDTO {
	dto := AccountRequestHealthDTO{
		AccountID: accountID,
		Mode:      RequestHealthModeSingle,
		Lines:     []AccountRequestHealthLineDTO{},
	}
	if account != nil && account.ProxyIPGroupID != nil {
		if group, ok := groups[*account.ProxyIPGroupID]; ok && len(group.ProxyIDs) > 0 {
			dto.Mode = RequestHealthModeIPGroup
			dto.IPGroupName = group.Name
			ids := append([]int64(nil), group.ProxyIDs...)
			sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
			for _, proxyID := range ids {
				proxy := proxies[proxyID]
				dto.Lines = append(dto.Lines, s.buildLine(ctx, accountID, proxyID, MaskRequestHealthHost(proxy.Host), window, group.PerIPConcurrency, false))
			}
			return dto
		}
	}

	proxyID := int64(0)
	ip := "—"
	max := 1
	if account != nil {
		if account.ProxyID != nil {
			proxyID = *account.ProxyID
		}
		if account.Proxy != nil {
			ip = MaskRequestHealthHost(account.Proxy.Host)
		} else if proxy, ok := proxies[proxyID]; ok {
			ip = MaskRequestHealthHost(proxy.Host)
		}
		if account.Concurrency > 0 {
			max = account.Concurrency
		}
	}
	dto.Lines = []AccountRequestHealthLineDTO{s.buildLine(ctx, accountID, proxyID, ip, window, max, true)}
	return dto
}

func (s *AccountRequestHealthService) buildLine(ctx context.Context, accountID, proxyID int64, ip string, window, max int, fallbackAccount bool) AccountRequestHealthLineDTO {
	if max <= 0 {
		max = 1
	}
	events, err := s.store.List(ctx, accountID, proxyID, window)
	if err != nil {
		log.Printf("[request-health] list events failed: %v", err)
		events = nil
	} else if fallbackAccount && proxyID > 0 && len(events) == 0 {
		if fallback, fallbackErr := s.store.List(ctx, accountID, 0, window); fallbackErr == nil {
			events = fallback
		}
	}
	current, cooldown := s.store.Runtime(ctx, accountID, proxyID)
	line := AccountRequestHealthLineDTO{
		ProxyID:       proxyID,
		IP:            ip,
		Outcomes:      make([]RequestHealthOutcomeDTO, 0, len(events)),
		Current:       current,
		Max:           max,
		CooldownUntil: cooldown,
	}
	for _, ev := range events {
		line.Outcomes = append(line.Outcomes, RequestHealthOutcomeDTO{
			Slot:       ev.Slot,
			StatusCode: ev.StatusCode,
			Message:    ev.Message,
			OccurredAt: ev.OccurredAt,
			Model:      ev.Model,
			Endpoint:   ev.Endpoint,
		})
	}
	if last := lastOutcome(line.Outcomes); last != nil && last.Slot == RequestHealthSlotFail {
		line.RateLimited = last.StatusCode == 429
		line.Overloaded = last.StatusCode == 529
	}
	return line
}

func lastOutcome(items []RequestHealthOutcomeDTO) *RequestHealthOutcomeDTO {
	if len(items) == 0 {
		return nil
	}
	return &items[len(items)-1]
}

func uniquePositiveIDs(ids []int64, limit int) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}
