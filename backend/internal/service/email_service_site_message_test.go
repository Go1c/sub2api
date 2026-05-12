package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type emailServiceSettingRepoStub struct {
	values map[string]string
}

func (s *emailServiceSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *emailServiceSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *emailServiceSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *emailServiceSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *emailServiceSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *emailServiceSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *emailServiceSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestBuildSiteMessageCopyEmailBodyIncludesFrontendURLInHeader(t *testing.T) {
	svc := NewEmailService(nil, nil)

	body := svc.buildSiteMessageCopyEmailBody("不要说话", "不要大声说话", "https://api.lumio.games/")

	require.Contains(t, body, "<h1>Site Message</h1>")
	require.Contains(t, body, `href="https://api.lumio.games/"`)
	require.Contains(t, body, "https://api.lumio.games/")
	require.NotContains(t, body, "%!")
}

func TestBuildSiteMessageCopyEmailBodyEscapesFrontendURL(t *testing.T) {
	svc := NewEmailService(nil, nil)

	body := svc.buildSiteMessageCopyEmailBody("subject", "body", `https://example.com/?a=1&next=<script>`)

	require.Contains(t, body, `href="https://example.com/?a=1&amp;next=&lt;script&gt;"`)
	require.Contains(t, body, "https://example.com/?a=1&amp;next=&lt;script&gt;")
	require.NotContains(t, body, "<script>")
	require.NotContains(t, body, "%!")
}

func TestSiteMessageCopyFrontendURLReadsConfiguredSetting(t *testing.T) {
	repo := &emailServiceSettingRepoStub{
		values: map[string]string{
			SettingKeyFrontendURL: " https://api.lumio.games/ ",
		},
	}
	svc := NewEmailService(repo, nil)

	frontendURL := svc.siteMessageCopyFrontendURL(context.Background())

	require.Equal(t, "https://api.lumio.games/", frontendURL)
}
