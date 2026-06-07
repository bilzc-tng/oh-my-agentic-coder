package cli

import (
	"strings"

	"github.com/tngtech/oh-my-agentic-coder/internal/config"
)

// splitHarnessToken inspects the first token of a subcommand's args and
// resolves the inner-harness selector.
//
// The first positional slot (before any flag and before "--") is the harness
// selector slot:
//
//   - empty args, a leading flag, or a leading "--": no selector given →
//     default harness, args unchanged.
//   - a known harness name/alias: consume it → that harness, remaining args.
//   - any other bareword: treated as an attempted-but-unknown harness and
//     rejected (err non-nil), so typos like `omac start claud` fail loudly
//     instead of being silently passed through as an inner argument. Inner
//     arguments that happen to be barewords must be placed after "--".
//
// This implements the positional-harness UX: `omac start claude --verbose`,
// `omac start opencode`, `omac start` (defaults to opencode), and
// `omac start -- some-inner-arg`.
func splitHarnessToken(args []string) (config.Harness, []string, error) {
	if len(args) == 0 {
		return config.DefaultHarness(), args, nil
	}
	first := args[0]
	if first == "" || first == "--" || isFlag(first) {
		return config.DefaultHarness(), args, nil
	}
	if h, ok := config.LookupHarness(first); ok {
		return h, args[1:], nil
	}
	return config.Harness{}, nil, config.UnknownHarnessError(first)
}

// reorderFlagsFirst sorts args so all flag-like tokens ("-x", "--xx", "--xx=v")
// come before any positional. A bare "--" is a hard stop that forwards the rest
// verbatim, and "-" (a single dash) is treated as a positional (convention for
// stdin).
//
// This is a small QoL tweak so users can write either
//
//	omac register demo-echo --no-secrets
//
// or
//
//	omac register --no-secrets demo-echo
//
// without the stdlib flag package rejecting the first form.
//
// It does NOT know which flags take values; any "--foo bar" pair where the
// second token is not itself a flag is kept adjacent. Users with a positional
// literally starting with "-" should pass "--" first.
func reorderFlagsFirst(args []string) []string {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			// Everything after -- is positional verbatim.
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if isFlag(a) {
			flags = append(flags, a)
			// If this looks like "--foo" with no "=" and a following value token
			// that is not itself a flag, take that value with it.
			if !strings.Contains(a, "=") && i+1 < len(args) && !isFlag(args[i+1]) {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		positionals = append(positionals, a)
	}
	return append(flags, positionals...)
}

func isFlag(a string) bool {
	return len(a) >= 2 && a[0] == '-' && a != "-"
}
