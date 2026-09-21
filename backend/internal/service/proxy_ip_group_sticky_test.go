//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestStickyIPGroupConfiguration(t *testing.T) {
	for _, minutes := range []int{0, 20, 30, 50} {
		t.Run(strconv.Itoa(minutes), func(t *testing.T) {
			groups := newProxyIPGroupRepoStub()
			svc := &adminServiceImpl{proxyIPGroupRepo: groups, proxyRepo: &proxyListByIDsStub{}}
			input := CreateProxyIPGroupInput{Name: "sticky"}
			raw, _ := json.Marshal(map[string]any{"StickyMinutes": minutes})
			require.NoError(t, json.Unmarshal(raw, &input))
			group, err := svc.CreateProxyIPGroup(context.Background(), &input)
			require.NoError(t, err)
			encoded, err := json.Marshal(group)
			require.NoError(t, err)
			var fields map[string]any
			require.NoError(t, json.Unmarshal(encoded, &fields))
			require.Equal(t, float64(minutes), fields["StickyMinutes"])
		})
	}
}

func TestStickyIPGroupRejectsInvalidDuration(t *testing.T) {
	for _, minutes := range []int{-1, 1, 19, 51, 120} {
		groups := newProxyIPGroupRepoStub()
		svc := &adminServiceImpl{proxyIPGroupRepo: groups, proxyRepo: &proxyListByIDsStub{}}
		input := CreateProxyIPGroupInput{Name: "sticky"}
		raw, _ := json.Marshal(map[string]any{"StickyMinutes": minutes})
		require.NoError(t, json.Unmarshal(raw, &input))
		_, err := svc.CreateProxyIPGroup(context.Background(), &input)
		require.Error(t, err, "duration %d", minutes)
	}
}

func TestStickyTicketCannotFollowAccountIntoAnotherGroup(t *testing.T) {
	account := newOAuthProbeAccount(8, true)
	groupID := int64(99)
	account.ProxyIPGroupID = &groupID
	tickets := newMemoryTurnStateTicketStore()
	svc := newTurnStateProbeServiceForTest(t, account, tickets, nil)
	policy := DefaultTurnStateProbePolicy()
	policy.Enabled = true
	saved, err := svc.SavePolicy(context.Background(), policy)
	require.NoError(t, err)
	rec := TurnStateTicketRecord{AccountID: account.ID, State: "old-ticket", Status: "holding", PolicyRevision: saved.Revision, UpdatedAt: time.Now()}
	require.NoError(t, json.Unmarshal([]byte(`{"sticky_group_id":7,"sticky_proxy_id":1,"sticky_session":"old"}`), &rec))
	require.NoError(t, tickets.Put(context.Background(), rec))
	_, ok := svc.BindCurrent(context.Background(), account, "", "session", "")
	require.False(t, ok, "a ticket harvested for another group must not be injected")
}

func TestStickyTicketRenewalHonorsMinimumHoldingTime(t *testing.T) {
	account := newOAuthProbeAccount(8, true)
	tickets := newMemoryTurnStateTicketStore()
	svc := newTurnStateProbeServiceForTest(t, account, tickets, nil)
	policy := DefaultTurnStateProbePolicy()
	policy.Enabled = true
	saved, err := svc.SavePolicy(context.Background(), policy)
	require.NoError(t, err)
	now := time.Now()
	rec := TurnStateTicketRecord{AccountID: account.ID, State: "old-ticket", Status: "holding", PolicyRevision: saved.Revision, HarvestedAt: now.Add(-21 * time.Minute), ExpiresAt: now.Add(10 * time.Minute), RecheckAt: now.Add(9 * time.Minute)}
	raw, _ := json.Marshal(map[string]any{"sticky_group_id": 7, "sticky_until": now.Add(9 * time.Minute)})
	require.NoError(t, json.Unmarshal(raw, &rec))
	require.NoError(t, tickets.Put(context.Background(), rec))
	require.False(t, svc.ticketDue(context.Background(), account.ID, saved, now))
}

func TestStickySnapshotRoutesAndInjectsSameGeneration(t *testing.T) {
	ctx := context.Background()
	groups := newProxyIPGroupRepoStub()
	group := &ProxyIPGroup{Name: "sticky", StickyMinutes: 20, PerIPConcurrency: 1, ProxyIDs: []int64{3}}
	require.NoError(t, groups.Create(ctx, group))
	account := newOAuthProbeAccount(8, true)
	account.ProxyIPGroupID = &group.ID
	proxy := Proxy{ID: 3, Host: "proxy.example", Port: 2000, Protocol: "http", Username: "user-sid-old-t-5", Password: "secret", Status: StatusActive}
	proxies := &proxyListByIDsStub{proxies: map[int64]Proxy{3: proxy}}
	resolver := newOpenAIIPGroupResolver(groups, proxies, nil, nil, time.Hour)
	tickets := newMemoryTurnStateTicketStore()
	svc := newTurnStateProbeServiceForTest(t, account, tickets, nil)
	svc.groups = groups
	svc.proxyRepo = proxies
	policy := DefaultTurnStateProbePolicy()
	policy.Enabled = true
	saved, err := svc.SavePolicy(ctx, policy)
	require.NoError(t, err)
	rec := TurnStateTicketRecord{AccountID: 8, Identity: turnStateProbeTicketIdentity(account), State: "ticket-A", Status: "holding", PolicyRevision: saved.Revision, HarvestedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour), StickyGroupID: group.ID, StickyProxyID: 3, StickySession: "sessionA", StickySessionMinutes: 30, StickyExitExpiresAt: time.Now().Add(30 * time.Minute), StickyProxyFingerprint: stickyProxyFingerprint(&proxy)}
	require.NoError(t, tickets.Put(ctx, rec))
	gateway := &OpenAIGatewayService{ipGroupResolver: resolver, turnStateTickets: svc}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	route, release, err := gateway.lookupOpenAIProxyURL(ctx, c, account, []byte(`{"model":"gpt-6-astra"}`))
	require.NoError(t, err)
	defer release()
	require.Contains(t, route, "sid-sessionA-t-30")
	rec.State = "ticket-B"
	rec.StickySession = "sessionB"
	require.NoError(t, tickets.Put(ctx, rec))
	headers := http.Header{}
	gateway.applyTurnStateProbeHTTP(c, account, headers, []byte(`{}`), "gpt-6-astra")
	require.Equal(t, "ticket-A", headers.Get("X-Codex-Turn-State"))
}

func TestStickyLookupRewritesUdealSessionFromTicket(t *testing.T) {
	ctx := context.Background()
	groups := newProxyIPGroupRepoStub()
	group := &ProxyIPGroup{Name: "udeal", StickyMinutes: 20, PerIPConcurrency: 1, ProxyIDs: []int64{3}}
	require.NoError(t, groups.Create(ctx, group))
	account := newOAuthProbeAccount(8, true)
	account.ProxyIPGroupID = &group.ID
	proxy := Proxy{ID: 3, Host: "us.udealproxy.com", Port: 6666, Protocol: "http", Username: "userId-7155-custom-19653-region-us-session-ZHACt-sessTime-20", Password: "secret", Status: StatusActive}
	proxies := &proxyListByIDsStub{proxies: map[int64]Proxy{3: proxy}}
	resolver := newOpenAIIPGroupResolver(groups, proxies, nil, nil, time.Hour)
	tickets := newMemoryTurnStateTicketStore()
	svc := newTurnStateProbeServiceForTest(t, account, tickets, nil)
	svc.groups = groups
	svc.proxyRepo = proxies
	policy := DefaultTurnStateProbePolicy()
	policy.Enabled = true
	saved, err := svc.SavePolicy(ctx, policy)
	require.NoError(t, err)
	rec := TurnStateTicketRecord{AccountID: 8, Identity: turnStateProbeTicketIdentity(account), State: "ticket-A", Status: "holding", PolicyRevision: saved.Revision, HarvestedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour), StickyGroupID: group.ID, StickyProxyID: 3, StickySession: "abc12xyz", StickySessionMinutes: 30, StickyExitExpiresAt: time.Now().Add(30 * time.Minute), StickyProxyFingerprint: stickyProxyFingerprint(&proxy)}
	require.NoError(t, tickets.Put(ctx, rec))
	gateway := &OpenAIGatewayService{ipGroupResolver: resolver, turnStateTickets: svc}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	route, release, err := gateway.lookupOpenAIProxyURL(ctx, c, account, []byte(`{"model":"gpt-6-astra"}`))
	require.NoError(t, err)
	defer release()
	username := parsedUsername(t, route)
	require.Equal(t, "userId-7155-custom-19653-region-us-session-abc12xyz-sessTime-30", username)
	require.False(t, hasSidTParameters(username))
}

func TestStickyHarvestUsesGroupMemberAndPreservesPairOnFailure(t *testing.T) {
	ctx := context.Background()
	groups := newProxyIPGroupRepoStub()
	group := &ProxyIPGroup{Name: "sticky", StickyMinutes: 20, ProxyIDs: []int64{3}}
	require.NoError(t, groups.Create(ctx, group))
	account := newOAuthProbeAccount(8, true)
	account.ProxyIPGroupID = &group.ID
	proxy := Proxy{ID: 3, Host: "member.example", Port: 2000, Protocol: "http", Username: "user-region-US-sid-old-t-5", Password: "secret", Status: StatusActive}
	upstream := &recordingTurnStateHTTPUpstream{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Codex-Turn-State", strings.Repeat("a", 200))
		_, _ = io.WriteString(w, "data: {\"type\":\"response.created\",\"response\":{\"model\":\"gpt-6-astra\"}}\ndata: {\"type\":\"response.output_text.done\",\"text\":\"21\"}\n")
	})}
	tickets := newMemoryTurnStateTicketStore()
	svc := newTurnStateProbeServiceForTest(t, account, tickets, upstream)
	svc.groups = groups
	svc.proxyRepo = &proxyListByIDsStub{proxies: map[int64]Proxy{3: proxy}}
	policy := DefaultTurnStateProbePolicy()
	policy.Enabled = true
	_, err := svc.SavePolicy(ctx, policy)
	require.NoError(t, err)
	require.NoError(t, svc.ProbeAccount(ctx, account.ID))
	rec, err := tickets.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, group.ID, rec.StickyGroupID)
	require.Contains(t, upstream.lastProxy, "member.example")
	require.Contains(t, upstream.lastProxy, "-t-30")
	require.NotContains(t, upstream.lastProxy, "-sid-old")
	require.Equal(t, 1, strings.Count(upstream.lastProxy, "-sid-"))
	require.WithinDuration(t, time.Now().Add(20*time.Minute), rec.StickyUntil, 2*time.Second)
	require.NoError(t, svc.RunOne(ctx, account.ID))
	same, _ := tickets.Get(ctx, account.ID)
	require.Equal(t, rec.StickySession, same.StickySession)
	rec.StickyUntil = time.Now().Add(-time.Second)
	rec.RecheckAt = rec.StickyUntil
	require.NoError(t, tickets.Put(ctx, *rec))
	upstream.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(502) })
	require.Error(t, svc.ProbeAccount(ctx, account.ID))
	failed, _ := tickets.Get(ctx, account.ID)
	require.Equal(t, rec.State, failed.State)
	require.Equal(t, rec.StickySession, failed.StickySession)
	require.Equal(t, rec.StickyExitExpiresAt, failed.StickyExitExpiresAt)
	t.Run("configuration read failure retains original binding", func(t *testing.T) {
		previous := svc.proxyRepo
		svc.proxyRepo = &stickyFailingProxyRepo{}
		before, _ := tickets.Get(ctx, account.ID)
		require.Error(t, svc.RunOne(ctx, account.ID))
		after, _ := tickets.Get(ctx, account.ID)
		require.Equal(t, before, after)
		svc.proxyRepo = previous
	})
	t.Run("switching to ordinary group invalidates sticky ticket", func(t *testing.T) {
		group.StickyMinutes = 0
		require.NoError(t, groups.Update(ctx, group))
		_, ok := svc.BindCurrent(ctx, account, "", "", "")
		require.False(t, ok)
	})
	t.Run("proxy credential change bypasses minimum hold", func(t *testing.T) {
		group.StickyMinutes = 20
		require.NoError(t, groups.Update(ctx, group))
		rec.StickyUntil = time.Now().Add(20 * time.Minute)
		rec.Status = "holding"
		rec.RecheckAt = rec.StickyUntil
		require.NoError(t, tickets.Put(ctx, *rec))
		proxy.Password = "new-secret"
		svc.proxyRepo = &proxyListByIDsStub{proxies: map[int64]Proxy{3: proxy}}
		beforeProxy := upstream.lastProxy
		require.Error(t, svc.RunOne(ctx, account.ID), "must attempt harvest; synthetic upstream returns502")
		require.NotEqual(t, beforeProxy, upstream.lastProxy)
		require.Contains(t, upstream.lastProxy, "new-secret")
	})
}

func TestStickyConnectionCannotInjectAnotherGenerationOrSendAfterExpiry(t *testing.T) {
	account := newOAuthProbeAccount(8, true)
	gateway := &OpenAIGatewayService{turnStateTickets: injectTurnStateTicketLookup("new-ticket")}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	snapshot := &stickyRequestSnapshot{ticket: TurnStateTicketRecord{AccountID: 8, State: "old-ticket", ExpiresAt: time.Now().Add(time.Minute), StickyExitExpiresAt: time.Now().Add(time.Minute)}}
	c.Set(stickySnapshotKey, snapshot)
	payload := gateway.applyTurnStateProbeWSFrame(c, account, []byte(`{"type":"response.create"}`), "gpt-6-astra")
	require.Contains(t, string(payload), "old-ticket")
	require.NotContains(t, string(payload), "new-ticket")
	snapshot.ticket.StickyExitExpiresAt = time.Now().Add(-time.Second)
	require.Error(t, stickySnapshotError(c, account))
}

func TestStickyWSPoolSeparatesProxySessions(t *testing.T) {
	a := openAIWSAcquireRequest{ProxyURL: "http://user-sid-A-t-30:secret@proxy:2000"}
	b := a
	b.ProxyURL = "http://user-sid-B-t-30:secret@proxy:2000"
	require.NotEqual(t, openAIWSRequestCompatibility(a), openAIWSRequestCompatibility(b))
	require.Equal(t, openAIWSRequestCompatibility(a), openAIWSRequestCompatibility(a))
}

func TestStickyProxyRewritesUdealSessionWithoutSidT(t *testing.T) {
	proxy := Proxy{Host: "us.udealproxy.com", Port: 6666, Protocol: "http", Username: "userId-7155-custom-19653-region-us-session-ZHACt-sessTime-20", Password: "secret"}
	rewritten := stickyProxy(&proxy, "abc12xyz", 30)
	require.Equal(t, "userId-7155-custom-19653-region-us-session-abc12xyz-sessTime-30", rewritten.Username)
	require.NotContains(t, rewritten.Username, "-sid-")
	require.False(t, hasSidTParameters(rewritten.Username))
	require.Equal(t, 1, strings.Count(rewritten.Username, "-session-"))
	require.Equal(t, 1, strings.Count(rewritten.Username, "-sessTime-"))
}

func TestStickyProxyKeepsSidTForNonUdealUsernames(t *testing.T) {
	proxy := Proxy{Username: "user-region-US-sid-old-t-5"}
	rewritten := stickyProxy(&proxy, "sessionA", 30)
	require.Equal(t, "user-region-US-sid-sessionA-t-30", rewritten.Username)
}

func TestStickyLooksUdealRequiresHostOrSessionAndSessTime(t *testing.T) {
	require.True(t, stickyLooksUdeal(&Proxy{Host: "us.udealproxy.com", Username: "user-region-US"}))
	require.True(t, stickyLooksUdeal(&Proxy{Host: "UdealProxy.com", Username: "user-region-US"}))
	require.True(t, stickyLooksUdeal(&Proxy{Host: "proxy.example", Username: "userId-7155-custom-19653-region-us-session-ZHACt-sessTime-20"}))
	require.False(t, stickyLooksUdeal(&Proxy{Host: "proxy.example", Username: "user-region-US-sid-old-t-5"}))
	require.False(t, stickyLooksUdeal(&Proxy{Host: "proxy.example", Username: "user-session-only"}))
	require.False(t, stickyLooksUdeal(&Proxy{Host: "udealproxy.com.example", Username: "user-region-US"}))
}

func TestStickyProxyStripsCaseInsensitiveUdealSession(t *testing.T) {
	proxy := Proxy{Host: "us.udealproxy.com", Username: "userId-7155-custom-19653-region-us-SESSION-ZHACt-SESSTIME-20"}
	rewritten := stickyProxy(&proxy, "abc12xyz", 30)
	require.Equal(t, "userId-7155-custom-19653-region-us-session-abc12xyz-sessTime-30", rewritten.Username)
}

func TestRotateUdealStickySessionCallsUpdateAPI(t *testing.T) {
	var gotCustom, gotSession string
	var hasPassword bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/update", r.URL.Path)
		gotCustom = r.URL.Query().Get("custom")
		gotSession = r.URL.Query().Get("session")
		hasPassword = r.URL.Query().Get("password") != ""
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	restore := useStickyVendorTestServer(t, server)
	defer restore()
	host, _, err := net.SplitHostPort(parsedHostPort(t, server.URL))
	require.NoError(t, err)
	proxy := Proxy{Host: host, Port: 6666, Protocol: "http", Username: "userId-7155-custom-19653-region-us-session-ZHACt-sessTime-20", Password: "secret"}
	require.NoError(t, rotateUdealStickySession(context.Background(), &proxy, "abc12xyz"))
	require.Equal(t, "19653", gotCustom)
	require.Equal(t, "abc12xyz", gotSession)
	require.True(t, hasPassword)
}

func TestRotateUdealStickySessionSkipsNonUdeal(t *testing.T) {
	proxy := Proxy{Host: "proxy.example", Username: "user-region-US-sid-old-t-5", Password: "secret"}
	require.NoError(t, rotateUdealStickySession(context.Background(), &proxy, "abc12xyz"))
}

func TestRotateUdealStickySessionRejectsNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	restore := useStickyVendorTestServer(t, server)
	defer restore()
	host, _, err := net.SplitHostPort(parsedHostPort(t, server.URL))
	require.NoError(t, err)
	proxy := Proxy{Host: host, Port: 6666, Protocol: "http", Username: "userId-7155-custom-19653-region-us-session-ZHACt-sessTime-20", Password: "secret"}
	require.ErrorIs(t, rotateUdealStickySession(context.Background(), &proxy, "abc12xyz"), errStickyVendorRotate)
}

func parsedHostPort(t *testing.T, raw string) string {
	t.Helper()
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	return parsed.Host
}

func useStickyVendorTestServer(t *testing.T, server *httptest.Server) func() {
	t.Helper()
	_, port, err := net.SplitHostPort(parsedHostPort(t, server.URL))
	require.NoError(t, err)
	previousPort := stickyUdealUpdatePort
	previousClient := stickyVendorHTTPClient
	stickyUdealUpdatePort = port
	stickyVendorHTTPClient = &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: server.Client().Transport,
	}
	return func() {
		stickyUdealUpdatePort = previousPort
		stickyVendorHTTPClient = previousClient
	}
}

func TestNewStickyExitRotatesUdealThenRewritesUsername(t *testing.T) {
	ctx := context.Background()
	groups := newProxyIPGroupRepoStub()
	group := &ProxyIPGroup{Name: "udeal", StickyMinutes: 20, ProxyIDs: []int64{3}}
	require.NoError(t, groups.Create(ctx, group))
	proxy := Proxy{ID: 3, Host: "us.udealproxy.com", Port: 6666, Protocol: "http", Username: "userId-7155-custom-19653-region-us-session-ZHACt-sessTime-20", Password: "secret", Status: StatusActive}
	svc := &TurnStateProbeService{proxyRepo: &proxyListByIDsStub{proxies: map[int64]Proxy{3: proxy}}}
	previous := rotateStickyVendorSession
	var rotatedSession string
	rotateStickyVendorSession = func(_ context.Context, p *Proxy, session string) error {
		rotatedSession = session
		require.Equal(t, int64(3), p.ID)
		require.Contains(t, p.Username, "session-ZHACt")
		return nil
	}
	defer func() { rotateStickyVendorSession = previous }()
	exitURL, rec, err := svc.newStickyExit(ctx, group, 0)
	require.NoError(t, err)
	require.Equal(t, rotatedSession, rec.StickySession)
	require.Contains(t, exitURL, "session-"+rotatedSession)
	require.Contains(t, exitURL, "sessTime-30")
	require.NotContains(t, exitURL, "session-ZHACt")
	require.False(t, hasSidTParameters(parsedUsername(t, exitURL)))
	require.Equal(t, 30, rec.StickySessionMinutes)
}

func TestNewStickyExitStopsWhenVendorRotateFails(t *testing.T) {
	ctx := context.Background()
	groups := newProxyIPGroupRepoStub()
	group := &ProxyIPGroup{Name: "udeal", StickyMinutes: 20, ProxyIDs: []int64{3}}
	require.NoError(t, groups.Create(ctx, group))
	proxy := Proxy{ID: 3, Host: "us.udealproxy.com", Port: 6666, Protocol: "http", Username: "userId-7155-custom-19653-region-us-session-ZHACt-sessTime-20", Password: "secret", Status: StatusActive}
	svc := &TurnStateProbeService{proxyRepo: &proxyListByIDsStub{proxies: map[int64]Proxy{3: proxy}}}
	previous := rotateStickyVendorSession
	rotateStickyVendorSession = func(context.Context, *Proxy, string) error {
		return errStickyVendorRotate
	}
	defer func() { rotateStickyVendorSession = previous }()
	_, rec, err := svc.newStickyExit(ctx, group, 0)
	require.ErrorIs(t, err, errStickyVendorRotate)
	require.Empty(t, rec.StickySession)
}

func parsedUsername(t *testing.T, raw string) string {
	t.Helper()
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	return parsed.User.Username()
}

func hasSidTParameters(username string) bool {
	return stickySidTParameter.MatchString(username)
}

type stickyFailingProxyRepo struct{ ProxyRepository }

func (*stickyFailingProxyRepo) ListByIDs(context.Context, []int64) ([]Proxy, error) {
	return nil, errors.New("temporary configuration read failure")
}
