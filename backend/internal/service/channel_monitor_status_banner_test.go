package service

import "testing"

func TestNormalizeChannelMonitorStatusBanner(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "spaces", in: "   ", want: ""},
		{name: "trim", in: "  hello world  ", want: "hello world"},
		{name: "collapse whitespace", in: "a\n\tb  c", want: "a b c"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeChannelMonitorStatusBanner(tc.in)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}

	runes := make([]rune, 600)
	for i := range runes {
		runes[i] = 'x'
	}
	got := normalizeChannelMonitorStatusBanner(string(runes))
	if len([]rune(got)) != 500 {
		t.Fatalf("expected clamp to 500 runes, got %d", len([]rune(got)))
	}
}
