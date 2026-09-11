//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyAccountProxyIPGroup_RejectsNonOpenAI(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{Name: "fr"}))
	svc := &adminServiceImpl{proxyIPGroupRepo: groups}
	account := &Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth}
	gid := groups.nextID - 1
	err := svc.applyAccountProxyIPGroup(context.Background(), account, PlatformAnthropic, AccountTypeOAuth, &gid, true)
	require.ErrorIs(t, err, ErrProxyIPGroupNotAllowed)
}

func TestApplyAccountProxyIPGroup_RejectsOpenAIAPIKey(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{Name: "fr"}))
	svc := &adminServiceImpl{proxyIPGroupRepo: groups}
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	gid := int64(1)
	err := svc.applyAccountProxyIPGroup(context.Background(), account, PlatformOpenAI, AccountTypeAPIKey, &gid, true)
	require.ErrorIs(t, err, ErrProxyIPGroupNotAllowed)
}

func TestApplyAccountProxyIPGroup_ClearsProxyID(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{Name: "fr"}))
	svc := &adminServiceImpl{proxyIPGroupRepo: groups}
	proxyID := int64(9)
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyID: &proxyID}
	gid := int64(1)
	require.NoError(t, svc.applyAccountProxyIPGroup(context.Background(), account, PlatformOpenAI, AccountTypeOAuth, &gid, true))
	require.Equal(t, &gid, account.ProxyIPGroupID)
	require.Nil(t, account.ProxyID)
}

func TestApplyAccountProxyIPGroup_ZeroClearsGroup(t *testing.T) {
	svc := &adminServiceImpl{}
	gid := int64(3)
	zero := int64(0)
	account := &Account{ProxyIPGroupID: &gid}
	require.NoError(t, svc.applyAccountProxyIPGroup(context.Background(), account, PlatformOpenAI, AccountTypeOAuth, &zero, false))
	require.Nil(t, account.ProxyIPGroupID)
}

func TestBuildAccountForCreate_NoGroupKeepsProxyID(t *testing.T) {
	proxyID := int64(4)
	account, err := buildAccountForCreate(&CreateAccountInput{
		Name:     "codex",
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		ProxyID:  &proxyID,
	}, map[string]any{})
	require.NoError(t, err)
	require.Equal(t, &proxyID, account.ProxyID)
	require.Nil(t, account.ProxyIPGroupID)
}
