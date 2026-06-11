//go:build linux

package sandboxrun

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/tngtech/oh-my-agentic-coder/internal/sandboxprofile"
)

// CheckPlatform verifies the Linux sandbox prerequisites.
func CheckPlatform() error {
	if _, err := exec.LookPath("bwrap"); err != nil {
		return fmt.Errorf("bubblewrap (bwrap) not found on PATH — install it with your package manager (e.g. apt install bubblewrap / dnf install bubblewrap): %w", err)
	}
	return nil
}

// BuildChildArgv wraps the inner command in bwrap + the stage2
// re-exec. self is the path to the running omac binary.
func BuildChildArgv(g *Grants, innerArgv []string) ([]string, error) {
	if err := CheckPlatform(); err != nil {
		return nil, err
	}
	if g.NetworkMode == sandboxprofile.ModeFiltered && g.Enforcement == sandboxprofile.EnforceKernel {
		if !LandlockNetSupported() {
			return nil, fmt.Errorf(
				"kernel-enforced network filtering needs Landlock ABI >= 4 (Linux >= 6.7); this kernel has ABI %d.\n"+
					"Either upgrade the kernel or set network.enforcement to \"env-only\" in the sandbox profile (WARNING: advisory-only filtering)",
				LandlockABI())
		}
	}
	self, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve omac executable: %w", err)
	}
	stage2 := []string{self, "sandbox", "stage2"}
	stage2 = append(stage2, Stage2Args(g)...)
	stage2 = append(stage2, "--")
	stage2 = append(stage2, innerArgv...)
	return BuildBwrapArgv(g, stage2)
}
