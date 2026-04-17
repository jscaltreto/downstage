package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Config stores persistent user preferences across sessions.
type Config struct {
	LastProjectPath       string `json:"lastProjectPath"`
	LastActiveProjectFile string `json:"lastActiveProjectFile"`
}

// App is the Wails application backend.
type App struct {
	ctx            context.Context
	currentProject string
	configPath     string
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

	var config Config
	if a.configPath != "" {
		data, err := os.ReadFile(a.configPath)
		if err == nil {
			_ = json.Unmarshal(data, &config)
		}
	}

	if config.LastProjectPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			defaultPath := filepath.Join(home, "Documents", "Downstage Plays")
			_ = os.MkdirAll(defaultPath, 0755)
			config.LastProjectPath = defaultPath
			a.saveConfig(config)
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

func (a *App) saveConfig(config Config) {
	if a.configPath == "" {
		return
	}

	var current Config
	data, err := os.ReadFile(a.configPath)
	if err == nil {
		_ = json.Unmarshal(data, &current)
	}

	if config.LastProjectPath != "" {
		current.LastProjectPath = config.LastProjectPath
	}
	if config.LastActiveProjectFile != "" {
		current.LastActiveProjectFile = config.LastActiveProjectFile
	}

	data, err = json.Marshal(current)
	if err == nil {
		_ = os.WriteFile(a.configPath, data, 0644)
	}
}

// safePath validates that relPath resolves to a location inside the project
// root, following symlinks on both sides to prevent escapes.
func (a *App) safePath(relPath string) (string, error) {
	if a.currentProject == "" {
		return "", fmt.Errorf("no project open")
	}
	root, err := filepath.EvalSymlinks(a.currentProject)
	if err != nil {
		return "", fmt.Errorf("resolving project root: %w", err)
	}
	joined := filepath.Join(root, relPath)
	target, err := filepath.EvalSymlinks(joined)
	if err != nil {
		parent := filepath.Dir(joined)
		resolvedParent, err2 := filepath.EvalSymlinks(parent)
		if err2 != nil {
			return "", fmt.Errorf("resolving parent: %w", err2)
		}
		if !strings.HasPrefix(resolvedParent+string(filepath.Separator), root+string(filepath.Separator)) {
			return "", fmt.Errorf("path escapes project root")
		}
		return joined, nil
	}
	if !strings.HasPrefix(target+string(filepath.Separator), root+string(filepath.Separator)) && target != root {
		return "", fmt.Errorf("path escapes project root")
	}
	return target, nil
}

func (a *App) GetCurrentProject() string {
	return a.currentProject
}

func (a *App) GetLastActiveFile() string {
	if a.configPath == "" {
		return ""
	}
	var config Config
	data, err := os.ReadFile(a.configPath)
	if err == nil {
		_ = json.Unmarshal(data, &config)
	}
	return config.LastActiveProjectFile
}

func (a *App) BrowserOpenURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}
