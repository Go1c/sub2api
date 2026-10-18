package service

import "strings"

const opsRequestBodyQueueLimit = 64 * 1024

// PrepareOpsRequestBodyForQueue keeps a bounded UTF-8 copy of the client body
// for the ops error queue.
func PrepareOpsRequestBodyForQueue(raw []byte) (body *string, truncated bool, n int) {
	n = len(raw)
	if n == 0 {
		return nil, false, 0
	}
	text := string(raw)
	if n > opsRequestBodyQueueLimit {
		text = text[:opsRequestBodyQueueLimit]
		truncated = true
	}
	text = strings.ToValidUTF8(text, "")
	return &text, truncated, n
}
