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
	return readOpenAIWSClientMessageWithTimeoutStart(
		controlCtx,
		conn,
		timeout,
		timeoutStatus,
		timeoutReason,
		nil,
		nil,
	)
}

// readOpenAIWSClientMessageWithTimeoutStart supports readers whose timeout
// starts after a state transition, such as a completed passthrough turn. When
// timeoutActive is nil, a positive timeout starts immediately.
func readOpenAIWSClientMessageWithTimeoutStart(
	controlCtx context.Context,
	conn *coderws.Conn,
	timeout time.Duration,
	timeoutStatus coderws.StatusCode,
	timeoutReason string,
	timeoutStart <-chan struct{},
	timeoutActive func() bool,
) (coderws.MessageType, []byte, error) {
	if conn == nil {
		return 0, nil, errors.New("openai websocket client connection is nil")
	}
	if controlCtx == nil {
		controlCtx = context.Background()
	}

	readDone := make(chan openAIWSClientReadResult, 1)
	go func() {
		messageType, payload, err := conn.Read(context.Background())
		readDone <- openAIWSClientReadResult{messageType: messageType, payload: payload, err: err}
	}()

	var timer *time.Timer
	var timeoutCh <-chan time.Time
	startTimeout := func() {
		if timeout <= 0 || (timeoutActive != nil && !timeoutActive()) {
			return
		}
		if timer == nil {
			timer = time.NewTimer(timeout)
		} else {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(timeout)
		}
		timeoutCh = timer.C
	}
	if timeoutActive == nil || timeoutActive() {
		startTimeout()
	}
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	// Force-close the underlying transport immediately. Do not call
	// conn.Close first: its handshake waits for a peer close frame and
	// races the concurrent reader, delaying lease release by seconds.
	// Join the reader with a short bound so ingress leases are released
	// near the configured idle timeout (fork fix b50f84189).
	closeAndJoin := func(status coderws.StatusCode, reason string, cause error) (coderws.MessageType, []byte, error) {
		_ = conn.CloseNow()
		select {
		case <-readDone:
		case <-time.After(openAIWSClientReadJoinTimeout):
		}
		return 0, nil, NewOpenAIWSClientCloseError(status, reason, cause)
	}

	for {
		select {
		case result := <-readDone:
			return result.messageType, result.payload, result.err
		case <-timeoutStart:
			startTimeout()
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
}
