package desktop

import (
	goruntime "runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wailsapp/wails/v2/pkg/menu"
)

func TestBuildMenu_PlacesCatalogItemsUnderCorrectTopLevels(t *testing.T) {
	app := NewApp()
	m := BuildMenu(app, nil)
	require.NotNil(t, m)

	topLevels := map[string]map[string]bool{}
	for _, item := range m.Items {
		if item.Type != menu.SubmenuType || item.SubMenu == nil {
			continue
		}
		labels := map[string]bool{}
		collectLabels(item.SubMenu, labels)
		topLevels[item.Label] = labels
	}

	for _, cmd := range Commands() {
		if len(cmd.MenuPath) == 0 {
			continue
		}
		if !cmd.PlatformAllows(goruntime.GOOS) {
			continue
		}
		top := cmd.MenuPath[0]
		labels, ok := topLevels[top]
		require.True(t, ok, "top-level menu %q missing", top)
		assert.True(t, labels[cmd.Label], "command %q (label %q) not under %q submenu", cmd.ID, cmd.Label, top)
	}
}

func TestBuildMenu_FiltersLinuxOnlyAndNestsExportSubmenu(t *testing.T) {
	app := NewApp()

	labelsIn := func(m *menu.Menu, topName string) map[string]bool {
		got := map[string]bool{}
		for _, item := range m.Items {
			if item.Label == topName && item.SubMenu != nil {
				collectLabels(item.SubMenu, got)
				return got
			}
		}
		return got
	}

	linuxMenu := buildMenuForGOOS(app, nil, "linux")
	linuxEdit := labelsIn(linuxMenu, "Edit")
	assert.True(t, linuxEdit["Undo"], "Linux Edit should include explicit Undo")
	assert.True(t, linuxEdit["Cut"], "Linux Edit should include explicit Cut")
	linuxFile := labelsIn(linuxMenu, "File")
	assert.True(t, linuxFile["Quit Downstage Write"], "Linux File should include Quit")
	assert.True(t, linuxFile["PDF…"], "Export > PDF… must be walked via sub-submenu")

	darwinMenu := buildMenuForGOOS(app, nil, "darwin")
	darwinFile := labelsIn(darwinMenu, "File")
	assert.False(t, darwinFile["Quit Downstage Write"], "macOS AppMenu role provides Quit; catalog entry must be filtered")

	windowsMenu := buildMenuForGOOS(app, nil, "windows")
	windowsFile := labelsIn(windowsMenu, "File")
	assert.True(t, windowsFile["Quit Downstage Write"], "Windows File should include Quit")
}

func TestSetDisabledCommands_RebuildsMenuWithDisabledFlags(t *testing.T) {
	app := NewApp()
	initial := BuildMenu(app, nil)
	app.SetInitialMenu(initial)

	require.NoError(t, app.SetDisabledCommands([]string{"file.exportPdf", "file.saveVersion"}))

	var file *menu.Menu
	for _, item := range app.currentMenu.Items {
		if item.Label == "File" && item.SubMenu != nil {
			file = item.SubMenu
			break
		}
	}
	require.NotNil(t, file, "File submenu missing after rebuild")

	disabledLabels := collectDisabledLabels(file)
	assert.True(t, disabledLabels["PDF…"], "Export > PDF… should be Disabled")
	assert.True(t, disabledLabels["Save Version…"], "Save Version… should be Disabled")
}

func collectDisabledLabels(m *menu.Menu) map[string]bool {
	out := map[string]bool{}
	var walk func(*menu.Menu)
	walk = func(sub *menu.Menu) {
		for _, item := range sub.Items {
			if item.Type == menu.TextType && item.Disabled {
				out[item.Label] = true
			}
			if item.SubMenu != nil {
				walk(item.SubMenu)
			}
		}
	}
	walk(m)
	return out
}

func TestGetCommands_ReturnsPlatformFilteredCatalogProjection(t *testing.T) {
	app := NewApp()
	metas := app.GetCommands()

	want := 0
	for _, cmd := range Commands() {
		if cmd.PlatformAllows(goruntime.GOOS) {
			want++
		}
	}
	assert.Equal(t, want, len(metas), "GetCommands must mirror platform-filtered Commands length")

	ids := map[string]CommandMeta{}
	for _, m := range metas {
		ids[m.ID] = m
	}
	require.Contains(t, ids, CmdFileNewPlay)
	require.Contains(t, ids, CmdFileSettingsSpellcheck)

	assert.Equal(t, "New Play", ids[CmdFileNewPlay].Label)
	assert.True(t, ids[CmdFileSettingsSpellcheck].PaletteHidden)
	assert.False(t, ids[CmdFileNewPlay].PaletteHidden)

	if goruntime.GOOS == "linux" {
		assert.Contains(t, ids, CmdEditUndo)
		assert.Contains(t, ids, CmdFileQuit)
	} else {
		assert.NotContains(t, ids, CmdEditUndo, "edit.undo is Linux-only")
	}
	if goruntime.GOOS == "darwin" {
		assert.NotContains(t, ids, CmdFileQuit, "file.quit is filtered on darwin (AppMenu role covers it)")
	}
}

func collectLabels(m *menu.Menu, out map[string]bool) {
	for _, item := range m.Items {
		if item.Type == menu.TextType && item.Label != "" {
			out[item.Label] = true
		}
		if item.SubMenu != nil {
			collectLabels(item.SubMenu, out)
		}
	}
}
