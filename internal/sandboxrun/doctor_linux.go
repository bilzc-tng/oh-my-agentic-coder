//go:build linux

package sandboxrun

import "fmt"

// DoctorNotes returns extra platform diagnostics for `omac doctor`.
func DoctorNotes() []string {
	abi := LandlockABI()
	if abi >= landlockNetABI {
		return []string{fmt.Sprintf("[ok] Landlock ABI %d (network rules supported)", abi)}
	}
	return []string{fmt.Sprintf(
		"[warn] Landlock ABI %d < %d (kernel < 6.7): kernel-enforced network filtering unavailable; "+
			"filtered mode fails closed unless the sandbox profile sets network.enforcement to \"env-only\"",
		abi, landlockNetABI)}
}
