package cli

import (
	"reflect"
	"testing"
)

func TestSplitHarnessToken(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantName  string
		wantRest  []string
		wantError bool
	}{
		{"empty -> default", nil, "opencode", nil, false},
		{"leading flag -> default", []string{"--verbose"}, "opencode", []string{"--verbose"}, false},
		{"double dash -> default", []string{"--", "foo"}, "opencode", []string{"--", "foo"}, false},
		{"explicit opencode", []string{"opencode"}, "opencode", []string{}, false},
		{"explicit claude", []string{"claude"}, "claude-code", []string{}, false},
		{"claude alias cc", []string{"cc", "--verbose"}, "claude-code", []string{"--verbose"}, false},
		{"opencode then flags", []string{"opencode", "--inner", "x"}, "opencode", []string{"--inner", "x"}, false},
		{"unknown bareword -> error", []string{"claud"}, "", nil, true},
		{"unknown bareword with flags -> error", []string{"bash", "--x"}, "", nil, true},
	}
	for _, c := range cases {
		h, rest, err := splitHarnessToken(c.args)
		if c.wantError {
			if err == nil {
				t.Errorf("%s: expected error, got none", c.name)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
			continue
		}
		if h.Name != c.wantName {
			t.Errorf("%s: harness = %q, want %q", c.name, h.Name, c.wantName)
		}
		if !reflect.DeepEqual(rest, c.wantRest) {
			t.Errorf("%s: rest = %v, want %v", c.name, rest, c.wantRest)
		}
	}
}
