//go:build unit

package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

type paymentConfigHandlerSettingRepoStub struct {
	values map[string]string
}

func (r *paymentConfigHandlerSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, nil
}

func (r *paymentConfigHandlerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return r.values[key], nil
}

func (r *paymentConfigHandlerSettingRepoStub) Set(context.Context, string, string) error {
	return nil
}

func (r *paymentConfigHandlerSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = r.values[key]
	}
	return out, nil
}

func (r *paymentConfigHandlerSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}

func (r *paymentConfigHandlerSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return r.values, nil
}

func (r *paymentConfigHandlerSettingRepoStub) Delete(context.Context, string) error {
	return nil
}

func TestGetPaymentConfigReturnsStripePublishableKey(t *testing.T) {
	ctx := context.Background()
	client := newPaymentHandlerConfigTestClient(t)
	const publishableKey = "pk_test_public"
	const secretKey = "sk_test_should_not_escape"

	_, err := client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeStripe).
		SetName("Stripe").
		SetConfig(fmt.Sprintf(`{"publishableKey":%q,"secretKey":%q}`, publishableKey, secretKey)).
		SetSupportedTypes("card,link").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create stripe provider instance: %v", err)
	}

	gin.SetMode(gin.TestMode)
	configSvc := service.NewPaymentConfigService(client, &paymentConfigHandlerSettingRepoStub{
		values: map[string]string{
			service.SettingPaymentEnabled: "true",
		},
	}, nil)
	h := NewPaymentHandler(nil, configSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment/config", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	h.GetPaymentConfig(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), secretKey) {
		t.Fatalf("response leaked Stripe secret key: %s", w.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			StripePublishableKey string `json:"stripe_publishable_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	if resp.Data.StripePublishableKey != publishableKey {
		t.Fatalf("stripe_publishable_key = %q, want %q", resp.Data.StripePublishableKey, publishableKey)
	}
}

func newPaymentHandlerConfigTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	dbName := fmt.Sprintf(
		"file:%s?mode=memory&cache=shared",
		strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()),
	)
	db, err := sql.Open("sqlite", dbName)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}
