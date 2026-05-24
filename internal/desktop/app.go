package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventBeforeClose    = "downstage:before-close"
	eventFlushComplete  = "downstage:flush-complete"
	beforeCloseTimeout  = 2 * time.Second
	eventCommandExecute = "command:execute"
)

// Preferences stores persisted desktop UI settings.
type Preferences struct {
	Theme               string `json:"theme,omitempty"`
	PreviewHidden       bool   `json:"previewHidden,omitempty"`
	SpellcheckDisabled  bool   `json:"spellcheckDisabled,omitempty"`
	SidebarCollapsed    bool   `json:"sidebarCollapsed,omitempty"`
	SidebarWidth        int    `json:"sidebarWidth,omitempty"`
	LastDrawerTab       string `json:"lastDrawerTab,omitempty"`
	DrawerDock          string `json:"drawerDock,omitempty"`
	DrawerRightWidth    int    `json:"drawerRightWidth,omitempty"`
	ExportPageSize      string `json:"exportPageSize,omitempty"`      // "letter" (default) | "a4"
	ExportStyle         string `json:"exportStyle,omitempty"`         // "standard" (default) | "condensed"
	ExportLayout        string `json:"exportLayout,omitempty"`        // "single" (default) | "2up" | "booklet"
	ExportBookletGutter string `json:"exportBookletGutter,omitempty"` // e.g. "0.125in"
}

// WindowState stores the last normal window size.
type WindowState struct {
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

// Config is the on-disk desktop config.
type Config struct {
	LastLibraryPath       string      `json:"lastLibraryPath"`
	LastActiveLibraryFile string      `json:"lastActiveLibraryFile"`
	Preferences           Preferences `json:"preferences"`
	WindowState           WindowState `json:"windowState,omitempty"`
}

// legacyConfig matches the pre-rename on-disk shape.
type legacyConfig struct {
	LastProjectPath       string `json:"lastProjectPath,omitempty"`
	LastActiveProjectFile string `json:"lastActiveProjectFile,omitempty"`
}

// App is the Wails application backend.
type App struct {
	ctx context.Context

	// libMu guards currentLibrary. Wails dispatches each bound method on
	// its own goroutine; without the lock a concurrent ChangeLibraryLocation
	// races against every reader. Writers (initApp, ChangeLibraryLocation)
	// hold Lock only around the field mutation and the immediately
	// following git-init — never while a user-facing dialog is open.
	// Readers hold RLock for the duration of the method to keep
	// currentLibrary consistent across the method's I/O.
	libMu          sync.RWMutex
	currentLibrary string

	configPath string
	configMu   sync.Mutex

	menuMu      sync.Mutex
	currentMenu *menu.Menu

	hiddenMu sync.Mutex
}

func NewApp() *App {
	a := &App{}
	if userConfigDir, err := os.UserConfigDir(); err == nil {
		a.configPath = filepath.Join(userConfigDir, "downstage", "config.json")
		_ = os.MkdirAll(filepath.Dir(a.configPath), 0755)
	}
	return a
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.initApp()
}

func (a *App) initApp() {
	config, _ := a.readConfig()

	if config.LastLibraryPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			defaultPath := filepath.Join(home, "Documents", "Downstage Plays")
			_ = os.MkdirAll(defaultPath, 0755)
			config.LastLibraryPath = defaultPath
			_ = a.updateConfig(func(c *Config) {
				c.LastLibraryPath = defaultPath
			})
		}
	}

	if config.LastLibraryPath != "" {
		if _, err := os.Stat(config.LastLibraryPath); err == nil {
			// Startup runs before any RPC can fire, so contention is
			// theoretical; lock for uniformity. ensureGitRepo takes
			// its own RLock, so swap fields under Lock and release
			// before calling it.
			a.libMu.Lock()
			a.currentLibrary = config.LastLibraryPath
			a.libMu.Unlock()
			_ = a.ensureGitRepo()
		}
	}
}

func (a *App) ensureGitRepo() error {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	if a.currentLibrary == "" {
		return nil
	}
	_, err := git.PlainOpen(a.currentLibrary)
	if err == git.ErrRepositoryNotExists {
		_, err = git.PlainInit(a.currentLibrary, false)
	}
	return err
}

func (a *App) readConfig() (Config, error) {
	a.configMu.Lock()
	defer a.configMu.Unlock()
	return a.readConfigLocked()
}

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
		a.backupCorruptConfig("read")
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		a.backupCorruptConfig("parse")
		return Config{}, nil
	}
	migrateLegacyConfig(&cfg, data)
	return cfg, nil
}

func (a *App) backupCorruptConfig(cause string) {
	if a.configPath == "" {
		return
	}
	bak := fmt.Sprintf("%s.bak.%d", a.configPath, time.Now().Unix())
	if err := os.Rename(a.configPath, bak); err != nil {
		slog.Warn("config: backup of unreadable file failed",
			"cause", cause, "path", a.configPath, "err", err)
		return
	}
	slog.Warn("config: backed up unreadable file; starting with defaults",
		"cause", cause, "from", a.configPath, "to", bak)
}

func migrateLegacyConfig(cfg *Config, raw []byte) {
	if cfg.LastLibraryPath != "" && cfg.LastActiveLibraryFile != "" {
		return
	}
	var legacy legacyConfig
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return
	}
	if cfg.LastLibraryPath == "" && legacy.LastProjectPath != "" {
		cfg.LastLibraryPath = legacy.LastProjectPath
	}
	if cfg.LastActiveLibraryFile == "" && legacy.LastActiveProjectFile != "" {
		cfg.LastActiveLibraryFile = legacy.LastActiveProjectFile
	}
}

func (a *App) writeConfig(cfg Config) error {
	a.configMu.Lock()
	defer a.configMu.Unlock()
	return a.writeConfigLocked(cfg)
}

func (a *App) writeConfigLocked(cfg Config) error {
	if a.configPath == "" {
		return nil
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	tmp := a.configPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, a.configPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

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

func (a *App) SetActiveLibraryFile(rel string) error {
	return a.updateConfig(func(c *Config) {
		c.LastActiveLibraryFile = rel
	})
}

const defaultTheme = "system"

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

func (a *App) SetPreferences(prefs Preferences) error {
	return a.updateConfig(func(c *Config) {
		c.Preferences = prefs
	})
}

const minSaneWindowDimension = 400

func (a *App) GetWindowState() (WindowState, error) {
	cfg, err := a.readConfig()
	if err != nil {
		return WindowState{}, err
	}
	return cfg.WindowState, nil
}

func (a *App) SaveWindowBounds(width, height int) error {
	return a.updateConfig(func(c *Config) {
		c.WindowState.Width = width
		c.WindowState.Height = height
	})
}

func (a *App) SaveWindowBoundsIfNormal() error {
	if a.ctx == nil {
		return nil
	}
	if runtime.WindowIsMaximised(a.ctx) {
		return nil
	}
	w, h := runtime.WindowGetSize(a.ctx)
	return a.SaveWindowBounds(w, h)
}

// safePath validates relPath against the current library root. CALLER
// MUST HOLD a.libMu (read or write); we do NOT lock here because every
// caller is a bound App method that already holds RLock for the
// duration of its body, and Go's sync.RWMutex doesn't safely recurse
// when a writer is pending.
func (a *App) safePath(relPath string) (string, error) {
	if a.currentLibrary == "" {
		return "", fmt.Errorf("no library open")
	}
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	relPath = filepath.Clean(relPath)

	root, err := filepath.EvalSymlinks(a.currentLibrary)
	if err != nil {
		return "", fmt.Errorf("resolving library root: %w", err)
	}
	joined := filepath.Join(root, relPath)

	if info, lerr := os.Lstat(joined); lerr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("path escapes library root: leaf symlinks are not allowed")
		}
	} else if !errors.Is(lerr, fs.ErrNotExist) {
		return "", fmt.Errorf("stat path: %w", lerr)
	}

	target, err := filepath.EvalSymlinks(joined)
	if err != nil {
		parent := filepath.Dir(joined)
		resolvedParent, perr := filepath.EvalSymlinks(parent)
		if perr != nil {
			return "", fmt.Errorf("resolving parent: %w", perr)
		}
		if !pathInsideRoot(resolvedParent, root) {
			return "", fmt.Errorf("path escapes library root")
		}
		return joined, nil
	}
	if !pathInsideRoot(target, root) {
		return "", fmt.Errorf("path escapes library root")
	}
	return target, nil
}

func pathInsideRoot(p, root string) bool {
	if p == root {
		return true
	}
	return strings.HasPrefix(p+string(filepath.Separator), root+string(filepath.Separator))
}

func (a *App) GetCurrentLibrary() string {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	return a.currentLibrary
}

func (a *App) GetLastActiveFile() string {
	cfg, _ := a.readConfig()
	return cfg.LastActiveLibraryFile
}

func (a *App) BrowserOpenURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

func (a *App) RevealLibraryInExplorer() error {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	if a.currentLibrary == "" {
		return fmt.Errorf("no library open")
	}
	if _, err := os.Stat(a.currentLibrary); err != nil {
		return fmt.Errorf("library path unavailable: %w", err)
	}

	switch goruntime.GOOS {
	case "darwin":
		return detachedExec(exec.Command("open", a.currentLibrary))
	case "windows":
		return detachedExec(exec.Command("explorer", a.currentLibrary))
	case "linux":
		return revealOnLinux(a.currentLibrary)
	default:
		return fmt.Errorf("reveal not supported on %s", goruntime.GOOS)
	}
}

func revealOnLinux(path string) error {
	uri := "file://" + path
	dbus := exec.Command(
		"dbus-send",
		"--session",
		"--dest=org.freedesktop.FileManager1",
		"--type=method_call",
		"/org/freedesktop/FileManager1",
		"org.freedesktop.FileManager1.ShowFolders",
		"array:string:"+uri,
		"string:",
	)
	if err := dbus.Run(); err == nil {
		return nil
	}
	return detachedExec(exec.Command("xdg-open", path))
}

func detachedExec(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("reveal failed: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func (a *App) ShowAboutDialog() error {
	if a.ctx == nil {
		return nil
	}
	_, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "About Downstage Write",
		Message: fmt.Sprintf("Downstage Write\nVersion %s — Alpha preview\nhttps://getdownstage.com", Version),
		Buttons: []string{"OK"},
	})
	return err
}

func (a *App) SetInitialMenu(m *menu.Menu) {
	a.menuMu.Lock()
	defer a.menuMu.Unlock()
	a.currentMenu = m
}

func (a *App) GetCommands() []CommandMeta {
	cmds := Commands()
	out := make([]CommandMeta, 0, len(cmds))
	for _, c := range cmds {
		if !c.PlatformAllows(goruntime.GOOS) {
			continue
		}
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

func (a *App) Quit() {
	runtime.Quit(a.ctx)
}

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

func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	_ = a.SaveWindowBoundsIfNormal()

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
