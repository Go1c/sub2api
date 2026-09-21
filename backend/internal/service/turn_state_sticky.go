package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/gin-gonic/gin"
	"regexp"
	"strings"
	"time"
)

// Only replace supplier session parameters. Credentials stay in proxy records.
var stickySessionParameter = regexp.MustCompile(`-(?:sid|t)-[^-]+`)

func stickyProxy(p *Proxy, sid string, minutes int) *Proxy {
	cp := *p
	cp.Username = stickySessionParameter.ReplaceAllString(strings.TrimSpace(p.Username), "") + "-sid-" + sid + fmt.Sprintf("-t-%d", minutes)
	return &cp
}

func stickyProxyFingerprint(p *Proxy) string {
	sum := sha256.Sum256([]byte(p.URL()))
	return fmt.Sprintf("%x", sum[:])
}

func (r *TurnStateTicketRecord) copyStickyFrom(other TurnStateTicketRecord) {
	r.StickyGroupID = other.StickyGroupID
	r.StickyProxyID = other.StickyProxyID
	r.StickySession = other.StickySession
	r.StickyUntil = other.StickyUntil
	r.StickyExitExpiresAt = other.StickyExitExpiresAt
	r.StickyProxyFingerprint = other.StickyProxyFingerprint
	r.StickySessionMinutes = other.StickySessionMinutes
}

func (s *TurnStateProbeService) stickyGroup(ctx context.Context, account *Account) (*ProxyIPGroup, error) {
	if !accountUsesOpenAIIPGroup(account) || s.groups == nil {
		return nil, nil
	}
	group, err := s.groups.GetByID(ctx, *account.ProxyIPGroupID)
	if err != nil {
		return nil, err
	}
	if group.StickyMinutes == 0 {
		return nil, nil
	}
	return group, nil
}

func (s *TurnStateProbeService) newStickyExit(ctx context.Context, group *ProxyIPGroup, attempt int) (string, TurnStateTicketRecord, error) {
	var rec TurnStateTicketRecord
	if s.proxyRepo == nil {
		return "", rec, ErrTurnStateProbeInvalid
	}
	proxies, err := s.proxyRepo.ListByIDs(ctx, group.ProxyIDs)
	if err != nil {
		return "", rec, err
	}
	live := make([]Proxy, 0, len(proxies))
	for _, p := range proxies {
		if proxyIsLive(&p, time.Now()) && strings.TrimSpace(p.Username) != "" && strings.TrimSpace(p.Password) != "" {
			live = append(live, p)
		}
	}
	if len(live) == 0 {
		return "", rec, ErrTurnStateProbeInvalid
	}
	p := live[attempt%len(live)]
	sid, err := randomTurnStateProbeSID()
	if err != nil {
		return "", rec, err
	}
	minutes := group.StickyMinutes + 10
	rec = TurnStateTicketRecord{StickyGroupID: group.ID, StickyProxyID: p.ID, StickySession: sid, StickySessionMinutes: minutes, StickyExitExpiresAt: time.Now().Add(time.Duration(minutes) * time.Minute), StickyProxyFingerprint: stickyProxyFingerprint(&p)}
	return stickyProxy(&p, sid, minutes).URL(), rec, nil
}

type turnStateStickyLookup interface {
	StickyTicket(context.Context, *Account) (*TurnStateTicketRecord, error)
}

func (s *TurnStateProbeService) StickyTicket(ctx context.Context, account *Account) (*TurnStateTicketRecord, error) {
	ticket, ok := s.currentHoldingTicket(ctx, account)
	if !ok || ticket.StickyGroupID == 0 {
		return nil, errOpenAIIPGroupNoProxy
	}
	return ticket, nil
}

const stickySnapshotKey = "openai_sticky_ticket_snapshot"

type stickyRequestSnapshot struct {
	ticket   TurnStateTicketRecord
	proxyURL string
}

func stickySnapshot(c *gin.Context, account *Account) *stickyRequestSnapshot {
	if c == nil || account == nil {
		return nil
	}
	value, ok := c.Get(stickySnapshotKey)
	if !ok {
		return nil
	}
	snapshot, _ := value.(*stickyRequestSnapshot)
	if snapshot == nil || snapshot.ticket.AccountID != account.ID {
		return nil
	}
	return snapshot
}

func stickySnapshotError(c *gin.Context, account *Account) error {
	snapshot := stickySnapshot(c, account)
	if snapshot != nil && (!snapshot.ticket.ExpiresAt.After(time.Now()) || !snapshot.ticket.StickyExitExpiresAt.After(time.Now())) {
		return newOpenAIIPGroupNoProxyFailoverError()
	}
	return nil
}

func (s *OpenAIGatewayService) lookupStickyProxy(ctx context.Context, c *gin.Context, account *Account) (string, func(), bool, error) {
	noop := func() {}
	if s == nil || !accountUsesOpenAIIPGroup(account) || s.ipGroupResolver == nil || s.ipGroupResolver.groups == nil {
		return "", noop, false, nil
	}
	group, err := s.ipGroupResolver.groups.GetByID(ctx, *account.ProxyIPGroupID)
	if err != nil {
		return "", noop, true, err
	}
	if group.StickyMinutes == 0 {
		if c != nil {
			c.Set(stickySnapshotKey, nil)
		}
		return "", noop, false, nil
	}
	lookup, ok := s.turnStateTickets.(turnStateStickyLookup)
	if !ok || c == nil {
		return "", noop, true, newOpenAIIPGroupNoProxyFailoverError()
	}
	ticket, err := lookup.StickyTicket(ctx, account)
	if err != nil || ticket == nil || ticket.StickyGroupID != group.ID {
		return "", noop, true, newOpenAIIPGroupNoProxyFailoverError()
	}
	members, err := s.ipGroupResolver.loadLiveMembers(ctx, group)
	if err != nil {
		return "", noop, true, err
	}
	proxy := members[ticket.StickyProxyID]
	if proxy == nil || stickyProxyFingerprint(proxy) != ticket.StickyProxyFingerprint {
		return "", noop, true, newOpenAIIPGroupNoProxyFailoverError()
	}
	release, acquired, err := s.ipGroupResolver.tryOccupy(ctx, account.ID, proxy.ID, group.PerIPConcurrency, true)
	if err != nil {
		return "", noop, true, err
	}
	if !acquired {
		return "", noop, true, newOpenAIIPGroupNoProxyFailoverError()
	}
	url := stickyProxy(proxy, ticket.StickySession, ticket.StickySessionMinutes).URL()
	c.Set(stickySnapshotKey, &stickyRequestSnapshot{ticket: *ticket, proxyURL: url})
	RememberEgressProxyID(c, proxy.ID)
	return url, release, true, nil
}

func (s *OpenAIGatewayService) lookupOpenAIWSProxyURL(ctx context.Context, c *gin.Context, account *Account, sessionHash string) (string, func(), error) {
	if url, release, handled, err := s.lookupStickyProxy(ctx, c, account); handled {
		return url, release, err
	}
	return s.mustOpenAIAccountProxyURL(ctx, account, sessionHash)
}

// Configuration changes invalidate the whole pair, including the minimum hold.
func (s *TurnStateProbeService) stickyConfigurationMatches(ctx context.Context, account *Account, ticket *TurnStateTicketRecord) (bool, error) {
	if ticket == nil {
		return false, nil
	}
	group, err := s.stickyGroup(ctx, account)
	if err != nil {
		return false, err
	}
	if group == nil {
		return ticket.StickyGroupID == 0, nil
	}
	if ticket.StickyGroupID != group.ID || ticket.StickySessionMinutes != group.StickyMinutes+10 || s.proxyRepo == nil {
		return false, nil
	}
	member := false
	for _, id := range group.ProxyIDs {
		if id == ticket.StickyProxyID {
			member = true
			break
		}
	}
	if !member {
		return false, nil
	}
	proxies, err := s.proxyRepo.ListByIDs(ctx, []int64{ticket.StickyProxyID})
	if err != nil {
		return false, err
	}
	if len(proxies) != 1 {
		return false, nil
	}
	return proxyIsLive(&proxies[0], time.Now()) && stickyProxyFingerprint(&proxies[0]) == ticket.StickyProxyFingerprint, nil
}
