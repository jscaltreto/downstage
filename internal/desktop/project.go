package desktop

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const dictionaryPath = ".downstage/dictionary.txt"

// ProjectFile represents a .ds file in the project directory.
type ProjectFile struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updatedAt"`
}

func (a *App) OpenProjectFolder() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Project Folder",
	})
	if err != nil {
		return "", err
	}
	if selection == "" {
		return "", nil
	}
	a.currentProject = selection

	// Switch projects: update only the project fields. Prior to the
	// updateConfig migration this path wrote Config{...} wholesale,
	// silently zeroing Preferences and WindowState every time the user
	// changed folders. Now other subtrees are preserved.
	if err := a.updateConfig(func(c *Config) {
		c.LastProjectPath = selection
		c.LastActiveProjectFile = ""
	}); err != nil {
		slog.Warn("persisting project switch failed", "err", err)
	}

	_ = a.ensureGitRepo()
	return selection, nil
}

func (a *App) GetProjectFiles() ([]ProjectFile, error) {
	if a.currentProject == "" {
		return nil, nil
	}

	var files []ProjectFile
	walkErr := filepath.WalkDir(a.currentProject, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// A single unreadable dir (permissions, IO) should not abort
			// the whole listing — log and continue.
			slog.Warn("project walk: skipping", "path", path, "err", err)
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if name := d.Name(); name == ".git" || name == ".downstage" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".ds") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			slog.Warn("project walk: stat failed", "path", path, "err", err)
			return nil
		}
		rel, err := filepath.Rel(a.currentProject, path)
		if err != nil {
			return err
		}
		files = append(files, ProjectFile{
			Path:      rel,
			Name:      info.Name(),
			UpdatedAt: info.ModTime().Format(time.RFC3339),
		})
		return nil
	})
	if walkErr != nil {
		return files, walkErr
	}

	sort.SliceStable(files, func(i, j int) bool {
		return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path)
	})
	return files, nil
}

func (a *App) ReadProjectFile(relPath string) (string, error) {
	fullPath, err := a.safePath(relPath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *App) WriteProjectFile(relPath string, content string) error {
	fullPath, err := a.safePath(relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}

func (a *App) CreateProjectFile(name string, content string) (string, error) {
	if a.currentProject == "" {
		return "", fmt.Errorf("no project open")
	}

	filename := name
	if !strings.HasSuffix(strings.ToLower(filename), ".ds") {
		filename += ".ds"
	}

	base := strings.TrimSuffix(filename, ".ds")
	finalName := filename
	counter := 1
	for {
		fullPath, err := a.safePath(finalName)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			break
		}
		finalName = fmt.Sprintf("%s-%d.ds", base, counter)
		counter++
	}

	fullPath, err := a.safePath(finalName)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", err
	}
	return finalName, nil
}

func (a *App) dictionaryFile() string {
	if a.currentProject == "" {
		return ""
	}
	return filepath.Join(a.currentProject, dictionaryPath)
}

func (a *App) readDictionary() []string {
	path := a.dictionaryFile()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var words []string
	for _, line := range strings.Split(string(data), "\n") {
		w := strings.TrimSpace(line)
		if w != "" {
			words = append(words, w)
		}
	}
	return words
}

func (a *App) writeDictionary(words []string) error {
	path := a.dictionaryFile()
	if path == "" {
		return fmt.Errorf("no project open")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	sort.Strings(words)
	return os.WriteFile(path, []byte(strings.Join(words, "\n")+"\n"), 0644)
}

func (a *App) GetSpellAllowlist() []string {
	words := a.readDictionary()
	if words == nil {
		return []string{}
	}
	return words
}

func (a *App) AddSpellAllowlistWord(word string) (bool, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return false, nil
	}
	words := a.readDictionary()
	key := strings.ToLower(word)
	for _, existing := range words {
		if strings.ToLower(existing) == key {
			return false, nil
		}
	}
	words = append(words, word)
	if err := a.writeDictionary(words); err != nil {
		return false, err
	}
	return true, nil
}

func (a *App) RemoveSpellAllowlistWord(word string) (bool, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return false, nil
	}
	words := a.readDictionary()
	key := strings.ToLower(word)
	var next []string
	found := false
	for _, existing := range words {
		if strings.ToLower(existing) == key {
			found = true
		} else {
			next = append(next, existing)
		}
	}
	if !found {
		return false, nil
	}
	if err := a.writeDictionary(next); err != nil {
		return false, err
	}
	return true, nil
}
