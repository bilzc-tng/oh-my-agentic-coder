package sandboxprofile

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath performs ~ and $VAR / ${VAR} expansion on a single path
// entry, evaluated in the supervisor's environment, and absolutizes
// the result. It does not require the path to exist.
func ExpandPath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("empty path")
	}
	if p == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand %q: %w", p, err)
		}
		p = home
	} else if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand %q: %w", p, err)
		}
		p = filepath.Join(home, p[2:])
	}
	p = os.Expand(p, func(name string) string {
		return os.Getenv(name)
	})
	if p == "" {
		return "", fmt.Errorf("path expands to empty string")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("absolutize %q: %w", p, err)
	}
	return abs, nil
}

// ExpandExisting expands every entry of paths and returns those that
// exist on disk. Entries that expand but do not exist are skipped with
// a notice on w (nono behaviour: missing grant targets are not fatal).
// Entries that fail to expand produce an error.
func ExpandExisting(paths []string, w io.Writer) ([]string, error) {
	var out []string
	for _, raw := range paths {
		p, err := ExpandPath(raw)
		if err != nil {
			return nil, fmt.Errorf("filesystem path %q: %w", raw, err)
		}
		if _, statErr := os.Lstat(p); statErr != nil {
			if os.IsNotExist(statErr) {
				if w != nil {
					fmt.Fprintf(w, "omac sandbox: notice: skipping nonexistent path %s\n", p)
				}
				continue
			}
			return nil, fmt.Errorf("filesystem path %q: %w", raw, statErr)
		}
		out = append(out, p)
	}
	return out, nil
}
