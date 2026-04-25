//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package sandbox

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// claimTerminalFor sets the controlling terminal's foreground process group
// to pgid (so terminal-driver-driven SIGINT/SIGQUIT/SIGTSTP from the
// keyboard are delivered there). It returns a non-nil restore function
// that must be called when the foreground child exits, even on the
// non-terminal path (then it's a no-op).
//
// Strategy:
//
//  1. Open /dev/tty for read+write. If that fails, we have no controlling
//     terminal; both the claim and the restore are no-ops.
//  2. Read the current foreground pgid via TIOCGPGRP.
//  3. Block SIGTTOU around tcsetpgrp, because tcsetpgrp from a non-foreground
//     pgid would otherwise stop us with SIGTTOU. Doing the call from within
//     a SIGTTOU block + ignoring the signal is the standard workaround used
//     by shells (bash, zsh, ksh) when they hand the terminal off to a job.
//  4. Set the new pgid.
//  5. The returned function reverses these steps when called.
func claimTerminalFor(pgid int) (tty *os.File, restore func()) {
	noop := func() {}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		// No controlling terminal (daemonized, CI, redirected stdin, etc.)
		return nil, noop
	}
	fd := int(tty.Fd())

	prevPgid, err := unix.IoctlGetInt(fd, unix.TIOCGPGRP)
	if err != nil {
		_ = tty.Close()
		return nil, noop
	}

	// Ignore SIGTTOU so the imminent tcsetpgrp does not stop us. We restore
	// the previous disposition in the restore func.
	prevTTOU := signalIgnore(syscall.SIGTTOU)
	prevTTIN := signalIgnore(syscall.SIGTTIN)

	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPGRP, pgid); err != nil {
		// Best effort: restore signals and bail. Most likely cause is that
		// stdin/stdout were redirected to a non-tty even though /dev/tty
		// existed (it usually does on macOS even from launchd contexts).
		signalReset(syscall.SIGTTOU, prevTTOU)
		signalReset(syscall.SIGTTIN, prevTTIN)
		_ = tty.Close()
		return nil, noop
	}

	restore = func() {
		// Hand the terminal back to whoever owned it before us. Errors
		// here are non-fatal and not user-actionable.
		_ = unix.IoctlSetPointerInt(fd, unix.TIOCSPGRP, prevPgid)
		signalReset(syscall.SIGTTOU, prevTTOU)
		signalReset(syscall.SIGTTIN, prevTTIN)
		_ = tty.Close()
	}
	return tty, restore
}

// signalIgnore sets SIG_IGN-equivalent behaviour for the given signal and
// returns a token that signalReset can use to put it back. We implement
// this with signal.Notify(unbuffered, no listener) — Go does not expose
// SIG_IGN directly. The handler installed by Notify simply discards the
// signal, which is functionally equivalent for our purposes.
type sigToken struct{ ch chan os.Signal }

func signalIgnore(sig syscall.Signal) sigToken {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sig)
	go func() {
		for range ch {
		}
	}()
	return sigToken{ch: ch}
}

func signalReset(_ syscall.Signal, t sigToken) {
	if t.ch == nil {
		return
	}
	signal.Stop(t.ch)
	close(t.ch)
}
