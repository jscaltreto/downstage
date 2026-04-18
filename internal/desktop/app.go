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
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Event names for the OnBeforeClose flush handshake. See AGENTS.md.
const (
	eventBeforeClose   = "downstage:before-close"
	eventFlushComplete = "downstage:flush-complete"

	// beforeCloseTimeout bounds how long BeforeClose will wait on the
	// frontend flush. A broken frontend must not lock the window closed.
	beforeCloseTimeout = 2 * time.Second

	// Menu-click fan-out event. The Click callback on every catalog-derived
	// menu item emits this with the command ID; the frontend dispatcher
	// subscribes once and routes by ID.
	eventCommandExecute = "command:execute"
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
	SidebarWidth       int    `json:"sidebarWidth,omitempty"`       // px; 0 → frontend default 256
	LastDrawerTab      string `json:"lastDrawerTab,omitempty"`      // "" → 'issues'
}

// WindowState persists window geometry so the desktop app reopens where
// the user left it. Kept as a separate top-level Config field (not nested
// in Preferences) because geometry is written on a different cadence
// than UI prefs — live-updated on resize/move, not on user action —
// and the frontend's prefs-cache atomicity contract should not drag
// window state along for the ride.
//
// Placed is the "has the user ever moved this window?" guard. Without
// it, a stored (0, 0) is indistinguishable from "no saved position,
// default is fine" and we'd needlessly move the window on first run.
type WindowState struct {
	Width     int  `json:"width,omitempty"`
	Height    int  `json:"height,omitempty"`
	X         int  `json:"x,omitempty"`
	Y         int  `json:"y,omitempty"`
	Maximized bool `json:"maximized,omitempty"`
	Placed    bool `json:"placed,omitempty"`
}

// Config stores persistent user preferences across sessions.
type Config struct {
	LastProjectPath       string      `json:"lastProjectPath"`
	LastActiveProjectFile string      `json:"lastActiveProjectFile"`
	Preferences           Preferences `json:"preferences"`
	WindowState           WindowState `json:"windowState,omitempty"`
}

// App is the Wails application backend.
type App struct {
	ctx            context.Context
	currentProject string
	configPath     string
	configMu       sync.Mutex // guards read-modify-write of the on-disk config

	// menuMu guards currentMenu and the SetDisabledCommands rebuild path.
	// currentMenu is the most recently-applied *menu.Menu, kept so tests
	// (and future state-reflection logic) can inspect the rendered tree.
	menuMu      sync.Mutex
	currentMenu *menu.Menu
}

func NewApp() *App {
	a := &App{}
	// configPath is resolved eagerly so callers that run before OnStartup
	// — notably main.go reading WindowState to pick the initial window
	// size — can touch config. Errors here are non-fatal; readConfig
	// handles an empty configPath by returning a zero-value Config.
	if userConfigDir, err := os.UserConfigDir(); err == nil {
		a.configPath = filepath.Join(userConfigDir, "downstage", "config.json")
		_ = os.MkdirAll(filepath.Dir(a.configPath), 0755)
	}
	return a
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.initApp()
	a.restoreWindowState()
}

func (a *App) initApp() {
	config, _ := a.readConfig()

	if config.LastProjectPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			defaultPath := filepath.Join(home, "Documents", "Downstage Plays")
			_ = os.MkdirAll(defaultPath, 0755)
			config.LastProjectPath = defaultPath
			_ = a.updateConfig(func(c *Config) {
				c.LastProjectPath = defaultPath
			})
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
// exists. Takes configMu itself — safe for external callers reading
// atomically. Internal code doing a read-modify-write should use
// updateConfig instead so the full cycle is serialized.
func (a *App) readConfig() (Config, error) {
	a.configMu.Lock()
	defer a.configMu.Unlock()
	return a.readConfigLocked()
}

// readConfigLocked performs the disk read; assumes configMu is already
// held by the caller. Used by updateConfig to keep the whole RMW cycle
// under a single lock acquisition.
func (a *App) readConfigLocked() (Config, error) {
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
// Prefer updateConfig for any read-modify-write so subtree writers can't
// race.
func (a *App) writeConfig(cfg Config) error {
	a.configMu.Lock()
	defer a.configMu.Unlock()
	return a.writeConfigLocked(cfg)
}

// writeConfigLocked persists the given Config; assumes configMu is held.
func (a *App) writeConfigLocked(cfg Config) error {
	if a.configPath == "" {
		return nil
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(a.configPath, data, 0644)
}

// updateConfig is the single atomic read-modify-write path for Config.
// It holds configMu for the duration of the callback so independent
// subtree writers (Preferences, WindowState, LastActiveProjectFile) can
// interleave at the Go level without dropping each other's changes.
//
// Callers mutate only the fields they own, leaving the rest of the
// struct intact. The frontend's prefs-cache still serializes its own
// Preferences-specific writes on top of this — both layers are needed:
// Go serializes cross-subtree races; prefs-cache serializes
// cross-Preferences-writer races.
func (a *App) updateConfig(mutate func(*Config)) error {
	a.configMu.Lock()
	defer a.configMu.Unlock()
	cfg, err := a.readConfigLocked()
	if err != nil {
		return err
	}
	mutate(&cfg)
	return a.writeConfigLocked(cfg)
}

// SetActiveProjectFile persists the last-opened file for the active project.
// Called from the frontend on file selection so `ReadProjectFile` doesn't
// touch config on every read.
func (a *App) SetActiveProjectFile(rel string) error {
	return a.updateConfig(func(c *Config) {
		c.LastActiveProjectFile = rel
	})
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

// SetPreferences replaces the entire Preferences block in Config via
// updateConfig, so concurrent writes from other subtrees (WindowState,
// LastActiveProjectFile) don't get lost. Non-preference fields are
// preserved.
func (a *App) SetPreferences(prefs Preferences) error {
	return a.updateConfig(func(c *Config) {
		c.Preferences = prefs
	})
}

// minSaneWindowDimension is the lower bound for a persisted window
// size we'll honor on startup. A stored 0x0 or tiny window is almost
// certainly a bug or an OS glitch; fall back to the default in that case.
const minSaneWindowDimension = 400

// GetWindowState returns the persisted window geometry. Empty / never-
// placed returns a zero-value WindowState; main.go treats that as
// "use defaults".
func (a *App) GetWindowState() (WindowState, error) {
	cfg, err := a.readConfig()
	if err != nil {
		return WindowState{}, err
	}
	return cfg.WindowState, nil
}

// SaveWindowBounds persists the window's current Width/Height/X/Y. The
// caller is responsible for verifying that the window is unmaximized
// — saving maximized bounds would clobber the last-known "normal" size.
// SaveWindowBoundsIfNormal does that check internally; SaveWindowBounds
// is the unchecked form exposed for tests.
func (a *App) SaveWindowBounds(width, height, x, y int) error {
	return a.updateConfig(func(c *Config) {
		c.WindowState.Width = width
		c.WindowState.Height = height
		c.WindowState.X = x
		c.WindowState.Y = y
		c.WindowState.Placed = true
	})
}

// SaveWindowBoundsIfNormal reads the current geometry from the Wails
// runtime and persists it only if the window is unmaximized. Called
// from the frontend on debounced window-resize events, so we never
// overwrite the last-known normal size with a maximized screen rect.
func (a *App) SaveWindowBoundsIfNormal() error {
	if a.ctx == nil {
		return nil
	}
	if runtime.WindowIsMaximised(a.ctx) {
		return nil
	}
	w, h := runtime.WindowGetSize(a.ctx)
	x, y := runtime.WindowGetPosition(a.ctx)
	return a.SaveWindowBounds(w, h, x, y)
}

// SaveWindowMaximized flips the Maximized flag without touching the
// stored bounds. BeforeClose calls this so a maximize-then-quit cycle
// remembers "restore to maximized" while keeping the previous normal
// bounds intact for the eventual un-maximize.
func (a *App) SaveWindowMaximized(maximized bool) error {
	return a.updateConfig(func(c *Config) {
		c.WindowState.Maximized = maximized
	})
}

// restoreWindowState is called from Startup after the webview exists.
// Initial size comes through options.App (read in main.go); position
// and maximize go through the runtime here because they have no
// options counterpart.
//
// Wails v2's Screen API exposes logical-pixel sizes but not per-screen
// origins, so we can't rigorously check "is this point on a display".
// The OS window manager clamps absurd coordinates to the nearest
// display on restore, which is an acceptable fallback.
func (a *App) restoreWindowState() {
	if a.ctx == nil {
		return
	}
	ws, err := a.GetWindowState()
	if err != nil || !ws.Placed {
		return
	}
	runtime.WindowSetPosition(a.ctx, ws.X, ws.Y)
	if ws.Maximized {
		runtime.WindowMaximise(a.ctx)
	}
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

// SetInitialMenu is called from main() after BuildMenu produces the
// startup menu. This lets SetDisabledCommands see the current tree even
// before any user-driven state change. No lock is needed because main()
// calls this strictly before wails.Run, but we take the lock anyway to
// keep invariants uniform.
func (a *App) SetInitialMenu(m *menu.Menu) {
	a.menuMu.Lock()
	defer a.menuMu.Unlock()
	a.currentMenu = m
}

// GetCommands returns the palette-facing projection of the catalog.
// The frontend calls this once on palette open to render labels and
// categories. Handlers are keyed by ID on the frontend; metadata stays
// authoritative here.
func (a *App) GetCommands() []CommandMeta {
	cmds := Commands()
	out := make([]CommandMeta, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, CommandMeta{
			ID:            c.ID,
			Label:         c.Label,
			Category:      string(c.Category),
			Accelerator:   c.Accelerator,
			PaletteHidden: c.PaletteHidden,
		})
	}
	return out
}

// SetDisabledCommands rebuilds the native menu with the listed IDs
// flagged as Disabled, and applies the new menu via
// runtime.MenuUpdateApplicationMenu. The frontend dispatcher calls this
// after diff-and-skip so the wire is quiet when the disabled set is
// stable across reactive blips.
//
// Ordering: the disabled-set diff happens on the frontend, so a no-op
// call here means something genuinely changed. Still cheap enough to
// rebuild unconditionally.
func (a *App) SetDisabledCommands(ids []string) error {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}

	a.menuMu.Lock()
	defer a.menuMu.Unlock()

	next := BuildMenu(a, set)
	a.currentMenu = next
	if a.ctx != nil {
		runtime.MenuUpdateApplicationMenu(a.ctx)
	}
	return nil
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
	// Capture window geometry before the flush handshake. Reads are
	// synchronous and fit comfortably inside the beforeCloseTimeout
	// budget. Doing it first guarantees geometry is persisted even if
	// the frontend flush stalls and we time out.
	_ = a.SaveWindowBoundsIfNormal()
	_ = a.SaveWindowMaximized(runtime.WindowIsMaximised(ctx))

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
