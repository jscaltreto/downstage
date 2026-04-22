package desktop

import (
	goruntime "runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wailsapp/wails/v2/pkg/menu"
)

// BuildMenu should produce a tree whose leaves correspond to the
// catalog's non-empty MenuPath entries. Walk the tree, collect IDs
// encountered via the Click callbacks that emitCommand closed over, and
// compare against the catalog.
//
// Because the Click callback is an opaque func, we verify existence by
// walking the tree and confirming the expected LABELS appear under the
// expected top-level groupings. That's sufficient coverage for
// "BuildMenu wired the catalog in correctly" without poking at
// unexported runtime internals.
func TestBuildMenu_PlacesCatalogItemsUnderCorrectTopLevels(t *testing.T) {
	app := NewApp()
	m := BuildMenu(app, nil)
	require.NotNil(t, m)

	// Build a map of top-level name → set of leaf labels present in that
	// submenu. EditMenu and WindowMenu roles add extra items we ignore.
	topLevels := map[string]map[string]bool{}
	for _, item := range m.Items {
		if item.Type != menu.SubmenuType || item.SubMenu == nil {
			continue
		}
		labels := map[string]bool{}
		collectLabels(item.SubMenu, labels)
		topLevels[item.Label] = labels
	}

	// Every catalog entry with a MenuPath that's visible on this GOOS
	// must show up under its declared top-level (collectLabels recurses
	// into sub-submenus so File > Export > PDF… still lands under "File").
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

// buildMenuForGOOS is deterministic regardless of the host OS. Prove
// the platform filter + sub-submenu logic works by invoking it for
// each of the three GOOS values we support and looking for specific
// items.
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

// SetDisabledCommands must produce a menu tree where the listed IDs
// have Disabled: true on their rendered item. Uses the same build path
// BuildMenu uses, so any real wiring break surfaces here.
func TestSetDisabledCommands_RebuildsMenuWithDisabledFlags(t *testing.T) {
	app := NewApp()
	initial := BuildMenu(app, nil)
	app.SetInitialMenu(initial)

	// Disable a couple of commands. Don't bother invoking the Wails
	// MenuUpdateApplicationMenu runtime hook (requires a live ctx); just
	// verify the rebuild produced the correct Disabled flags.
	require.NoError(t, app.SetDisabledCommands([]string{"file.exportPdf", "file.saveVersion"}))

	// Walk the rebuilt tree and find the matching labels under File,
	// recursing into sub-submenus so a disabled flag on a nested item
	// (File > Export > PDF…) surfaces.
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
	assert.True(t, disabledLabels["Save Version"], "Save Version should be Disabled")
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

// GetCommands returns the palette-facing projection. Should produce one
// entry per catalog command, in declaration order, with no Click
// callbacks or MenuPath leaking through.
func TestGetCommands_ReturnsPlatformFilteredCatalogProjection(t *testing.T) {
	app := NewApp()
	metas := app.GetCommands()

	// The result should mirror the catalog minus any entries filtered by
	// the current host's platform restriction. Count catalog commands
	// that allow the current GOOS and expect the same length.
	want := 0
	for _, cmd := range Commands() {
		if cmd.PlatformAllows(goruntime.GOOS) {
			want++
		}
	}
	assert.Equal(t, want, len(metas), "GetCommands must mirror platform-filtered Commands length")

	// Spot-check a few known IDs.
	ids := map[string]CommandMeta{}
	for _, m := range metas {
		ids[m.ID] = m
	}
	require.Contains(t, ids, CmdFileNewPlay)
	require.Contains(t, ids, CmdFileSettingsSpellcheck)

	assert.Equal(t, "New Play", ids[CmdFileNewPlay].Label)
	// Palette-hidden command surfaces its flag so the palette can filter.
	assert.True(t, ids[CmdFileSettingsSpellcheck].PaletteHidden)
	assert.False(t, ids[CmdFileNewPlay].PaletteHidden)

	// Platform restriction: Linux-only Undo is in the palette on Linux,
	// absent elsewhere. macOS-only Quit (via AppMenu role) is never in
	// the catalog Platforms-wise — our CmdFileQuit is Linux+Windows.
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
