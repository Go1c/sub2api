package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestProxyResponsesWebSocketFromClient_PassiveImageGenNamespaceDoesNotCloseDisabledGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	first := []byte(`{
		"type":"response.create",
		"model":"gpt-6-astra",
		"stream":false,
		"input":"write code",
		"tool_choice":"auto",
		"tools":[
			{"type":"function","name":"Read"},
			{"type":"namespace","name":"image_gen","tools":[{"type":"function","name":"imagegen"}]}
		]
	}`)
	serverErr, clientConn, captureConn := startIngressImagePermissionWS(t, false)
	defer func() { _ = clientConn.CloseNow() }()

	writeWSText(t, clientConn, first)
	event := readWSText(t, clientConn)
	require.Equal(t, "response.completed", gjson.GetBytes(event, "type").String())
	require.Equal(t, "resp_ws_image_permission", gjson.GetBytes(event, "response.id").String())
	_ = clientConn.Close(coderws.StatusNormalClosure, "done")

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("waiting for ingress websocket to finish timed out")
	}
	require.Len(t, captureConn.writes, 1)
	require.True(t, gjson.Get(requestToJSONString(captureConn.writes[0]), `tools.#(name=="image_gen")`).Exists() ||
		gjson.Get(requestToJSONString(captureConn.writes[0]), `tools.#(type=="namespace")`).Exists())
}

func TestProxyResponsesWebSocketFromClient_NativeImageGenerationStillClosesDisabledGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	first := []byte(`{
		"type":"response.create",
		"model":"gpt-6-astra",
		"stream":false,
		"input":"draw a cat",
		"tools":[{"type":"image_generation","model":"gpt-image-2"}]
	}`)
	serverErr, clientConn, _ := startIngressImagePermissionWS(t, false)
	defer func() { _ = clientConn.CloseNow() }()

	writeWSText(t, clientConn, first)

	select {
	case err := <-serverErr:
		var closeErr *OpenAIWSClientCloseError
		require.ErrorAs(t, err, &closeErr)
		require.Equal(t, coderws.StatusPolicyViolation, closeErr.StatusCode())
		require.Equal(t, ImageGenerationPermissionMessage(), closeErr.Reason())
	case <-time.After(5 * time.Second):
		t.Fatal("waiting for ingress websocket permission close timed out")
	}
}

func startIngressImagePermissionWS(t *testing.T, allowImage bool) (<-chan error, *coderws.Conn, *openAIWSCaptureConn) {
	t.Helper()

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 1
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 1
	cfg.Gateway.OpenAIWS.QueueLimitPerConn = 8
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

	captureConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.completed","response":{"id":"resp_ws_image_permission","model":"gpt-6-astra","usage":{"input_tokens":1,"output_tokens":1}}}`),
		},
	}
	captureDialer := &openAIWSCaptureDialer{conn: captureConn}
	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(captureDialer)
	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     &httpUpstreamRecorder{},
		cache:            &stubGatewayCache{},
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
		openaiWSPool:     pool,
	}

	groupID := int64(3)
	apiKey := &APIKey{
		ID:      1,
		UserID:  1,
		GroupID: &groupID,
		Group: &Group{
			ID:                   groupID,
			AllowImageGeneration: allowImage,
		},
	}
	account := &Account{
		ID:          41,
		Name:        "openai-ws-image-permission",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "sk-test",
		},
		Extra: map[string]any{
			"responses_websockets_v2_enabled": true,
		},
	}

	serverErrCh := make(chan error, 1)
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{
			CompressionMode: coderws.CompressionContextTakeover,
		})
		if err != nil {
			serverErrCh <- err
			return
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		req := r.Clone(r.Context())
		req.Header = req.Header.Clone()
		req.Header.Set("User-Agent", "codex_cli_rs/0.98.0")
		ginCtx.Request = req
		ginCtx.Set("api_key", apiKey)

		readCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		msgType, firstMessage, readErr := conn.Read(readCtx)
		cancel()
		if readErr != nil {
			serverErrCh <- readErr
			return
		}
		if msgType != coderws.MessageText && msgType != coderws.MessageBinary {
			serverErrCh <- errors.New("unsupported websocket client message type")
			return
		}

		serverErrCh <- svc.ProxyResponsesWebSocketFromClient(r.Context(), ginCtx, conn, account, "sk-test", firstMessage, nil)
	}))
	t.Cleanup(wsServer.Close)

	dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
	clientConn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(wsServer.URL, "http"), nil)
	cancelDial()
	require.NoError(t, err)
	return serverErrCh, clientConn, captureConn
}

func writeWSText(t *testing.T, conn *coderws.Conn, payload []byte) {
	t.Helper()
	writeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	require.NoError(t, conn.Write(writeCtx, coderws.MessageText, payload))
}

func readWSText(t *testing.T, conn *coderws.Conn) []byte {
	t.Helper()
	readCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, event, err := conn.Read(readCtx)
	require.NoError(t, err)
	return event
}
