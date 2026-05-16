//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/suite"
)

type SettingRepoSuite struct {
	suite.Suite
	ctx  context.Context
	repo *settingRepository
}

func (s *SettingRepoSuite) SetupTest() {
	s.ctx = context.Background()
	tx := testEntTx(s.T())
	s.repo = NewSettingRepository(tx.Client()).(*settingRepository)
}

func TestSettingRepoSuite(t *testing.T) {
	suite.Run(t, new(SettingRepoSuite))
}

func (s *SettingRepoSuite) TestSetAndGetValue() {
	s.Require().NoError(s.repo.Set(s.ctx, "k1", "v1"), "Set")
	got, err := s.repo.GetValue(s.ctx, "k1")
	s.Require().NoError(err, "GetValue")
	s.Require().Equal("v1", got, "GetValue mismatch")
}

func (s *SettingRepoSuite) TestSet_Upsert() {
	s.Require().NoError(s.repo.Set(s.ctx, "k1", "v1"), "Set")
	s.Require().NoError(s.repo.Set(s.ctx, "k1", "v2"), "Set upsert")
	got, err := s.repo.GetValue(s.ctx, "k1")
	s.Require().NoError(err, "GetValue after upsert")
	s.Require().Equal("v2", got, "upsert mismatch")
}

func (s *SettingRepoSuite) TestGetValue_Missing() {
	_, err := s.repo.GetValue(s.ctx, "nonexistent")
	s.Require().Error(err, "expected error for missing key")
	s.Require().ErrorIs(err, service.ErrSettingNotFound)
}

func (s *SettingRepoSuite) TestSetMultiple_AndGetMultiple() {
	s.Require().NoError(s.repo.SetMultiple(s.ctx, map[string]string{"k2": "v2", "k3": "v3"}), "SetMultiple")
	m, err := s.repo.GetMultiple(s.ctx, []string{"k2", "k3"})
	s.Require().NoError(err, "GetMultiple")
	s.Require().Equal("v2", m["k2"])
	s.Require().Equal("v3", m["k3"])
}

func (s *SettingRepoSuite) TestGetMultiple_EmptyKeys() {
	m, err := s.repo.GetMultiple(s.ctx, []string{})
	s.Require().NoError(err, "GetMultiple with empty keys")
	s.Require().Empty(m, "expected empty map")
}

func (s *SettingRepoSuite) TestGetMultiple_Subset() {
	s.Require().NoError(s.repo.SetMultiple(s.ctx, map[string]string{"a": "1", "b": "2", "c": "3"}))
	m, err := s.repo.GetMultiple(s.ctx, []string{"a", "c", "nonexistent"})
	s.Require().NoError(err, "GetMultiple subset")
	s.Require().Equal("1", m["a"])
	s.Require().Equal("3", m["c"])
	_, exists := m["nonexistent"]
	s.Require().False(exists, "nonexistent key should not be in map")
}

func (s *SettingRepoSuite) TestGetAll() {
	s.Require().NoError(s.repo.SetMultiple(s.ctx, map[string]string{"x": "1", "y": "2"}))
	all, err := s.repo.GetAll(s.ctx)
	s.Require().NoError(err, "GetAll")
	s.Require().GreaterOrEqual(len(all), 2, "expected at least 2 settings")
	s.Require().Equal("1", all["x"])
	s.Require().Equal("2", all["y"])
}

func (s *SettingRepoSuite) TestDelete() {
	s.Require().NoError(s.repo.Set(s.ctx, "todelete", "val"))
	s.Require().NoError(s.repo.Delete(s.ctx, "todelete"), "Delete")
	_, err := s.repo.GetValue(s.ctx, "todelete")
	s.Require().Error(err, "expected missing key error after Delete")
	s.Require().ErrorIs(err, service.ErrSettingNotFound)
}

func (s *SettingRepoSuite) TestDelete_Idempotent() {
	// Delete a key that doesn't exist should not error
	s.Require().NoError(s.repo.Delete(s.ctx, "nonexistent_delete"), "Delete nonexistent should be idempotent")
}

func (s *SettingRepoSuite) TestSetMultiple_Upsert() {
	s.Require().NoError(s.repo.Set(s.ctx, "upsert_key", "old_value"))
	s.Require().NoError(s.repo.SetMultiple(s.ctx, map[string]string{"upsert_key": "new_value", "new_key": "new_val"}))

	got, err := s.repo.GetValue(s.ctx, "upsert_key")
	s.Require().NoError(err)
	s.Require().Equal("new_value", got, "SetMultiple should upsert existing key")

	got2, err := s.repo.GetValue(s.ctx, "new_key")
	s.Require().NoError(err)
	s.Require().Equal("new_val", got2)
}

func (s *SettingRepoSuite) TestSetMultiple_UpdatesExistingKey() {
	s.Require().NoError(s.repo.Set(s.ctx, "existing_update_key", "old_value"))
	before, err := s.repo.Get(s.ctx, "existing_update_key")
	s.Require().NoError(err)

	s.Require().NoError(s.repo.SetMultiple(s.ctx, map[string]string{"existing_update_key": "new_value"}))

	after, err := s.repo.Get(s.ctx, "existing_update_key")
	s.Require().NoError(err)
	s.Require().Equal(before.ID, after.ID)
	s.Require().Equal("new_value", after.Value)
}

func (s *SettingRepoSuite) TestSetMultiple_MixedExistingAndNewKeys() {
	s.Require().NoError(s.repo.Set(s.ctx, "mixed_existing_key", "old_value"))
	s.Require().NoError(s.repo.SetMultiple(s.ctx, map[string]string{
		"mixed_existing_key": "new_value",
		"mixed_new_key":      "created_value",
	}))

	got, err := s.repo.GetMultiple(s.ctx, []string{"mixed_existing_key", "mixed_new_key"})
	s.Require().NoError(err)
	s.Require().Equal("new_value", got["mixed_existing_key"])
	s.Require().Equal("created_value", got["mixed_new_key"])
}

func (s *SettingRepoSuite) TestSetMultiple_RollsBackWhenOneKeyFails() {
	repo := NewSettingRepository(testEntClient(s.T())).(*settingRepository)
	firstKey := "set_multiple_tx_rollback_first"
	invalidKey := "set_multiple_tx_rollback_zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"

	err := repo.SetMultiple(s.ctx, map[string]string{
		firstKey:   "should_not_persist",
		invalidKey: "too_long",
	})
	s.Require().Error(err)

	_, err = repo.GetValue(s.ctx, firstKey)
	s.Require().ErrorIs(err, service.ErrSettingNotFound)
}

func (s *SettingRepoSuite) TestUpdateSettingsWithAuthSourceDefaults_DisableWeChatPCAfterEnableIsIdempotent() {
	svc := service.NewSettingService(s.repo, &config.Config{})
	authDefaults := &service.AuthSourceDefaultSettings{}
	enabled := &service.SystemSettings{
		WeChatConnectEnabled:             true,
		WeChatConnectAppID:               "wx_legacy_app",
		WeChatConnectAppSecret:           "wx_legacy_secret",
		WeChatConnectOpenAppID:           "wx_pc_app",
		WeChatConnectOpenAppSecret:       "wx_pc_secret",
		WeChatConnectOpenEnabled:         true,
		WeChatConnectMode:                "open",
		WeChatConnectScopes:              "snsapi_login",
		WeChatConnectRedirectURL:         "https://example.com/api/v1/auth/wechat/callback",
		WeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
	}
	disabled := &service.SystemSettings{
		WeChatConnectEnabled:             false,
		WeChatConnectAppID:               "wx_legacy_app",
		WeChatConnectOpenAppID:           "wx_pc_app",
		WeChatConnectOpenEnabled:         false,
		WeChatConnectMode:                "open",
		WeChatConnectScopes:              "snsapi_login",
		WeChatConnectRedirectURL:         "https://example.com/api/v1/auth/wechat/callback",
		WeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
	}

	s.Require().NoError(svc.UpdateSettingsWithAuthSourceDefaults(s.ctx, enabled, authDefaults))
	s.Require().NoError(svc.UpdateSettingsWithAuthSourceDefaults(s.ctx, disabled, authDefaults))
	s.Require().NoError(svc.UpdateSettingsWithAuthSourceDefaults(s.ctx, disabled, authDefaults))

	got, err := s.repo.GetMultiple(s.ctx, []string{
		service.SettingKeyWeChatConnectEnabled,
		service.SettingKeyWeChatConnectOpenEnabled,
		service.SettingKeyWeChatConnectOpenAppID,
	})
	s.Require().NoError(err)
	s.Require().Equal("false", got[service.SettingKeyWeChatConnectEnabled])
	s.Require().Equal("false", got[service.SettingKeyWeChatConnectOpenEnabled])
	s.Require().Equal("wx_pc_app", got[service.SettingKeyWeChatConnectOpenAppID])
}

// TestSet_EmptyValue 测试保存空字符串值
// 这是一个回归测试，确保可选设置（如站点Logo、API端点地址等）可以保存为空字符串
func (s *SettingRepoSuite) TestSet_EmptyValue() {
	// 测试 Set 方法保存空值
	s.Require().NoError(s.repo.Set(s.ctx, "empty_key", ""), "Set with empty value should succeed")

	got, err := s.repo.GetValue(s.ctx, "empty_key")
	s.Require().NoError(err, "GetValue for empty value")
	s.Require().Equal("", got, "empty value should be preserved")
}

// TestSetMultiple_WithEmptyValues 测试批量保存包含空字符串的设置
// 模拟用户保存站点设置时部分字段为空的场景
func (s *SettingRepoSuite) TestSetMultiple_WithEmptyValues() {
	// 模拟保存站点设置，部分字段有值，部分字段为空
	settings := map[string]string{
		"site_name":     "Sub2api",
		"site_subtitle": "Subscription to API",
		"site_logo":     "", // 用户未上传Logo
		"api_base_url":  "", // 用户未设置API地址
		"contact_info":  "", // 用户未设置联系方式
		"doc_url":       "", // 用户未设置文档链接
	}

	s.Require().NoError(s.repo.SetMultiple(s.ctx, settings), "SetMultiple with empty values should succeed")

	// 验证所有值都正确保存
	result, err := s.repo.GetMultiple(s.ctx, []string{"site_name", "site_subtitle", "site_logo", "api_base_url", "contact_info", "doc_url"})
	s.Require().NoError(err, "GetMultiple after SetMultiple with empty values")

	s.Require().Equal("Sub2api", result["site_name"])
	s.Require().Equal("Subscription to API", result["site_subtitle"])
	s.Require().Equal("", result["site_logo"], "empty site_logo should be preserved")
	s.Require().Equal("", result["api_base_url"], "empty api_base_url should be preserved")
	s.Require().Equal("", result["contact_info"], "empty contact_info should be preserved")
	s.Require().Equal("", result["doc_url"], "empty doc_url should be preserved")
}

// TestSetMultiple_UpdateToEmpty 测试将已有值更新为空字符串
// 确保用户可以清空之前设置的值
func (s *SettingRepoSuite) TestSetMultiple_UpdateToEmpty() {
	// 先设置非空值
	s.Require().NoError(s.repo.Set(s.ctx, "clearable_key", "initial_value"))

	got, err := s.repo.GetValue(s.ctx, "clearable_key")
	s.Require().NoError(err)
	s.Require().Equal("initial_value", got)

	// 更新为空值
	s.Require().NoError(s.repo.SetMultiple(s.ctx, map[string]string{"clearable_key": ""}), "Update to empty should succeed")

	got, err = s.repo.GetValue(s.ctx, "clearable_key")
	s.Require().NoError(err)
	s.Require().Equal("", got, "value should be updated to empty string")
}

// TestSetMultiple_ConsecutiveSavesAreIdempotent 直接复现生产场景：
// 管理员 PUT /api/v1/admin/settings 携带相同 key 连续两次必须都成功，
// 且 id 不变（即第二次走 UPDATE 而非 INSERT）。这是 b161265e 之后仍出现
// settings_key_key 报错的反向断言。
func (s *SettingRepoSuite) TestSetMultiple_ConsecutiveSavesAreIdempotent() {
	repo := NewSettingRepository(testEntClient(s.T())).(*settingRepository)
	const key = "custom_menu_items"

	s.Require().NoError(repo.SetMultiple(s.ctx, map[string]string{key: "[]"}))
	first, err := repo.Get(s.ctx, key)
	s.Require().NoError(err)

	s.Require().NoError(repo.SetMultiple(s.ctx, map[string]string{key: `[{"label":"home"}]`}))
	second, err := repo.Get(s.ctx, key)
	s.Require().NoError(err)

	s.Require().Equal(first.ID, second.ID, "id must not change across consecutive SetMultiple calls")
	s.Require().Equal(`[{"label":"home"}]`, second.Value)

	// 再保存一次，确保第三次也不会 duplicate key
	s.Require().NoError(repo.SetMultiple(s.ctx, map[string]string{key: `[{"label":"home"},{"label":"about"}]`}))
	third, err := repo.Get(s.ctx, key)
	s.Require().NoError(err)
	s.Require().Equal(first.ID, third.ID)
}
