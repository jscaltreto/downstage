package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Event names for the OnBeforeClose flush handshake. See AGENTS.md.
const (
	eventBeforeClose   = "downstage:before-close"
	eventFlushComplete = "downstage:flush-complete"

	// beforeCloseTimeout bounds how long BeforeClose will wait on the
	// frontend flush. A broken frontend must not lock the window closed.
	beforeCloseTimeout = 2 * time.Second
)

// Preferences captures every persisted UI preference the desktop app
// exposes. It lives inside Config as a single nested struct so the
// frontend can round-trip the whole thing as one typed unit. Fields use
// negative-semantic booleans (hidden/disabled/collapsed) so the JSON
// zero value aligns with the default behavior — no pointer gymnastics,
// no config version tag required.
type Preferences struct {
	Theme              string `json:"theme,omitempty"`              // "", "light", "dark", "system"
	PreviewHidden      bool   `json:"previewHidden,omitempty"`      // default false (visible)
	SpellcheckDisabled bool   `json:"spellcheckDisabled,omitempty"` // default false (enabled)
	SidebarCollapsed   bool   `json:"sidebarCollapsed,omitempty"`   // default false (expanded)
}

// Config stores persistent user preferences across sessions.
type Config struct {
	LastProjectPath       string      `json:"lastProjectPath"`
	LastActiveProjectFile string      `json:"lastActiveProjectFile"`
	Preferences           Preferences `json:"preferences"`
}

// App is the Wails application backend.
type App struct {
	ctx            context.Context
	currentProject string
	configPath     string
	configMu       sync.Mutex // guards read-modify-write of the on-disk config
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.initApp()
}

func (a *App) initApp() {
	userConfigDir, err := os.UserConfigDir()
	if err == nil {
		a.configPath = filepath.Join(userConfigDir, "downstage", "config.json")
		_ = os.MkdirAll(filepath.Dir(a.configPath), 0755)
	}

	config, _ := a.readConfig()

	if config.LastProjectPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			defaultPath := filepath.Join(home, "Documents", "Downstage Plays")
			_ = os.MkdirAll(defaultPath, 0755)
			config.LastProjectPath = defaultPath
			_ = a.writeConfig(config)
		}
	}

	if config.LastProjectPath != "" {
		if _, err := os.Stat(config.LastProjectPath); err == nil {
			a.currentProject = config.LastProjectPath
			_ = a.ensureGitRepo()
		}
	}
}

func (a *App) ensureGitRepo() error {
	if a.currentProject == "" {
		return nil
	}
	_, err := git.PlainOpen(a.currentProject)
	if err == git.ErrRepositoryNotExists {
		_, err = git.PlainInit(a.currentProject, false)
	}
	return err
}

// readConfig returns the on-disk config or a zero-value Config if none
// exists. It is safe to call concurrently with writeConfig (both take the
// mutex). Errors are returned rather than swallowed so callers can decide
// how to react — initApp ignores them deliberately (first run).
func (a *App) readConfig() (Config, error) {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	var cfg Config
	if a.configPath == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// writeConfig persists the given Config verbatim. This is an EXPLICIT write
// — callers are expected to read first, mutate fields they own, then write.
// There is deliberately no asymmetric merge here (the old saveConfig could
// not clear a field because empty values were skipped, which broke the
// project-switch case).
func (a *App) writeConfig(cfg Config) error {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	if a.configPath == "" {
		return nil
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(a.configPath, data, 0644)
}

// SetActiveProjectFile persists the last-opened file for the active project.
// Called from the frontend on file selection so `ReadProjectFile` doesn't
// touch config on every read.
func (a *App) SetActiveProjectFile(rel string) error {
	cfg, err := a.readConfig()
	if err != nil {
		return err
	}
	cfg.LastActiveProjectFile = rel
	return a.writeConfig(cfg)
}

// defaultTheme is applied by GetPreferences when the persisted theme is
// the zero value (""). "system" means "follow OS preference" which is what
// a brand-new install should do before the user picks.
const defaultTheme = "system"

// GetPreferences returns the persisted preferences with defaults applied.
// Unknown/empty Theme is normalized to "system" so the frontend never has
// to know which fields carry a sentinel.
func (a *App) GetPreferences() (Preferences, error) {
	cfg, err := a.readConfig()
	if err != nil {
		return Preferences{}, err
	}
	prefs := cfg.Preferences
	if prefs.Theme == "" {
		prefs.Theme = defaultTheme
	}
	return prefs, nil
}

// SetPreferences replaces the entire Preferences block in Config. The
// read-modify-write cycle is serialized by configMu (inside the
// readConfig/writeConfig pair), so concurrent SetPreferences calls can't
// interleave and lose fields. Non-preference Config fields
// (LastProjectPath, LastActiveProjectFile) are preserved.
func (a *App) SetPreferences(prefs Preferences) error {
	cfg, err := a.readConfig()
	if err != nil {
		return err
	}
	cfg.Preferences = prefs
	return a.writeConfig(cfg)
}

// safePath validates that relPath resolves to a location inside the project
// root. It rejects:
//   - absolute inputs (writers always work relative to the project)
//   - any leaf symlink, whether its target exists, is dangling, or points
//     inside or outside the project (writers don't need leaf symlinks, and
//     allowing them opens a class of TOCTOU / dangling-leaf bypasses)
//   - live symlink chains whose final target escapes the project root
func (a *App) safePath(relPath string) (string, error) {
	if a.currentProject == "" {
		return "", fmt.Errorf("no project open")
	}
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	relPath = filepath.Clean(relPath)

	root, err := filepath.EvalSymlinks(a.currentProject)
	if err != nil {
		return "", fmt.Errorf("resolving project root: %w", err)
	}
	joined := filepath.Join(root, relPath)

	// If the target is a leaf symlink (live or dangling), reject outright.
	// os.Lstat only errors with non-ENOENT when something is genuinely wrong
	// (permission, IO); propagate those.
	if info, lerr := os.Lstat(joined); lerr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("path escapes project root: leaf symlinks are not allowed")
		}
	} else if !errors.Is(lerr, fs.ErrNotExist) {
		return "", fmt.Errorf("stat path: %w", lerr)
	}

	target, err := filepath.EvalSymlinks(joined)
	if err != nil {
		// Path does not exist yet (a new file). Parent must resolve inside
		// the project root.
		parent := filepath.Dir(joined)
		resolvedParent, perr := filepath.EvalSymlinks(parent)
		if perr != nil {
			return "", fmt.Errorf("resolving parent: %w", perr)
		}
		if !pathInsideRoot(resolvedParent, root) {
			return "", fmt.Errorf("path escapes project root")
		}
		return joined, nil
	}
	if !pathInsideRoot(target, root) {
		return "", fmt.Errorf("path escapes project root")
	}
	return target, nil
}

// pathInsideRoot reports whether p is root itself or is nested inside root.
// Both arguments must already be cleaned and, ideally, symlink-resolved.
func pathInsideRoot(p, root string) bool {
	if p == root {
		return true
	}
	return strings.HasPrefix(p+string(filepath.Separator), root+string(filepath.Separator))
}

func (a *App) GetCurrentProject() string {
	return a.currentProject
}

func (a *App) GetLastActiveFile() string {
	cfg, _ := a.readConfig()
	return cfg.LastActiveProjectFile
}

func (a *App) BrowserOpenURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

// BeforeClose is the Wails OnBeforeClose hook. It emits a flush-request
// event to the frontend, waits for the frontend's acknowledgement, and then
// allows the window to close. If the frontend doesn't reply within
// beforeCloseTimeout (broken frontend, no active listener, etc.), the close
// proceeds rather than hang the user's window.
//
// Ordering note: the one-shot listener must be registered BEFORE emitting
// the request, otherwise a fast frontend can reply before we subscribe.
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	done := make(chan struct{})
	runtime.EventsOnce(ctx, eventFlushComplete, func(_ ...interface{}) {
		close(done)
	})
	runtime.EventsEmit(ctx, eventBeforeClose)

	select {
	case <-done:
	case <-time.After(beforeCloseTimeout):
	}
	return false
}
