package service

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// kin Turn-State 默认糖果题（与 origin/main-kin TurnStateProbe 一致）。
const (
	MonitorIQDefaultQuestion = "黑色袋子中有苹果味、桃子味、西瓜味糖果;每种分为圆形和五角星形，可用手感区分形状。圆形依次有7、9、8颗;五角星形依次有7、6、4颗。事先决定摸出的数量，最少取多少颗，才能保证拿到不同形状的苹果味和桃子味糖果?"
	MonitorIQDefaultAnswer   = "21"
)

const (
	MonitorIQStatusOK      = "iq_ok"
	MonitorIQStatusDown    = "iq_down"
	MonitorIQStatusTestErr = "test_error"
	MonitorIQStatusNetwork = "monitor_network"
)

var monitorIQSpaceRE = regexp.MustCompile(`\s+`)

func monitorCheckModeUsesIQ(checkMode string) bool {
	return defaultCheckMode(checkMode) == MonitorCheckModeIQ
}

func usesIQ(opts *CheckOptions) bool {
	return opts != nil && strings.TrimSpace(opts.IQPrompt) != ""
}

func defaultIQQuestion(question string) string {
	question = strings.TrimSpace(question)
	if question == "" {
		return MonitorIQDefaultQuestion
	}
	return question
}

func defaultIQAnswer(answer string) string {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return MonitorIQDefaultAnswer
	}
	return answer
}

func applyIQDefaults(m *ChannelMonitor) {
	if m == nil || !monitorCheckModeUsesIQ(m.CheckMode) {
		return
	}
	m.IQQuestion = defaultIQQuestion(m.IQQuestion)
	m.IQAnswer = defaultIQAnswer(m.IQAnswer)
}

func applyIQCheckOptions(opts *CheckOptions, m *ChannelMonitor) {
	if opts == nil || m == nil {
		return
	}
	opts.IQPrompt = defaultIQQuestion(m.IQQuestion)
	opts.IQAnswer = defaultIQAnswer(m.IQAnswer)
	opts.IQFuzzyMatch = m.IQFuzzyMatch
	opts.MaxTokens = monitorIQMaxTokens
}

func monitorIQAnswerMatches(got, want string, fuzzy bool) bool {
	want = normalizeMonitorIQAnswer(want)
	got = normalizeMonitorIQAnswer(got)
	if want == "" || got == "" {
		return false
	}
	if fuzzy {
		return strings.Contains(got, want)
	}
	return got == want
}

func normalizeMonitorIQAnswer(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		if r == ',' || r == '，' || r == '.' || r == '。' || r == ':' || r == '：' || r == ';' || r == '；' {
			continue
		}
		b.WriteRune(r)
	}
	return monitorIQSpaceRE.ReplaceAllString(b.String(), "")
}

func isMonitorNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return isMonitorNetworkError(urlErr.Err)
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "server misbehaving") ||
		(strings.Contains(msg, "dns") && strings.Contains(msg, "not found"))
}

type iqClassifyInput struct {
	err        error
	statusCode int
	respText   string
	rawBody    string
	latency    durationMillis
	expected   string
	fuzzy      bool
}

type durationMillis interface {
	Milliseconds() int64
}

func classifyIQOutcome(in iqClassifyInput) (status, iqStatus, message string) {
	if in.err != nil {
		if isMonitorNetworkError(in.err) {
			return MonitorStatusError, MonitorIQStatusNetwork, "monitor-side DNS lookup failed"
		}
		return MonitorStatusFailed, MonitorIQStatusTestErr, truncateMessage(sanitizeErrorMessage(in.err.Error()))
	}
	if in.statusCode < 200 || in.statusCode >= 300 {
		bodySnippet := truncateForErrorBody(in.rawBody)
		return MonitorStatusFailed, MonitorIQStatusTestErr,
			truncateMessage(sanitizeErrorMessage(fmt.Sprintf("upstream HTTP %d: %s", in.statusCode, bodySnippet)))
	}
	if strings.TrimSpace(in.respText) == "" {
		return MonitorStatusFailed, MonitorIQStatusTestErr, "iq: empty upstream text"
	}

	latencyMs := 0
	if in.latency != nil {
		latencyMs = int(in.latency.Milliseconds())
	}
	status = MonitorStatusOperational
	message = ""
	if in.latency != nil && in.latency.Milliseconds() >= monitorDegradedThreshold.Milliseconds() {
		status = MonitorStatusDegraded
		message = truncateMessage(fmt.Sprintf("slow response: %dms", latencyMs))
	}
	if monitorIQAnswerMatches(in.respText, in.expected, in.fuzzy) {
		return status, MonitorIQStatusOK, message
	}
	mismatch := truncateMessage(sanitizeErrorMessage(fmt.Sprintf("iq mismatch (expected %s, got %q)", in.expected, in.respText)))
	if message == "" {
		message = mismatch
	}
	return status, MonitorIQStatusDown, message
}
