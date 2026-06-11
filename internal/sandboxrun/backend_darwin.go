//go:build darwin

package sandboxrun

import (
	"fmt"
	"os/exec"
)

// sandboxExecPath is the Seatbelt CLI shipped with every macOS.
const sandboxExecPath = "/usr/bin/sandbox-exec"

// CheckPlatform verifies the macOS sandbox prerequisites.
func CheckPlatform() error {
	if _, err := exec.LookPath(sandboxExecPath); err != nil {
		return fmt.Errorf("sandbox-exec not found at %s: %w", sandboxExecPath, err)
	}
	return nil
}

// BuildChildArgv wraps the inner command in sandbox-exec with the
// generated SBPL profile. The supervisor's filtered env is applied by
// the caller on the exec.Cmd.
func BuildChildArgv(g *Grants, innerArgv []string) ([]string, error) {
	if err := CheckPlatform(); err != nil {
		return nil, err
	}
	profile := GenerateSBPL(g)
	argv := []string{sandboxExecPath, "-p", profile, "--"}
	argv = append(argv, innerArgv...)
	return argv, nil
}
