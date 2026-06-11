//go:build !linux

package sandboxrun

import "fmt"

// RunStage2 only exists on Linux (it is exec'd inside bwrap).
func RunStage2(_ []string) error {
	return fmt.Errorf("omac sandbox stage2 is Linux-only")
}
