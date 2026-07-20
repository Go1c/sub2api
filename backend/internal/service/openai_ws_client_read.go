package service

import (
	"context"
	"errors"
	"time"

	coderws "github.com/coder/websocket"
)

// openAIWSClientReadJoinTimeout bounds how long we wait for the concurrent
// reader to exit after CloseNow aborts the transport.
const openAIWSClientReadJoinTimeout = 500 * time.Millisecond

type openAIWSClientReadResult struct {
	messageType coderws.MessageType
	payload     []byte
	err         error
}

// ReadOpenAIWSClientMessage keeps one reader alive until a control event forces
// session teardown. Resource release must not wait for a full close handshake:
// coder/websocket.Close can block for several seconds when the peer is not
// reading, and once Close has marked the conn closing, CloseNow can no longer
// force-drop the transport. Always force-close first, then join the reader.
func ReadOpenAIWSClientMessage(
	controlCtx context.Context,
	conn *coderws.Conn,
	timeout time.Duration,
	timeoutStatus coderws.StatusCode,
	timeoutReason string,
) (coderws.MessageType, []byte, error) {
	if conn == nil {
		return 0, nil, errors.New("openai websocket client connection is nil")
	}
	if controlCtx == nil {
		controlCtx = context.Background()
	}

	readDone := make(chan openAIWSClientReadResult, 1)
	go func() {
		// Use Background so a short-lived parent cancel does not close the
		// socket via setupReadTimeout before we decide how to tear down.
		messageType, payload, err := conn.Read(context.Background())
		readDone <- openAIWSClientReadResult{messageType: messageType, payload: payload, err: err}
	}()

	var timeoutCh <-chan time.Time
	var timer *time.Timer
	if timeout > 0 {
		timer = time.NewTimer(timeout)
		timeoutCh = timer.C
		defer timer.Stop()
	}

	closeAndJoin := func(status coderws.StatusCode, reason string, cause error) (coderws.MessageType, []byte, error) {
		// Force-close the underlying transport immediately. Do not call
		// conn.Close first: its handshake waits for a peer close frame and
		// races the concurrent reader, delaying lease release by seconds.
		_ = conn.CloseNow()
		select {
		case <-readDone:
		case <-time.After(openAIWSClientReadJoinTimeout):
		}
		return 0, nil, NewOpenAIWSClientCloseError(status, reason, cause)
	}

	select {
	case result := <-readDone:
		return result.messageType, result.payload, result.err
	case <-timeoutCh:
		return closeAndJoin(timeoutStatus, timeoutReason, context.DeadlineExceeded)
	case <-controlCtx.Done():
		cause := context.Cause(controlCtx)
		if errors.Is(cause, ErrOpenAIWSIngressLeaseLost) {
			return closeAndJoin(
				coderws.StatusTryAgainLater,
				"websocket ingress capacity lease lost; please reconnect",
				cause,
			)
		}
		return closeAndJoin(coderws.StatusGoingAway, "websocket request canceled", cause)
	}
}
