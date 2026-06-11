package sandboxrun

import (
	"fmt"
	"io"
	"os"

	"github.com/tngtech/oh-my-agentic-coder/internal/netprompt"
	"github.com/tngtech/oh-my-agentic-coder/internal/netproxy"
	"github.com/tngtech/oh-my-agentic-coder/internal/sandbox"
	"github.com/tngtech/oh-my-agentic-coder/internal/sandboxprofile"
)

// Options bundles the inputs for Run.
type Options struct {
	Flags   *sandboxprofile.Flags
	Workdir string
	Stderr  io.Writer
}

// Run is the `omac sandbox run` supervisor: resolve profile + flags,
// start the filtering proxy, launch the sandboxed child, forward
// signals, propagate the exit code, tear everything down. Returns the
// process exit code.
func Run(opts Options) int {
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	// Fatal/pre-launch problems go to stderr (the TUI hasn't started
	// yet). Runtime diagnostics — proxy decisions, prompt notices —
	// fire while the inner TUI owns the terminal, so they go through
	// the diag sink (a log file when stderr is a terminal).
	diag := newDiagSink(stderr)
	defer diag.Close()

	fail := func(format string, args ...any) int {
		fmt.Fprintf(stderr, "omac sandbox: "+format+"\n", args...)
		return 1
	}

	profile, err := sandboxprofile.Resolve(opts.Flags.ProfileRef)
	if err != nil {
		return fail("%v", err)
	}
	merged, warnings := sandboxprofile.Merge(profile, opts.Flags)
	for _, w := range warnings {
		fmt.Fprintf(stderr, "omac sandbox: warning: %s\n", w)
	}
	if err := merged.Validate(); err != nil {
		return fail("%v", err)
	}

	grants, err := ResolveGrants(merged, opts.Workdir, diag.Writer())
	if err != nil {
		return fail("%v", err)
	}
	if err := grants.Validate(); err != nil {
		return fail("%v", err)
	}

	logf := diag.Logf

	// Injected child env (proxy vars). Built before the backend so the
	// proxy port can land in the kernel rules.
	injected := map[string]string{}

	var proxy *netproxy.Server
	if grants.NetworkMode == sandboxprofile.ModeFiltered {
		if grants.Enforcement == sandboxprofile.EnforceEnvOnly {
			fmt.Fprintln(stderr, "omac sandbox: WARNING: network.enforcement is \"env-only\" — "+
				"filtering relies on HTTP(S)_PROXY env vars only and is trivially bypassable. "+
				"No kernel network guarantee is in effect.")
		}
		proxy, err = buildProxy(merged, diag.Writer(), logf)
		if err != nil {
			return fail("%v", err)
		}
		defer proxy.Close()
		grants.ProxyPort = proxy.Port()
		for k, v := range proxy.EnvVars() {
			injected[k] = v
		}
	}

	childArgv, err := BuildChildArgv(grants, opts.Flags.InnerArgv)
	if err != nil {
		return fail("%v", err)
	}

	// Last line before the inner process owns the terminal: tell the
	// user where runtime diagnostics will land.
	diag.AnnouncePath()

	env := sandboxprofile.FilterEnv(os.Environ(), merged.Environment.AllowVars, injected)
	code, err := sandbox.ExecWithEnv(childArgv, env, nil)
	if err != nil {
		return fail("%v", err)
	}
	return code
}

// buildProxy assembles learned policy, prompter, filter and server.
func buildProxy(p *sandboxprofile.Profile, stderr io.Writer, logf func(string, ...any)) (*netproxy.Server, error) {
	var learned netproxy.LearnedStore
	learnedPath, err := netprompt.DefaultLearnedPath(profileNameOrDefault(p))
	if err == nil {
		lp, lerr := netprompt.LoadLearnedPolicy(learnedPath)
		if lerr != nil {
			fmt.Fprintf(stderr, "omac sandbox: warning: %v (starting with empty learned policy)\n", lerr)
			lp, _ = netprompt.LoadLearnedPolicy("")
		}
		learned = lp
	}

	var prompter netproxy.Prompter
	onUnavailableAllow := p.Network.OnUnavailable() == sandboxprofile.OnUnavailableAllow
	if p.Network.PromptEnabled() {
		np, available := netprompt.NewPrompter(p.Network.PromptTimeoutSecs(), logf)
		if available {
			prompter = np
		} else {
			fmt.Fprintf(stderr, "omac sandbox: notice: no dialog backend available; network prompt falls back to on_unavailable=%s\n",
				p.Network.OnUnavailable())
		}
	}

	filter := netproxy.NewFilter(netproxy.FilterConfig{
		AllowDomains:       p.Network.AllowDomain,
		DenyDomains:        p.Network.DenyDomain,
		PromptEnabled:      p.Network.PromptEnabled(),
		OnUnavailableAllow: onUnavailableAllow,
		Prompter:           prompter,
		Learned:            learned,
		Logf:               logf,
	})
	srv, err := netproxy.NewServer(filter, logf)
	if err != nil {
		return nil, err
	}
	if err := srv.Start(); err != nil {
		return nil, err
	}
	return srv, nil
}

func profileNameOrDefault(p *sandboxprofile.Profile) string {
	if p.Meta.Name != "" {
		return p.Meta.Name
	}
	return "default"
}
