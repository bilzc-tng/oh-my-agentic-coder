package sandboxprofile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// boolPtr is a tiny helper for NetworkPrompt.Enabled.
func boolPtr(b bool) *bool { return &b }

// builtinProfiles returns the compiled-in profiles. "default" mirrors
// the tng-sandbox nono profile that omac shipped before the built-in
// sandbox existed (opencode-nono/tng-sandbox.json), minus the dropped
// credential-injection block.
func builtinProfiles() map[string]*Profile {
	return map[string]*Profile{
		"default": {
			Meta:    Meta{Name: "default"},
			Workdir: Workdir{Access: AccessReadWrite},
			Filesystem: Filesystem{
				Allow: []string{
					"~/.local/share/opencode",
					"~/.local/state/opencode",
					"~/.claude",
					"~/.cache",
					"~/Library/Caches",
					"~/go",
					"~/.rustup",
					"~/.cargo",
				},
				Read: []string{
					"~/.config/opencode",
					"~/.opencode/bin",
					"~/.nvm",
					"~/.gitconfig",
					"~/.gitignore_global",
					"~/.claude.json",
				},
			},
			Network: Network{
				Mode:            ModeFiltered,
				ListenPort:      []int{4097},
				AllowTCPConnect: []int{22},
				NetworkPrompt: &NetworkPrompt{
					Enabled:           boolPtr(true),
					PromptTimeoutSecs: DefaultPromptTimeoutSecs,
					OnUnavailable:     OnUnavailableDeny,
				},
			},
		},
	}
}

// UserProfileDir returns ~/.config/omac/profiles.
func UserProfileDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "omac", "profiles"), nil
}

// Resolve loads a profile reference:
//   - a path (contains a separator or ends in .json): load that file;
//   - otherwise ~/.config/omac/profiles/<ref>.json if it exists;
//   - otherwise a compiled-in profile of that name.
//
// An empty ref means "default".
func Resolve(ref string) (*Profile, error) {
	if ref == "" {
		ref = "default"
	}
	if strings.ContainsRune(ref, os.PathSeparator) || strings.HasSuffix(ref, ".json") {
		return loadFile(ref)
	}
	var searched []string
	if dir, err := UserProfileDir(); err == nil {
		p := filepath.Join(dir, ref+".json")
		searched = append(searched, p)
		if _, statErr := os.Stat(p); statErr == nil {
			return loadFile(p)
		}
	}
	if bp, ok := builtinProfiles()[ref]; ok {
		return bp, nil
	}
	return nil, fmt.Errorf("sandbox profile %q not found (searched: %s, builtin profiles: %s)",
		ref, strings.Join(searched, ", "), strings.Join(builtinNames(), ", "))
}

func loadFile(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sandbox profile %s: %w", path, err)
	}
	p, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return p, nil
}

func builtinNames() []string {
	var names []string
	for n := range builtinProfiles() {
		names = append(names, n)
	}
	return names
}
